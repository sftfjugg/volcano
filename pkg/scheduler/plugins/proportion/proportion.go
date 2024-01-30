/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proportion

import (
	"math"
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"volcano.sh/apis/pkg/apis/scheduling"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/api/helpers"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/metrics"
	"volcano.sh/volcano/pkg/scheduler/plugins/util"
)

// PluginName indicates name of volcano scheduler plugin.
const PluginName = "proportion"

type proportionPlugin struct {
	totalResource  *api.Resource
	totalGuarantee *api.Resource
	queueOpts      map[api.QueueID]*queueAttr
	// Arguments given for the plugin
	pluginArguments framework.Arguments
}

type queueAttr struct {
	queueID api.QueueID
	name    string
	weight  int32
	share   float64

	deserved  *api.Resource
	allocated *api.Resource
	request   *api.Resource
	// elastic represents the sum of job's elastic resource, job's elastic = job.allocated - job.minAvailable
	elastic *api.Resource
	// inqueue represents the resource request of the inqueue job
	inqueue    *api.Resource
	capability *api.Resource
	// realCapability represents the resource limit of the queue, LessEqual capability
	realCapability *api.Resource
	guarantee      *api.Resource
	// children represents the children of the queue
	children map[api.QueueID]*queueAttr
	// parrent represents the parrent of the queue
	parent api.QueueID
}

// New return proportion action
func New(arguments framework.Arguments) framework.Plugin {
	return &proportionPlugin{
		totalResource:   api.EmptyResource(),
		totalGuarantee:  api.EmptyResource(),
		queueOpts:       map[api.QueueID]*queueAttr{},
		pluginArguments: arguments,
	}
}

func (pp *proportionPlugin) Name() string {
	return PluginName
}

func (pp *proportionPlugin) OnSessionOpen(ssn *framework.Session) {
	// Prepare scheduling data for this session.
	pp.totalResource.Add(ssn.TotalResource)

	klog.V(4).Infof("The total resource is <%v>", pp.totalResource)
	for _, queue := range ssn.Queues {
		if len(queue.Queue.Spec.Guarantee.Resource) == 0 {
			continue
		}
		guarantee := api.NewResource(queue.Queue.Spec.Guarantee.Resource)
		pp.totalGuarantee.Add(guarantee)
	}
	klog.V(4).Infof("The total guarantee resource is <%v>", pp.totalGuarantee)
	// Build attributes for Queues.
	for _, job := range ssn.Jobs {
		klog.V(4).Infof("Considering Job <%s/%s>.", job.Namespace, job.Name)
		if _, found := pp.queueOpts[job.Queue]; !found {
			queue := ssn.Queues[job.Queue]
			attr := &queueAttr{
				queueID: queue.UID,
				name:    queue.Name,
				weight:  queue.Weight,

				deserved:  api.EmptyResource(),
				allocated: api.EmptyResource(),
				request:   api.EmptyResource(),
				elastic:   api.EmptyResource(),
				inqueue:   api.EmptyResource(),
				guarantee: api.EmptyResource(),
				children:  make(map[api.QueueID]*queueAttr),
			}
			if len(queue.Queue.Spec.Capability) != 0 {
				attr.capability = api.NewResource(queue.Queue.Spec.Capability)
				if attr.capability.MilliCPU <= 0 {
					attr.capability.MilliCPU = math.MaxFloat64
				}
				if attr.capability.Memory <= 0 {
					attr.capability.Memory = math.MaxFloat64
				}
			}
			if len(queue.Queue.Spec.Guarantee.Resource) != 0 {
				attr.guarantee = api.NewResource(queue.Queue.Spec.Guarantee.Resource)
			}
			realCapability := pp.totalResource.Clone().Sub(pp.totalGuarantee).Add(attr.guarantee)
			if attr.capability == nil {
				attr.realCapability = realCapability
			} else {
				attr.realCapability = helpers.Min(realCapability, attr.capability)
			}
			pp.queueOpts[job.Queue] = attr
			klog.V(4).Infof("Added Queue <%s> attributes.", job.Queue)
		}

		attr := pp.queueOpts[job.Queue]
		for status, tasks := range job.TaskStatusIndex {
			if api.AllocatedStatus(status) {
				for _, t := range tasks {
					attr.allocated.Add(t.Resreq)
					attr.request.Add(t.Resreq)
				}
			} else if status == api.Pending {
				for _, t := range tasks {
					attr.request.Add(t.Resreq)
				}
			}
		}

		if job.PodGroup.Status.Phase == scheduling.PodGroupInqueue {
			attr.inqueue.Add(job.GetMinResources())
		}

		// calculate inqueue resource for running jobs
		// the judgement 'job.PodGroup.Status.Running >= job.PodGroup.Spec.MinMember' will work on cases such as the following condition:
		// Considering a Spark job is completed(driver pod is completed) while the podgroup keeps running, the allocated resource will be reserved again if without the judgement.
		if job.PodGroup.Status.Phase == scheduling.PodGroupRunning &&
			job.PodGroup.Spec.MinResources != nil &&
			int32(util.CalculateAllocatedTaskNum(job)) >= job.PodGroup.Spec.MinMember {
			inqueued := util.GetInqueueResource(job, job.Allocated)
			attr.inqueue.Add(inqueued)
		}
		attr.elastic.Add(job.GetElasticResources())
		klog.V(5).Infof("Queue %s allocated <%s> request <%s> inqueue <%s> elastic <%s>",
			attr.name, attr.allocated.String(), attr.request.String(), attr.inqueue.String(), attr.elastic.String())
	}

	for queueID, queueInfo := range ssn.Queues {
		if _, ok := pp.queueOpts[queueID]; !ok {
			metrics.UpdateQueueAllocated(queueInfo.Name, 0, 0)
		}
	}
	// Temporary map to hold parent-child relationships
	tempChildren := make(map[string][]*queueAttr)

	for _, queue := range ssn.Queues {
		parentName := queue.Queue.Spec.Parent

		attr := pp.queueOpts[queue.UID]

		if parentName != "" {
			tempChildren[parentName] = append(tempChildren[parentName], attr)
		}
	}

	// Assign children to parent queues
	for _, queue := range ssn.Queues {
		queueName := queue.Name
		if children, exists := tempChildren[queueName]; exists {
			parentAttr := pp.queueOpts[queue.UID]
			for _, childAttr := range children {
				childAttr.parent = queue.UID
				parentAttr.children[childAttr.queueID] = childAttr
			}
		}
	}

	// Record metrics
	for _, attr := range pp.queueOpts {
		metrics.UpdateQueueAllocated(attr.name, attr.allocated.MilliCPU, attr.allocated.Memory)
		metrics.UpdateQueueRequest(attr.name, attr.request.MilliCPU, attr.request.Memory)
		metrics.UpdateQueueWeight(attr.name, attr.weight)
		queue := ssn.Queues[attr.queueID]
		metrics.UpdateQueuePodGroupInqueueCount(attr.name, queue.Queue.Status.Inqueue)
		metrics.UpdateQueuePodGroupPendingCount(attr.name, queue.Queue.Status.Pending)
		metrics.UpdateQueuePodGroupRunningCount(attr.name, queue.Queue.Status.Running)
		metrics.UpdateQueuePodGroupUnknownCount(attr.name, queue.Queue.Status.Unknown)
	}

	remaining := pp.totalResource.Clone()
	meet := map[api.QueueID]struct{}{}
	for {
		totalWeight := int32(0)
		for _, attr := range pp.queueOpts {
			if _, found := meet[attr.queueID]; found {
				continue
			}
			totalWeight += attr.weight
		}

		// If no queues, break
		if totalWeight == 0 {
			klog.V(4).Infof("Exiting when total weight is 0")
			break
		}

		oldRemaining := remaining.Clone()
		// Calculates the deserved of each Queue.
		// increasedDeserved is the increased value for attr.deserved of processed queues
		// decreasedDeserved is the decreased value for attr.deserved of processed queues
		increasedDeserved := api.EmptyResource()
		decreasedDeserved := api.EmptyResource()
		for _, attr := range pp.queueOpts {
			klog.V(4).Infof("Considering Queue <%s>: weight <%d>, total weight <%d>.",
				attr.name, attr.weight, totalWeight)
			if _, found := meet[attr.queueID]; found {
				continue
			}

			oldDeserved := attr.deserved.Clone()
			attr.deserved.Add(remaining.Clone().Multi(float64(attr.weight) / float64(totalWeight)))

			if attr.realCapability != nil {
				attr.deserved.MinDimensionResource(attr.realCapability, api.Infinity)
			}
			attr.deserved.MinDimensionResource(attr.request, api.Zero)

			attr.deserved = helpers.Max(attr.deserved, attr.guarantee)
			pp.updateShare(attr)
			pp.updateParentQueue(attr)
			klog.V(4).Infof("Format queue <%s> deserved resource to <%v>", attr.name, attr.deserved)

			if attr.request.LessEqual(attr.deserved, api.Zero) {
				meet[attr.queueID] = struct{}{}
				klog.V(4).Infof("queue <%s> is meet", attr.name)
			} else if reflect.DeepEqual(attr.deserved, oldDeserved) {
				meet[attr.queueID] = struct{}{}
				klog.V(4).Infof("queue <%s> is meet cause of the capability", attr.name)
			}

			klog.V(4).Infof("The attributes of queue <%s> in proportion: deserved <%v>, realCapability <%v>, allocate <%v>, request <%v>, elastic <%v>, share <%0.2f>",
				attr.name, attr.deserved, attr.realCapability, attr.allocated, attr.request, attr.elastic, attr.share)

			increased, decreased := attr.deserved.Diff(oldDeserved, api.Zero)
			increasedDeserved.Add(increased)
			decreasedDeserved.Add(decreased)

			// Record metrics
			metrics.UpdateQueueDeserved(attr.name, attr.deserved.MilliCPU, attr.deserved.Memory)
		}

		remaining.Sub(increasedDeserved).Add(decreasedDeserved)
		klog.V(4).Infof("Remaining resource is  <%s>", remaining)
		if remaining.IsEmpty() || reflect.DeepEqual(remaining, oldRemaining) {
			klog.V(4).Infof("Exiting when remaining is empty or no queue has more resource request:  <%v>", remaining)
			break
		}
	}

	ssn.AddQueueOrderFn(pp.Name(), func(l, r interface{}) int {
		lv := l.(*api.QueueInfo)
		rv := r.(*api.QueueInfo)

		// Compare leaf queues
		if pp.isLeafNode(lv.UID) && pp.isLeafNode(rv.UID) {
			// Find the parent of the leaf nodes
			lvParent := pp.getParentQueue(lv.UID)
			rvParent := pp.getParentQueue(rv.UID)
			// If both leaf nodes have the same parent
			if lvParent == rvParent {
				// Compare the weight of the leaf nodes
				if pp.queueOpts[lv.UID].weight != pp.queueOpts[rv.UID].weight {
					return int(pp.queueOpts[rv.UID].weight - pp.queueOpts[lv.UID].weight)
				}
				// Compare the share of the leaf nodes
				if pp.queueOpts[lv.UID].share != pp.queueOpts[rv.UID].share {
					if pp.queueOpts[lv.UID].share < pp.queueOpts[rv.UID].share {
						return -1
					}
					return 1
				}
				return 0
			}

			// Find the level of the leaf nodes
			lvLevel := pp.getQueueLevel(lv.UID)
			rvLevel := pp.getQueueLevel(rv.UID)

			// If both leaf nodes are at the same level
			if lvLevel == rvLevel {
				// Compare the parent queues of the leaf nodes
				lvParentQueue := pp.getParentQueue(lvParent)
				rvParentQueue := pp.getParentQueue(rvParent)

				// Compare the weight of the parent queues
				if pp.queueOpts[lvParentQueue].weight != pp.queueOpts[rvParentQueue].weight {
					return int(pp.queueOpts[rvParentQueue].weight - pp.queueOpts[lvParentQueue].weight)
				}
				// Compare the share of the parent queues
				if pp.queueOpts[lvParentQueue].share != pp.queueOpts[rvParentQueue].share {
					if pp.queueOpts[lvParentQueue].share < pp.queueOpts[rvParentQueue].share {
						return -1
					}
					return 1
				}
				return 0
			}

			// Find the lowest common ancestor (LCA) of the leaf nodes
			lca := pp.lowestCommonAncestor(lv.UID, rv.UID)

			// Compare each leaf node with its parent queue at the level of the LCA
			lvParentQueue := pp.getParentQueueBelowLCA(lv.UID, lca)
			rvParentQueue := pp.getParentQueueBelowLCA(rv.UID, lca)

			// Compare the weight of the parent queues
			if pp.queueOpts[lvParentQueue].weight != pp.queueOpts[rvParentQueue].weight {
				return int(pp.queueOpts[rvParentQueue].weight - pp.queueOpts[lvParentQueue].weight)
			}
			// Compare the share of the parent queues
			if pp.queueOpts[lvParentQueue].share != pp.queueOpts[rvParentQueue].share {
				if pp.queueOpts[lvParentQueue].share < pp.queueOpts[rvParentQueue].share {
					return -1
				}
				return 1
			}
			return 0
		}

		// Compare leaf and non-leaf queues, non-leaf queues are always at the end of the queue
		if pp.isLeafNode(lv.UID) && !pp.isLeafNode(rv.UID) {
			return -1
		}
		if !pp.isLeafNode(lv.UID) && pp.isLeafNode(rv.UID) {
			return 1
		}

		// Compare non-leaf queues
		if pp.queueOpts[lv.UID].weight != pp.queueOpts[rv.UID].weight {
			return int(pp.queueOpts[rv.UID].weight - pp.queueOpts[lv.UID].weight)
		}
		if pp.queueOpts[lv.UID].share == pp.queueOpts[rv.UID].share {
			return 0
		}
		if pp.queueOpts[lv.UID].share < pp.queueOpts[rv.UID].share {
			return -1
		}
		return 1
	})

	ssn.AddReclaimableFn(pp.Name(), func(reclaimer *api.TaskInfo, reclaimees []*api.TaskInfo) ([]*api.TaskInfo, int) {
		var victims []*api.TaskInfo
		allocations := map[api.QueueID]*api.Resource{}
		for _, reclaimee := range reclaimees {
			job := ssn.Jobs[reclaimee.Job]
			attr := pp.queueOpts[job.Queue]
			if _, found := allocations[job.Queue]; !found {
				allocations[job.Queue] = attr.allocated.Clone()
			}
			allocated := allocations[job.Queue]
			if allocated.LessPartly(reclaimer.Resreq, api.Zero) {
				klog.V(3).Infof("Failed to allocate resource for Task <%s/%s> in Queue <%s>, not enough resource.",
					reclaimee.Namespace, reclaimee.Name, job.Queue)
				continue
			}

			if !allocated.LessEqual(attr.deserved, api.Zero) {
				allocated.Sub(reclaimee.Resreq)
				victims = append(victims, reclaimee)
			}
		}
		klog.V(4).Infof("Victims from proportion plugins are %+v", victims)
		return victims, util.Permit
	})

	ssn.AddOverusedFn(pp.Name(), func(obj interface{}) bool {
		queue := obj.(*api.QueueInfo)
		attr := pp.queueOpts[queue.UID]

		overused := attr.deserved.LessEqual(attr.allocated, api.Zero)
		metrics.UpdateQueueOverused(attr.name, overused)
		if overused {
			klog.V(3).Infof("Queue <%v>: deserved <%v>, allocated <%v>, share <%v>",
				queue.Name, attr.deserved, attr.allocated, attr.share)
			// Check if parent queue is overusing
			if parent := attr.parent; parent != "" {
				parentAttr := pp.queueOpts[parent]
				parentOverused := parentAttr.deserved.LessEqual(parentAttr.allocated, api.Zero)
				if !parentOverused {
					parentAttr.deserved = parentAttr.deserved.Add(attr.deserved)
					parentAttr.allocated = parentAttr.allocated.Add(attr.allocated)
					parentOverused = true
					metrics.UpdateQueueOverused(parentAttr.name, parentOverused)
					klog.V(3).Infof("Parent Queue <%v>: deserved <%v>, allocated <%v>, share <%v>",
						parentAttr.name, parentAttr.deserved, parentAttr.allocated, parentAttr.share)
				}
			}
		}

		return overused
	})

	ssn.AddAllocatableFn(pp.Name(), func(queue *api.QueueInfo, candidate *api.TaskInfo) bool {
		attr := pp.queueOpts[queue.UID]
		if !pp.isLeafNode(queue.UID) {
			return false
			//todo: add logic for non-leaf node, allocate to the leaf node
		}
		free, _ := attr.deserved.Diff(attr.allocated, api.Zero)
		allocatable := candidate.Resreq.LessEqual(free, api.Zero)
		if !allocatable {
			klog.V(3).Infof("Queue <%v>: deserved <%v>, allocated <%v>; Candidate <%v>: resource request <%v>",
				queue.Name, attr.deserved, attr.allocated, candidate.Name, candidate.Resreq)
		}

		return allocatable
	})

	ssn.AddJobEnqueueableFn(pp.Name(), func(obj interface{}) int {
		job := obj.(*api.JobInfo)
		queueID := job.Queue
		attr := pp.queueOpts[queueID]
		queue := ssn.Queues[queueID]
		// If no capability is set, always enqueue the job.
		if attr.realCapability == nil {
			klog.V(4).Infof("Capability of queue <%s> was not set, allow job <%s/%s> to Inqueue.",
				queue.Name, job.Namespace, job.Name)
			return util.Permit
		}

		if job.PodGroup.Spec.MinResources == nil {
			klog.V(4).Infof("job %s MinResources is null.", job.Name)
			return util.Permit
		}
		minReq := job.GetMinResources()

		klog.V(5).Infof("job %s min resource <%s>, queue %s capability <%s> allocated <%s> inqueue <%s> elastic <%s>",
			job.Name, minReq.String(), queue.Name, attr.realCapability.String(), attr.allocated.String(), attr.inqueue.String(), attr.elastic.String())
		// The queue resource quota limit has not reached
		r := minReq.Add(attr.allocated).Add(attr.inqueue).Sub(attr.elastic)
		rr := attr.realCapability.Clone()

		for name := range rr.ScalarResources {
			if _, ok := r.ScalarResources[name]; !ok {
				delete(rr.ScalarResources, name)
			}
		}

		inqueue := r.LessEqual(rr, api.Infinity)
		klog.V(5).Infof("job %s inqueue %v", job.Name, inqueue)
		if inqueue {
			attr.inqueue.Add(job.GetMinResources())
			return util.Permit
		}
		ssn.RecordPodGroupEvent(job.PodGroup, v1.EventTypeNormal, string(scheduling.PodGroupUnschedulableType), "queue resource quota insufficient")
		return util.Reject
	})

	// Register event handlers.
	ssn.AddEventHandler(&framework.EventHandler{
		AllocateFunc: func(event *framework.Event) {
			job := ssn.Jobs[event.Task.Job]
			attr := pp.queueOpts[job.Queue]
			attr.allocated.Add(event.Task.Resreq)
			metrics.UpdateQueueAllocated(attr.name, attr.allocated.MilliCPU, attr.allocated.Memory)

			pp.updateShare(attr)
			pp.updateParentQueue(attr)
			klog.V(4).Infof("Proportion AllocateFunc: task <%v/%v>, resreq <%v>,  share <%v>",
				event.Task.Namespace, event.Task.Name, event.Task.Resreq, attr.share)
		},
		DeallocateFunc: func(event *framework.Event) {
			job := ssn.Jobs[event.Task.Job]
			attr := pp.queueOpts[job.Queue]
			attr.allocated.Sub(event.Task.Resreq)
			metrics.UpdateQueueAllocated(attr.name, attr.allocated.MilliCPU, attr.allocated.Memory)

			pp.updateShare(attr)
			pp.updateParentQueue(attr)
			klog.V(4).Infof("Proportion EvictFunc: task <%v/%v>, resreq <%v>,  share <%v>",
				event.Task.Namespace, event.Task.Name, event.Task.Resreq, attr.share)
		},
	})
}

func (pp *proportionPlugin) OnSessionClose(ssn *framework.Session) {
	pp.totalResource = nil
	pp.totalGuarantee = nil
	pp.queueOpts = nil
}

func (pp *proportionPlugin) updateShare(attr *queueAttr) {
	res := float64(0)

	// TODO(k82cn): how to handle fragment issues?
	for _, rn := range attr.deserved.ResourceNames() {
		share := helpers.Share(attr.allocated.Get(rn), attr.deserved.Get(rn))
		if share > res {
			res = share
		}
	}

	attr.share = res
	metrics.UpdateQueueShare(attr.name, attr.share)
}

// determine whether the queue is a leaf node
func (pp *proportionPlugin) isLeafNode(queueID api.QueueID) bool {
	attr := pp.queueOpts[queueID]
	if len(attr.children) == 0 {
		return true
	}
	return false
}

// get the parent queue ID
func (pp *proportionPlugin) getParentQueue(queueID api.QueueID) api.QueueID {
	attr := pp.queueOpts[queueID]
	return attr.parent
}

// get the level of the queue
func (pp *proportionPlugin) getQueueLevel(queueID api.QueueID) int {
	attr := pp.queueOpts[queueID]
	level := 0
	for attr.parent != "" {
		level++
		attr = pp.queueOpts[attr.parent]
	}
	return level
}

// get the parent queue ID one level below the lowest common ancestor (LCA)
func (pp *proportionPlugin) getParentQueueBelowLCA(queueID api.QueueID, lca api.QueueID) api.QueueID {
	attr := pp.queueOpts[queueID]

	// If the queue is one level below, return itself
	if attr.parent == lca {
		return queueID
	}
	// Otherwise, find the parent queue ID one level below the LCA
	var lastParent api.QueueID = queueID
	for attr.parent != lca {
		lastParent = attr.parent
		attr = pp.queueOpts[attr.parent]
	}
	return lastParent
}

// get the lowest common ancestor (LCA) of the two queues
func (pp *proportionPlugin) lowestCommonAncestor(queueID1 api.QueueID, queueID2 api.QueueID) api.QueueID {
	attr1 := pp.queueOpts[queueID1]
	attr2 := pp.queueOpts[queueID2]

	// find the level of the two queues
	level1 := pp.getQueueLevel(queueID1)
	level2 := pp.getQueueLevel(queueID2)

	// equalize the level of the two queues
	for level1 > level2 {
		attr1 = pp.queueOpts[attr1.parent]
		level1--
	}
	for level2 > level1 {
		attr2 = pp.queueOpts[attr2.parent]
		level2--
	}

	for attr1.parent != attr2.parent {
		attr1 = pp.queueOpts[attr1.parent]
		attr2 = pp.queueOpts[attr2.parent]
	}
	return attr1.parent
}

// Updates the resource state of parent queues based on the child queues' states.
func (pp *proportionPlugin) updateParentQueue(attr *queueAttr) {
	// find the child queue, return if not exists
	childAttr, exists := pp.queueOpts[attr.queueID]
	if !exists {
		return
	}

	// get the parent queue ID
	parentQueueID := childAttr.parent
	if parentQueueID == "" {
		return
	}
	// update the parent queue
	if parentAttr, exists := pp.queueOpts[parentQueueID]; exists {
		// initialize the parent queue's resource state
		totalAllocated := api.EmptyResource()
		totalRequested := api.EmptyResource()
		totalGuarantee := api.EmptyResource()

		// add the resource state of all the children queues
		for _, child := range parentAttr.children {
			totalAllocated.Add(child.allocated)
			totalRequested.Add(child.request)
			totalGuarantee.Add(child.guarantee)
		}

		// update the parent queue's resource state
		parentAttr.allocated = totalAllocated
		parentAttr.request = totalRequested
		parentAttr.guarantee = totalGuarantee
		pp.updateShare(parentAttr)
		// call itself recursively to get the most top parent queue
		pp.updateParentQueue(parentAttr)
	}
}

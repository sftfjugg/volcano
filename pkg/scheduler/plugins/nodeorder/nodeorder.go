/*
Copyright 2019 The Kubernetes Authors.

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

package nodeorder

import (
	"context"

	v1 "k8s.io/api/core/v1"
	utilFeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/features"
	k8sframework "k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"

	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/plugins/util"
	"volcano.sh/volcano/pkg/scheduler/plugins/util/k8s"
)

const (
	// PluginName indicates name of volcano scheduler plugin.
	PluginName = "nodeorder"

	// NodeAffinityWeight is the key for providing Node Affinity Priority Weight in YAML
	NodeAffinityWeight = "nodeaffinity.weight"
	// PodAffinityWeight is the key for providing Pod Affinity Priority Weight in YAML
	PodAffinityWeight = "podaffinity.weight"
	// LeastRequestedWeight is the key for providing Least Requested Priority Weight in YAML
	LeastRequestedWeight = "leastrequested.weight"
	// BalancedResourceWeight is the key for providing Balanced Resource Priority Weight in YAML
	BalancedResourceWeight = "balancedresource.weight"
	// MostRequestedWeight is the key for providing Most Requested Priority Weight in YAML
	MostRequestedWeight = "mostrequested.weight"
	// TaintTolerationWeight is the key for providing Taint Toleration Priority Weight in YAML
	TaintTolerationWeight = "tainttoleration.weight"
	// ImageLocalityWeight is the key for providing Image Locality Priority Weight in YAML
	ImageLocalityWeight = "imagelocality.weight"
	// PodTopologySpreadWeight is the key for providing Pod Topology Spread Priority Weight in YAML
	PodTopologySpreadWeight = "podtopologyspread.weight"
	// SelectorSpreadWeight is the key for providing Selector Spread Priority Weight in YAML
	selectorSpreadWeight = "selectorspread.weight"
	// VolumeBinding is the key for providing Volume Binding Priority Weight in YAML
	volumeBindingWeight = "volumebinding.weight"
)

type nodeOrderPlugin struct {
	// Arguments given for the plugin
	pluginArguments framework.Arguments
}

// New function returns nodeorder plugin object.
func New(arguments framework.Arguments) framework.Plugin {
	return &nodeOrderPlugin{pluginArguments: arguments}
}

func (pp *nodeOrderPlugin) Name() string {
	return PluginName
}

type priorityWeight struct {
	leastReqWeight          int
	mostReqWeight           int
	nodeAffinityWeight      int
	podAffinityWeight       int
	balancedResourceWeight  int
	taintTolerationWeight   int
	imageLocalityWeight     int
	podTopologySpreadWeight int
	selectorSpreadWeight    int
	volumeBindingWeight     int
}

// calculateWeight from the provided arguments.
//
// Currently only supported priorities are nodeaffinity, podaffinity, leastrequested,
// mostrequested, balancedresouce, imagelocality, tainttoleration, podtopologySpread.
//
// User should specify priority weights in the config in this format:
//
//	actions: "reclaim, allocate, backfill, preempt"
//	tiers:
//	- plugins:
//	  - name: priority
//	  - name: gang
//	  - name: conformance
//	- plugins:
//	  - name: drf
//	  - name: predicates
//	  - name: proportion
//	  - name: nodeorder
//	    arguments:
//	      leastrequested.weight: 1
//	      mostrequested.weight: 0
//	      nodeaffinity.weight: 1
//	      podaffinity.weight: 1
//	      balancedresource.weight: 1
//	      tainttoleration.weight: 1
//	      imagelocality.weight: 1
//	      podtopologyspread.weight: 2
func calculateWeight(args framework.Arguments) priorityWeight {
	// Initial values for weights.
	// By default, for backward compatibility and for reasonable scores,
	// least requested priority is enabled and most requested priority is disabled.
	weight := priorityWeight{
		leastReqWeight:          1,
		mostReqWeight:           0,
		nodeAffinityWeight:      1,
		podAffinityWeight:       1,
		balancedResourceWeight:  1,
		taintTolerationWeight:   1,
		imageLocalityWeight:     1,
		podTopologySpreadWeight: 2, // be consistent with kubernetes default setting.
		selectorSpreadWeight:    0,
		volumeBindingWeight:     1,
	}

	// Checks whether nodeaffinity.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.nodeAffinityWeight, NodeAffinityWeight)

	// Checks whether podaffinity.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.podAffinityWeight, PodAffinityWeight)

	// Checks whether leastrequested.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.leastReqWeight, LeastRequestedWeight)

	// Checks whether mostrequested.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.mostReqWeight, MostRequestedWeight)

	// Checks whether balancedresource.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.balancedResourceWeight, BalancedResourceWeight)

	// Checks whether tainttoleration.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.taintTolerationWeight, TaintTolerationWeight)

	// Checks whether imagelocality.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.imageLocalityWeight, ImageLocalityWeight)

	// Checks whether podtopologyspread.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.podTopologySpreadWeight, PodTopologySpreadWeight)

	// Checks whether selectorspread.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.selectorSpreadWeight, selectorSpreadWeight)

	// Checks whether volumebinding.weight is provided or not, if given, modifies the value in weight struct.
	args.GetInt(&weight.volumeBindingWeight, volumeBindingWeight)

	return weight
}

func (pp *nodeOrderPlugin) OnSessionOpen(ssn *framework.Session) {
	weight := calculateWeight(pp.pluginArguments)
	pl := util.NewPodListerFromNode(ssn)
	nodeMap := util.GenerateNodeMapAndSlice(ssn.Nodes)

	// Register event handlers to update task info in PodLister & nodeMap
	ssn.AddEventHandler(&framework.EventHandler{
		AllocateFunc: func(event *framework.Event) {
			pod := pl.UpdateTask(event.Task, event.Task.NodeName)

			nodeName := event.Task.NodeName
			node, found := nodeMap[nodeName]
			if !found {
				klog.Warningf("node order, update pod %s/%s allocate to NOT EXIST node [%s]", pod.Namespace, pod.Name, nodeName)
			} else {
				node.AddPod(pod)
				klog.V(4).Infof("node order, update pod %s/%s allocate to node [%s]", pod.Namespace, pod.Name, nodeName)
			}
		},
		DeallocateFunc: func(event *framework.Event) {
			pod := pl.UpdateTask(event.Task, "")

			nodeName := event.Task.NodeName
			node, found := nodeMap[nodeName]
			if !found {
				klog.Warningf("node order, update pod %s/%s allocate from NOT EXIST node [%s]", pod.Namespace, pod.Name, nodeName)
			} else {
				err := node.RemovePod(pod)
				if err != nil {
					klog.Errorf("Failed to update pod %s/%s and deallocate from node [%s]: %s", pod.Namespace, pod.Name, nodeName, err.Error())
				} else {
					klog.V(4).Infof("node order, update pod %s/%s deallocate from node [%s]", pod.Namespace, pod.Name, nodeName)
				}
			}
		},
	})

	fts := feature.Features{
		EnablePodAffinityNamespaceSelector: utilFeature.DefaultFeatureGate.Enabled(features.PodAffinityNamespaceSelector),
		EnablePodDisruptionBudget:          utilFeature.DefaultFeatureGate.Enabled(features.PodDisruptionBudget),
		EnablePodOverhead:                  utilFeature.DefaultFeatureGate.Enabled(features.PodOverhead),
		EnableReadWriteOncePod:             utilFeature.DefaultFeatureGate.Enabled(features.ReadWriteOncePod),
		EnableVolumeCapacityPriority:       utilFeature.DefaultFeatureGate.Enabled(features.VolumeCapacityPriority),
		EnableCSIStorageCapacity:           utilFeature.DefaultFeatureGate.Enabled(features.CSIStorageCapacity),
	}

	// Initialize k8s scheduling plugins
	handle := k8s.NewFrameworkHandle(nodeMap, ssn.KubeClient(), ssn.InformerFactory())

	// NodeResources plugin with LeastAllocated strategy
	leastAllocated, _ := leastAllocated(handle, fts)
	// NodeResources plugin with MostAllocated strategy
	mostAllocation, _ := mostAllocated(handle, fts)
	// NodeResources plugin with BalancedAllocation strategy
	balancedAllocation, _ := balancedAllocated(handle, fts)
	// NodeAffinity plugin
	nodeAffinity, _ := nodeAffinity(handle)
	// ImageLocality plugin
	imageLocality, _ := imageLocality(handle)
	// VolumeBinding plugin
	volumeBinding, _ := volumeBinding(handle, fts)

	nodeOrderFn := func(task *api.TaskInfo, node *api.NodeInfo) (float64, error) {
		var nodeScore = 0.0

		state := k8sframework.NewCycleState()
		if weight.imageLocalityWeight != 0 {
			score, status := imageLocality.Score(context.TODO(), state, task.Pod, node.Name)
			if !status.IsSuccess() {
				klog.Warningf("Image Locality Priority Failed because of Error: %v", status.AsError())
				return 0, status.AsError()
			}

			// If imageLocalityWeight is provided, host.Score is multiplied with weight, if not, host.Score is added to total score.
			nodeScore += float64(score) * float64(weight.imageLocalityWeight)
			klog.V(4).Infof("Image Locality score: %f", nodeScore)
		}

		// NodeResourcesLeastAllocated
		if weight.leastReqWeight != 0 {
			score, status := leastAllocated.Score(context.TODO(), state, task.Pod, node.Name)
			if !status.IsSuccess() {
				klog.Warningf("Least Allocated Priority Failed because of Error: %v", status.AsError())
				return 0, status.AsError()
			}

			// If leastReqWeight is provided, host.Score is multiplied with weight, if not, host.Score is added to total score.
			nodeScore += float64(score) * float64(weight.leastReqWeight)
			klog.V(4).Infof("Least Request score: %f", nodeScore)
		}

		// NodeResourcesMostAllocated
		if weight.mostReqWeight != 0 {
			score, status := mostAllocation.Score(context.TODO(), state, task.Pod, node.Name)
			if !status.IsSuccess() {
				klog.Warningf("Most Allocated Priority Failed because of Error: %v", status.AsError())
				return 0, status.AsError()
			}

			// If mostRequestedWeight is provided, host.Score is multiplied with weight, it's 0 by default
			nodeScore += float64(score) * float64(weight.mostReqWeight)
			klog.V(4).Infof("Most Request score: %f", nodeScore)
		}

		// NodeResourcesBalancedAllocation
		if weight.balancedResourceWeight != 0 {
			score, status := balancedAllocation.Score(context.TODO(), state, task.Pod, node.Name)
			if !status.IsSuccess() {
				klog.Warningf("Balanced Resource Allocation Priority Failed because of Error: %v", status.AsError())
				return 0, status.AsError()
			}

			// If balancedResourceWeight is provided, host.Score is multiplied with weight, if not, host.Score is added to total score.
			nodeScore += float64(score) * float64(weight.balancedResourceWeight)
			klog.V(4).Infof("Balanced Request score: %f", nodeScore)
		}

		// NodeAffinity
		if weight.nodeAffinityWeight != 0 {
			score, status := nodeAffinity.Score(context.TODO(), state, task.Pod, node.Name)
			if !status.IsSuccess() {
				klog.Warningf("Calculate Node Affinity Priority Failed because of Error: %v", status.AsError())
				return 0, status.AsError()
			}

			// TODO: should we normalize the score
			// If nodeAffinityWeight is provided, host.Score is multiplied with weight, if not, host.Score is added to total score.
			nodeScore += float64(score) * float64(weight.nodeAffinityWeight)
			klog.V(4).Infof("Node Affinity score: %f", nodeScore)
		}

		// VolumeBinding
		if weight.volumeBindingWeight != 0 {
			score, status := volumeBinding.Score(context.TODO(), state, task.Pod, node.Name)
			if !status.IsSuccess() {
				klog.Warningf("Volume Binding Priority Failed because of Error: %v", status.AsError())
				return 0, status.AsError()
			}

			// If volumeBindingWeight is provided, host.Score is multiplied with weight, if not, host.Score is added to total score.
			nodeScore += float64(score) * float64(weight.volumeBindingWeight)
			klog.V(4).Infof("Volume Binding score: %f", nodeScore)
		}

		klog.V(4).Infof("Total Score for task %s/%s on node %s is: %f", task.Namespace, task.Name, node.Name, nodeScore)
		return nodeScore, nil
	}
	ssn.AddNodeOrderFn(pp.Name(), nodeOrderFn)

	// InterPodAffinity plugin
	interPodAffinity, _ := interPodAffinity(handle, fts)
	// TaintToleration plugin
	taintToleration, _ := taintToleration(handle)
	// PodTopologySpread plugin
	podTopologySpread, _ := podTopologySpread(handle, fts)
	// selectorSpread plugin
	selectorSpread, _ := selectorSpread(handle)

	batchNodeOrderFn := func(task *api.TaskInfo, nodeInfo []*api.NodeInfo) (map[string]float64, error) {
		// InterPodAffinity
		state := k8sframework.NewCycleState()
		nodes := make([]*v1.Node, 0, len(nodeInfo))
		for _, node := range nodeInfo {
			nodes = append(nodes, node.Node)
		}
		nodeScores := make(map[string]float64, len(nodes))

		podAffinityScores, podErr := interPodAffinityScore(interPodAffinity, state, task.Pod, nodes, weight.podAffinityWeight)
		if podErr != nil {
			return nil, podErr
		}

		nodeTolerationScores, err := taintTolerationScore(taintToleration, state, task.Pod, nodes, weight.taintTolerationWeight)
		if err != nil {
			return nil, err
		}

		podTopologySpreadScores, err := podTopologySpreadScore(podTopologySpread, state, task.Pod, nodes, weight.podTopologySpreadWeight)
		if err != nil {
			return nil, err
		}

		selectorSpreadScores, err := selectorSpreadScore(selectorSpread, state, task.Pod, nodes, weight.selectorSpreadWeight)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			nodeScores[node.Name] = podAffinityScores[node.Name] + nodeTolerationScores[node.Name] + podTopologySpreadScores[node.Name] + selectorSpreadScores[node.Name]
		}

		klog.V(4).Infof("Batch Total Score for task %s/%s is: %v", task.Namespace, task.Name, nodeScores)
		return nodeScores, nil
	}
	ssn.AddBatchNodeOrderFn(pp.Name(), batchNodeOrderFn)
}

func (pp *nodeOrderPlugin) OnSessionClose(ssn *framework.Session) {
}

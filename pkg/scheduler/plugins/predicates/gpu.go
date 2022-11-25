/*
Copyright 2020 The Kubernetes Authors.

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

package predicates

import (
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"volcano.sh/volcano/pkg/scheduler/api"
)

// checkNodeGPUSharingPredicate checks if a pod with gpu requirement can be scheduled on a node.
func checkNodeGPUSharingPredicate(pod *v1.Pod, nodeInfo *api.NodeInfo) (bool, error) {
	// no gpu sharing request
	if api.GetGPUMemoryOfPod(pod) <= 0 {
		return true, nil
	}
	ids := predicateGPUbyMemory(pod, nodeInfo)
	if len(ids) == 0 {
		return false, fmt.Errorf("no enough gpu memory on node %s", nodeInfo.Name)
	}
	return true, nil
}

func checkNodeGPUNumberPredicate(pod *v1.Pod, nodeInfo *api.NodeInfo) (bool, error) {
	//no gpu number request
	if api.GetGPUNumberOfPod(pod) <= 0 {
		return true, nil
	}
	ids := predicateGPUbyNumber(pod, nodeInfo)
	if len(ids) == 0 {
		return false, fmt.Errorf("no enough gpu number on node %s", nodeInfo.Name)
	}
	return true, nil
}

// predicateGPUbyMemory returns the available GPU ID
func predicateGPUbyMemory(pod *v1.Pod, node *api.NodeInfo) []int {
	gpuRequest := api.GetGPUMemoryOfPod(pod)
	allocatableGPUs := node.GetDevicesIdleGPUMemory()

	var devIDs []int

	for devID := range allocatableGPUs {
		if availableGPU, ok := allocatableGPUs[devID]; ok && availableGPU >= gpuRequest {
			devIDs = append(devIDs, devID)
		}
	}
	sort.Ints(devIDs)
	return devIDs
}

// predicateGPU returns the available GPU IDs
func predicateGPUbyNumber(pod *v1.Pod, node *api.NodeInfo) []int {
	gpuRequest := api.GetGPUNumberOfPod(pod)
	allocatableGPUs := node.GetDevicesIdleGPUs()

	if len(allocatableGPUs) < gpuRequest {
		klog.Errorf("Not enough gpu cards")
		return nil
	}

	return allocatableGPUs[:gpuRequest]
}

/*
Copyright 2022 The Volcano Authors.

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

package rescheduling

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"

	"volcano.sh/volcano/pkg/scheduler/api"
)

// DefaultLowNodeConf defines the default configuration for LNU strategy
var DefaultLowNodeConf = map[string]interface{}{
	"thresholds":                 map[string]float64{"cpu": 100, "memory": 100, "pods": 100},
	"targetThresholds":           map[string]float64{"cpu": 100, "memory": 100, "pods": 100},
	"thresholdPriorityClassName": "system-cluster-critical",
	"nodeFit":                    true,
}

type LowNodeUtilizationConf struct {
	Thresholds                 map[string]float64
	TargetThresholds           map[string]float64
	NumberOfNodes              int
	ThresholdPriority          int
	ThresholdPriorityClassName string
	NodeFit                    bool
}

// NewLowNodeUtilizationConf returns the pointer of LowNodeUtilizationConf object with default value
func NewLowNodeUtilizationConf() *LowNodeUtilizationConf {
	return &LowNodeUtilizationConf{
		Thresholds:                 map[string]float64{"cpu": 100, "memory": 100, "pods": 100},
		TargetThresholds:           map[string]float64{"cpu": 100, "memory": 100, "pods": 100},
		ThresholdPriorityClassName: "system-cluster-critical",
		NodeFit:                    true,
	}
}

// parse converts the config map to struct object
func (lnuc *LowNodeUtilizationConf) parse(configs map[string]interface{}) {
	if len(configs) == 0 {
		return
	}
	lowThresholdsConfigs, ok := configs["thresholds"]
	if ok {
		lowConfigs, _ := lowThresholdsConfigs.(map[string]int)
		parseThreshold(lowConfigs, lnuc, "Thresholds")
	}
	targetThresholdsConfigs, ok := configs["targetThresholds"]
	if ok {
		targetConfigs, _ := targetThresholdsConfigs.(map[string]int)
		parseThreshold(targetConfigs, lnuc, "TargetThresholds")
	}
}

func parseThreshold(thresholdsConfig map[string]int, lnuc *LowNodeUtilizationConf, param string) {
	if len(thresholdsConfig) > 0 {
		configValue := reflect.ValueOf(lnuc).Elem().FieldByName(param)
		config := configValue.Interface().(map[string]float64)

		cpuThreshold, ok := thresholdsConfig["cpu"]
		if ok {
			config["cpu"] = float64(cpuThreshold)
		}
		memoryThreshold, ok := thresholdsConfig["memory"]
		if ok {
			config["memory"] = float64(memoryThreshold)
		}
		podThreshold, ok := thresholdsConfig["pod"]
		if ok {
			config["pod"] = float64(podThreshold)
		}
	}
}

var victimsFnForLnu = func(tasks []*api.TaskInfo) []*api.TaskInfo {
	victims := make([]*api.TaskInfo, 0)

	// parse configuration arguments
	utilizationConfig := NewLowNodeUtilizationConf()
	parametersConfig := RegisteredStrategyConfigs["lowNodeUtilization"]
	var config map[string]interface{}
	config, ok := parametersConfig.(map[string]interface{})
	if !ok {
		klog.Error("parameters parse error for lowNodeUtilization")
		return victims
	}
	utilizationConfig.parse(config)
	klog.V(4).Infof("The configuration for lowNodeUtilization: %v", *utilizationConfig)

	// group the nodes into lowNodes and highNodes
	nodeUtilizationList := getNodeUtilization()
	klog.V(4).Infoln("The nodeUtilizationList:")
	for _, nodeUtilization := range nodeUtilizationList {
		klog.V(4).Infof("node: %s, utilization: %s \n", nodeUtilization.nodeInfo.Name, nodeUtilization.utilization)
		for _, pod := range nodeUtilization.pods {
			klog.V(4).Infof("pod: %s \n", pod.Name)
		}
	}

	lowNodes, highNodes := groupNodesByUtilization(nodeUtilizationList, lowThresholdFilter, highThresholdFilter, *utilizationConfig)
	klog.V(4).Infoln("The low nodes:")
	for _, node := range lowNodes {
		klog.V(4).Infoln(node.nodeInfo.Name)
	}
	klog.V(4).Infoln("The high nodes:")
	for _, node := range highNodes {
		klog.V(4).Infoln(node.nodeInfo.Name)
	}
	if len(lowNodes) == 0 {
		klog.V(4).Infof("The resource utilization of all nodes is above the threshold")
		return victims
	}
	if len(lowNodes) == len(Session.Nodes) {
		klog.V(4).Infof("The resource utilization of all nodes is below the threshold")
		return victims
	}
	if len(highNodes) == 0 {
		klog.V(4).Infof("The resource utilization of all nodes is below the target threshold")
		return victims
	}

	// select victims from lowNodes
	return evictPodsFromSourceNodes(highNodes, lowNodes, tasks, isContinueEvictPods, *utilizationConfig)
}

// lowThresholdFilter filter nodes which all resource dimensions are under the low utilization threshold
func lowThresholdFilter(node *v1.Node, usage *NodeUtilization, config interface{}) bool {
	utilizationConfig := parseArgToConfig(config)
	if utilizationConfig == nil {
		klog.V(4).Infof("lack of LowNodeUtilizationConf pointer parameter")
		return false
	}

	if node.Spec.Unschedulable {
		return false
	}
	nodeCapacity := getNodeCapacity(node)
	for rName, usage := range usage.utilization {
		if thresholdPercent, ok := utilizationConfig.Thresholds[string(rName)]; ok {
			threshold := getThresholdForNode(rName, thresholdPercent, nodeCapacity)
			if usage.Cmp(*threshold) == 1 {
				return false
			}
		}
	}
	return true
}

// highThresholdFilter filter nodes which at least one resource dimension above the target utilization threshold
func highThresholdFilter(node *v1.Node, usage *NodeUtilization, config interface{}) bool {
	utilizationConfig := parseArgToConfig(config)
	if utilizationConfig == nil {
		klog.V(4).Infof("lack of LowNodeUtilizationConf pointer parameter")
		return false
	}

	nodeCapacity := getNodeCapacity(node)
	for rName, usage := range usage.utilization {
		if thresholdPercent, ok := utilizationConfig.TargetThresholds[string(rName)]; ok {
			threshold := getThresholdForNode(rName, thresholdPercent, nodeCapacity)
			if usage.Cmp(*threshold) == 1 {
				return true
			}
		}
	}
	return false
}

// isContinueEvictPods judges whether continue to select victim pods
func isContinueEvictPods(node *v1.Node, usage *NodeUtilization, totalAllocatableResource map[v1.ResourceName]*resource.Quantity, config interface{}) bool {
	var isNodeOverused bool
	utilizationConfig := parseArgToConfig(config)
	nodeCapacity := getNodeCapacity(node)
	for rName, usage := range usage.utilization {
		if thresholdPercent, ok := utilizationConfig.TargetThresholds[string(rName)]; ok {
			threshold := getThresholdForNode(rName, thresholdPercent, nodeCapacity)
			if usage.Cmp(*threshold) == 1 {
				isNodeOverused = true
				break
			}
		}
	}
	if !isNodeOverused {
		return false
	}

	for rName := range totalAllocatableResource {
		if totalAllocatableResource[rName].CmpInt64(0) == 0 {
			return false
		}
	}
	return true
}

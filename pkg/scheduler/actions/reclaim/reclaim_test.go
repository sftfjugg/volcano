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

package reclaim

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"

	schedulingv1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/conf"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/plugins/capacity"
	"volcano.sh/volcano/pkg/scheduler/plugins/conformance"
	"volcano.sh/volcano/pkg/scheduler/plugins/gang"
	"volcano.sh/volcano/pkg/scheduler/plugins/priority"
	"volcano.sh/volcano/pkg/scheduler/plugins/proportion"
	"volcano.sh/volcano/pkg/scheduler/uthelper"
	"volcano.sh/volcano/pkg/scheduler/util"
)

func TestReclaim(t *testing.T) {
	tests := []uthelper.TestCommonStruct{
		{
			Name: "Two Queue with one Queue overusing resource, should reclaim",
			Plugins: map[string]framework.PluginBuilder{
				conformance.PluginName: conformance.New,
				gang.PluginName:        gang.New,
				proportion.PluginName:  proportion.New,
			},
			PodGroups: []*schedulingv1beta1.PodGroup{
				util.BuildPodGroupWithPrio("pg1", "c1", "q1", 0, nil, schedulingv1beta1.PodGroupInqueue, "low-priority"),
				util.BuildPodGroupWithPrio("pg2", "c1", "q2", 0, nil, schedulingv1beta1.PodGroupInqueue, "high-priority"),
			},
			Pods: []*v1.Pod{
				util.BuildPod("c1", "preemptee1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "false"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee2", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee3", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "false"}, make(map[string]string)),
				util.BuildPod("c1", "preemptor1", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("3", "3Gi", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1beta1.Queue{
				util.BuildQueue("q1", 1, nil),
				util.BuildQueue("q2", 1, nil),
			},
			ExpectEvictNum: 1,
			ExpectEvicted:  []string{"c1/preemptee2"}, // let pod2 in the middle when sort tasks be preemptable and will not disturb
		},
		{
			Name: "sort reclaimees when reclaiming from overusing queue",
			Plugins: map[string]framework.PluginBuilder{
				conformance.PluginName: conformance.New,
				gang.PluginName:        gang.New,
				priority.PluginName:    priority.New,
				proportion.PluginName:  proportion.New,
			},
			PriClass: []*schedulingv1.PriorityClass{
				util.BuildPriorityClass("low-priority", 100),
				util.BuildPriorityClass("mid-priority", 500),
				util.BuildPriorityClass("high-priority", 1000),
			},
			PodGroups: []*schedulingv1beta1.PodGroup{
				util.BuildPodGroupWithPrio("pg1", "c1", "q1", 0, nil, schedulingv1beta1.PodGroupInqueue, "mid-priority"),
				util.BuildPodGroupWithPrio("pg2", "c1", "q2", 0, nil, schedulingv1beta1.PodGroupInqueue, "low-priority"), // reclaimed first
				util.BuildPodGroupWithPrio("pg3", "c1", "q3", 0, nil, schedulingv1beta1.PodGroupInqueue, "high-priority"),
			},
			Pods: []*v1.Pod{
				util.BuildPod("c1", "preemptee1-1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee1-2", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee2-1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg2", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee2-2", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg2", map[string]string{schedulingv1beta1.PodPreemptable: "false"}, make(map[string]string)),
				util.BuildPod("c1", "preemptor1", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg3", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("4", "4Gi", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1beta1.Queue{
				util.BuildQueue("q1", 1, nil),
				util.BuildQueue("q2", 1, nil),
				util.BuildQueue("q3", 1, nil),
			},
			ExpectEvictNum: 1,
			ExpectEvicted:  []string{"c1/preemptee2-1"}, // low priority job's preemptable pod is evicted
		},
		{
			Name: "sort reclaimees when reclaiming from overusing queues with different queue priority",
			Plugins: map[string]framework.PluginBuilder{
				conformance.PluginName: conformance.New,
				gang.PluginName:        gang.New,
				proportion.PluginName:  proportion.New,
			},
			PodGroups: []*schedulingv1beta1.PodGroup{
				util.BuildPodGroupWithPrio("pg1", "c1", "q1", 0, nil, schedulingv1beta1.PodGroupInqueue, "mid-priority"),
				util.BuildPodGroupWithPrio("pg2", "c1", "q2", 0, nil, schedulingv1beta1.PodGroupInqueue, "mid-priority"),
				util.BuildPodGroupWithPrio("pg3", "c1", "q3", 0, nil, schedulingv1beta1.PodGroupInqueue, "mid-priority"),
			},
			Pods: []*v1.Pod{
				util.BuildPod("c1", "preemptee1-1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee1-2", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "false"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee2-1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg2", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPod("c1", "preemptee2-2", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg2", map[string]string{schedulingv1beta1.PodPreemptable: "false"}, make(map[string]string)),
				util.BuildPod("c1", "preemptor1", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg3", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("4", "4Gi", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1beta1.Queue{
				util.BuildQueueWithPriorityAndResourcesQuantity("q1", 5, nil, nil),
				util.BuildQueueWithPriorityAndResourcesQuantity("q2", 10, nil, nil), // highest queue priority
				util.BuildQueueWithPriorityAndResourcesQuantity("q3", 1, nil, nil),
			},
			ExpectEvictNum: 1,
			ExpectEvicted:  []string{"c1/preemptee1-1"}, // low queue priority job's preemptable pod is evicted
		},
		{
			// case about #3642
			Name: "can not reclaim resources when task preemption policy is never",
			Plugins: map[string]framework.PluginBuilder{
				conformance.PluginName: conformance.New,
				gang.PluginName:        gang.New,
				proportion.PluginName:  proportion.New,
			},
			PriClass: []*schedulingv1.PriorityClass{
				util.BuildPriorityClass("low-priority", 100),
				util.BuildPriorityClassWithPreemptionPolicy("high-priority", 1000, v1.PreemptNever),
			},
			PodGroups: []*schedulingv1beta1.PodGroup{
				util.BuildPodGroupWithPrio("pg1", "c1", "q1", 0, nil, schedulingv1beta1.PodGroupInqueue, "low-priority"),
				util.BuildPodGroupWithPrio("pg2", "c1", "q2", 0, nil, schedulingv1beta1.PodGroupInqueue, "high-priority"),
			},
			Pods: []*v1.Pod{
				util.BuildPod("c1", "preemptee1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
				util.BuildPodWithPreeemptionPolicy("c1", "preemptor1", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string), v1.PreemptNever),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("1", "1Gi", []api.ScalarResource{{Name: "pods", Value: "1"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1beta1.Queue{
				util.BuildQueue("q1", 5, nil),
				util.BuildQueue("q2", 10, nil),
			},
			ExpectEvictNum: 0,
			ExpectEvicted:  []string{}, // no victims should be reclaimed
		},
	}

	reclaim := New()
	trueValue := true
	tiers := []conf.Tier{
		{
			Plugins: []conf.PluginOption{
				{
					Name:               conformance.PluginName,
					EnabledReclaimable: &trueValue,
				},
				{
					Name:               gang.PluginName,
					EnabledReclaimable: &trueValue,
				},
				{ // proportion plugin will cause deserved resource large than preemptable pods's usage, and return less victims
					Name:               proportion.PluginName,
					EnabledReclaimable: &trueValue,
					EnabledQueueOrder:  &trueValue,
				},
				{
					Name:             priority.PluginName,
					EnabledJobOrder:  &trueValue,
					EnabledTaskOrder: &trueValue,
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.RegisterSession(tiers, nil)
			defer test.Close()
			test.Run([]framework.Action{reclaim})
			if err := test.CheckAll(i); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEnableGangReclaim(t *testing.T) {
	req1 := api.BuildResourceList("1", "1G")
	req2 := api.BuildResourceList("2", "2G")
	min := api.BuildResourceList("2", "2G")
	mid := api.BuildResourceList("3", "3G")
	max := api.BuildResourceList("4", "4G") // 2*req2
	common := uthelper.TestCommonStruct{
		Plugins: map[string]framework.PluginBuilder{
			conformance.PluginName: conformance.New,
			gang.PluginName:        gang.New,
			capacity.PluginName:    capacity.New,
		},
		PodGroups: []*schedulingv1beta1.PodGroup{
			util.BuildPodGroupWithMinResources("pg0", "c1", "q1", 0, nil, nil, schedulingv1beta1.PodGroupRunning),
			util.BuildPodGroupWithMinResources("pg1", "c1", "q2", 1, nil, req1, schedulingv1beta1.PodGroupRunning),
			util.BuildPodGroupWithMinResources("pg2", "c1", "q2", 2, nil, max, schedulingv1beta1.PodGroupInqueue),
		},
		Pods: []*v1.Pod{
			util.BuildPod("c1", "preemptee1", "n1", v1.PodRunning, req2, "pg0", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
			util.BuildPod("c1", "preemptee2", "n1", v1.PodRunning, req1, "pg1", map[string]string{schedulingv1beta1.PodPreemptable: "true"}, make(map[string]string)),
			util.BuildPod("c1", "preemptor1", "", v1.PodPending, req2, "pg2", nil, nil),
			util.BuildPod("c1", "preemptor2", "", v1.PodPending, req2, "pg2", nil, nil),
		},
		Nodes: []*v1.Node{
			util.BuildNode("n1", api.BuildResourceList("4", "4Gi", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
		},
		Queues: []*schedulingv1beta1.Queue{
			util.BuildQueueWithResourcesQuantity("q1", req1, min),
			util.BuildQueueWithResourcesQuantity("q2", mid, max),
		},
	}
	tests := []struct {
		enableGang bool
		uthelper.TestCommonStruct
	}{
		{
			enableGang: false,
			TestCommonStruct: uthelper.TestCommonStruct{
				Name:           "when enableGangCheckOverused=false, can reclaim one pod but can not meet gang",
				Plugins:        common.Plugins,
				PodGroups:      common.PodGroups,
				Pods:           common.Pods,
				Nodes:          common.Nodes,
				Queues:         common.Queues,
				ExpectEvictNum: 1,
				ExpectEvicted:  []string{"c1/preemptee1"},
			},
		},
		{
			enableGang: true,
			TestCommonStruct: uthelper.TestCommonStruct{
				Name:           "when enableGangCheckOverused=true, can not reclaim",
				Plugins:        common.Plugins,
				PodGroups:      common.PodGroups,
				Pods:           common.Pods,
				Nodes:          common.Nodes,
				Queues:         common.Queues,
				ExpectEvictNum: 0,
				ExpectEvicted:  []string{},
			},
		},
	}

	reclaim := New()
	trueValue := true
	tiers := []conf.Tier{
		{
			Plugins: []conf.PluginOption{
				{
					Name:               conformance.PluginName,
					EnabledReclaimable: &trueValue,
				},
				{
					Name:               gang.PluginName,
					EnabledReclaimable: &trueValue,
				},
				{
					Name:               capacity.PluginName,
					EnabledReclaimable: &trueValue,
					EnabledOverused:    &trueValue,
					EnabledAllocatable: &trueValue,
					EnablePreemptive:   &trueValue,
				},
				{
					Name:             priority.PluginName,
					EnabledJobOrder:  &trueValue,
					EnabledTaskOrder: &trueValue,
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.RegisterSession(tiers, []conf.Configuration{{Name: reclaim.Name(), Arguments: map[string]interface{}{conf.EnableGangCheckOverusedKey: test.enableGang}}})
			defer test.Close()
			test.Run([]framework.Action{reclaim})
			if err := test.CheckAll(i); err != nil {
				t.Fatal(err)
			}
		})
	}
}

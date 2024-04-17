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

	schedulingv1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/conf"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/plugins/conformance"
	"volcano.sh/volcano/pkg/scheduler/plugins/gang"
	"volcano.sh/volcano/pkg/scheduler/plugins/proportion"
	"volcano.sh/volcano/pkg/scheduler/uthelper"
	"volcano.sh/volcano/pkg/scheduler/util"
)

func TestReclaim(t *testing.T) {
	plugins := map[string]framework.PluginBuilder{
		conformance.PluginName: conformance.New,
		gang.PluginName:        gang.New,
		proportion.PluginName:  proportion.New,
	}

	tests := []uthelper.TestCommonStruct{
		{
			Name: "Two Queue with one Queue overusing resource, should reclaim",
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
			EvictNum: 1,
			Evicted:  []string{"c1/preemptee2"}, // let pod2 in the middle when sort tasks be preemptable and will not disturb
		},
	}

	reclaim := New()

	for i, test := range tests {
		trueValue := true
		tiers := []conf.Tier{
			{
				Plugins: []conf.PluginOption{
					{
						Name:               "conformance",
						EnabledReclaimable: &trueValue,
					},
					{
						Name:               "gang",
						EnabledReclaimable: &trueValue,
					},
					{
						Name:               "proportion",
						EnabledReclaimable: &trueValue,
					},
				},
			},
		}
		test.Plugins = plugins
		test.RegisterSession(tiers, nil)
		test.Run([]framework.Action{reclaim})
		if err := test.CheckAll(i); err != nil {
			t.Fatal(err)
		}
	}
}

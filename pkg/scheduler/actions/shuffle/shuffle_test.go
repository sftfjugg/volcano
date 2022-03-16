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

package shuffle

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	schedulingv1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/cache"
	"volcano.sh/volcano/pkg/scheduler/conf"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/plugins/rescheduling"
	"volcano.sh/volcano/pkg/scheduler/util"
)

func TestShuffle(t *testing.T) {
	framework.RegisterPluginBuilder("rescheduling", rescheduling.New)
	defer framework.CleanupPluginBuilders()
	util.FnsLastExecTime["reschedulingFns"] = time.Now()
	defer delete(util.FnsLastExecTime, "reschedulingFns")

	var highPriority int32
	var lowPriority int32
	highPriority = 100000
	lowPriority = 10
	tests := []struct {
		name      string
		podGroups []*schedulingv1beta1.PodGroup
		pods      []*v1.Pod
		nodes     []*v1.Node
		queues    []*schedulingv1beta1.Queue
		interval  string
		expected  int
	}{
		{
			name: "scheduled task test for rescheduling cycle",
			nodes: []*v1.Node{
				util.BuildNode("node1", util.BuildResourceList("4", "8Gi"), make(map[string]string)),
				util.BuildNode("node2", util.BuildResourceList("4", "8Gi"), make(map[string]string)),
			},
			queues: []*schedulingv1beta1.Queue{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: schedulingv1beta1.QueueSpec{
						Weight: 1,
					},
				},
			},
			podGroups: []*schedulingv1beta1.PodGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg1",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "default",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg2",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "default",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg3",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "default",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg4",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "default",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
			},
			pods: []*v1.Pod{
				util.BuildPod("test", "pod1", "node1", v1.PodRunning, util.BuildResourceList("1", "2G"), "pg1", make(map[string]string), make(map[string]string)),
				util.BuildPod("test", "pod2", "node1", v1.PodRunning, util.BuildResourceList("1", "2G"), "pg2", make(map[string]string), make(map[string]string)),
				util.BuildPod("test", "pod3", "node1", v1.PodRunning, util.BuildResourceList("1", "2G"), "pg3", make(map[string]string), make(map[string]string)),
				util.BuildPod("test", "pod4", "node2", v1.PodRunning, util.BuildResourceList("1", "2G"), "pg4", make(map[string]string), make(map[string]string)),
			},
			interval: "5m",
			expected: 0,
		},
		{
			name: "rescheduling works with lowNodeUtilization strategy on condition that 3 nodes and 6 different pods",
			nodes: []*v1.Node{
				util.BuildNode("node1", util.BuildResourceList("4", "8Gi"), make(map[string]string)),
				util.BuildNode("node2", util.BuildResourceList("4", "8Gi"), make(map[string]string)),
				util.BuildNode("node3", util.BuildResourceList("4", "8Gi"), make(map[string]string)),
			},
			queues: []*schedulingv1beta1.Queue{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: schedulingv1beta1.QueueSpec{
						Weight: 1,
					},
				},
			},
			podGroups: []*schedulingv1beta1.PodGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg1",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue:             "default",
						PriorityClassName: "high-priority",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg2",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue:             "default",
						PriorityClassName: "high-priority",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg3",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue:             "default",
						PriorityClassName: "high-priority",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg4",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue:             "default",
						PriorityClassName: "high-priority",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg5",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue:             "default",
						PriorityClassName: "low-priority",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg6",
						Namespace: "test",
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue:             "default",
						PriorityClassName: "high-priority",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupRunning,
					},
				},
			},
			pods: []*v1.Pod{
				util.BuildPod("test", "pod1", "node1", v1.PodRunning, util.BuildResourceList("1.5", "3G"), "pg1", make(map[string]string), make(map[string]string)),
				util.BuildBestEffortPod("test", "pod2", "node1", v1.PodRunning, "pg2", make(map[string]string), make(map[string]string)),
				util.BuildPod("test", "pod3", "node1", v1.PodRunning, util.BuildResourceList("2", "4G"), "pg3", make(map[string]string), make(map[string]string)),
				util.BuildPodWithPriority("test", "pod4", "node2", v1.PodRunning, util.BuildResourceList("1", "2G"), "pg4", make(map[string]string), make(map[string]string), &highPriority),
				util.BuildPodWithPriority("test", "pod5", "node2", v1.PodRunning, util.BuildResourceList("0.5", "1G"), "pg5", make(map[string]string), make(map[string]string), &lowPriority),
				util.BuildPodWithPriority("test", "pod6", "node2", v1.PodRunning, util.BuildResourceList("1.5", "3G"), "pg6", make(map[string]string), make(map[string]string), &highPriority),
			},
			interval: "1s",
			expected: 3,
		},
	}
	shuffle := New()

	for i, test := range tests {
		binder := &util.FakeBinder{
			Binds:   map[string]string{},
			Channel: make(chan string, 1),
		}
		evictor := &util.FakeEvictor{
			Channel: make(chan string),
		}
		schedulerCache := &cache.SchedulerCache{
			Nodes:           make(map[string]*api.NodeInfo),
			Jobs:            make(map[api.JobID]*api.JobInfo),
			Queues:          make(map[api.QueueID]*api.QueueInfo),
			Binder:          binder,
			Evictor:         evictor,
			StatusUpdater:   &util.FakeStatusUpdater{},
			VolumeBinder:    &util.FakeVolumeBinder{},
			PriorityClasses: make(map[string]*schedulingv1.PriorityClass),

			Recorder: record.NewFakeRecorder(100),
		}
		schedulerCache.PriorityClasses["high-priority"] = &schedulingv1.PriorityClass{
			Value: highPriority,
		}
		schedulerCache.PriorityClasses["low-priority"] = &schedulingv1.PriorityClass{
			Value: lowPriority,
		}

		for _, node := range test.nodes {
			schedulerCache.AddNode(node)
		}
		for _, q := range test.queues {
			schedulerCache.AddQueueV1beta1(q)
		}
		for _, ss := range test.podGroups {
			schedulerCache.AddPodGroupV1beta1(ss)
		}
		for _, pod := range test.pods {
			schedulerCache.AddPod(pod)
		}

		time.Sleep(1 * time.Second)
		trueValue := true
		ssn := framework.OpenSession(schedulerCache, []conf.Tier{
			{
				Plugins: []conf.PluginOption{
					{
						Name:          "rescheduling",
						EnabledVictim: &trueValue,
						Arguments: map[string]interface{}{
							"interval": test.interval,
							"strategies": []rescheduling.Strategy{
								{
									Name: "lowNodeUtilization",
									Parameters: map[string]interface{}{
										"thresholds": map[string]int{
											"cpu":    30,
											"memory": 30,
											"pods":   30,
										},
										"targetThresholds": map[string]int{
											"cpu":    70,
											"memory": 70,
											"pods":   70,
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil)
		defer framework.CloseSession(ssn)

		shuffle.Execute(ssn)

		for {
			select {
			case <-evictor.Channel:
			case <-time.After(2 * time.Second):
				goto LOOP
			}
		}

	LOOP:
		if test.expected != len(evictor.Evicts()) {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, test.name, test.expected, len(evictor.Evicts()))
		}
	}
}

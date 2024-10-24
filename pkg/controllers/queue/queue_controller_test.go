/*
Copyright 2019 The Volcano Authors.

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

package queue

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	schedulingv1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	vcclient "volcano.sh/apis/pkg/client/clientset/versioned/fake"
	informerfactory "volcano.sh/apis/pkg/client/informers/externalversions"
	"volcano.sh/volcano/pkg/controllers/apis"
	"volcano.sh/volcano/pkg/controllers/framework"
)

func newFakeController() *queuecontroller {
	KubeBatchClientSet := vcclient.NewSimpleClientset()
	KubeClientSet := kubeclient.NewSimpleClientset()

	vcSharedInformers := informerfactory.NewSharedInformerFactory(KubeBatchClientSet, 0)

	controller := &queuecontroller{}
	opt := framework.ControllerOption{
		VolcanoClient:           KubeBatchClientSet,
		KubeClient:              KubeClientSet,
		VCSharedInformerFactory: vcSharedInformers,
	}

	controller.Initialize(&opt)

	return controller
}

func TestAddQueue(t *testing.T) {
	testCases := []struct {
		Name        string
		queue       *schedulingv1beta1.Queue
		ExpectValue int
	}{
		{
			Name: "AddQueue",
			queue: &schedulingv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name: "c1",
				},
				Spec: schedulingv1beta1.QueueSpec{
					Weight: 1,
				},
			},
			ExpectValue: 1,
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()

		c.addQueue(testcase.queue)

		if testcase.ExpectValue != c.queue.Len() {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, c.queue.Len())
		}
	}
}

func TestDeleteQueue(t *testing.T) {
	testCases := []struct {
		Name        string
		queue       *schedulingv1beta1.Queue
		ExpectValue bool
	}{
		{
			Name: "DeleteQueue",
			queue: &schedulingv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name: "c1",
				},
				Spec: schedulingv1beta1.QueueSpec{
					Weight: 1,
				},
			},
			ExpectValue: false,
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()
		c.podGroups[testcase.queue.Name] = make(map[string]struct{})

		c.deleteQueue(testcase.queue)

		if _, ok := c.podGroups[testcase.queue.Name]; ok != testcase.ExpectValue {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, ok)
		}
	}

}

func TestAddPodGroup(t *testing.T) {
	namespace := "c1"

	testCases := []struct {
		Name        string
		podGroup    *schedulingv1beta1.PodGroup
		ExpectValue int
	}{
		{
			Name: "addpodgroup",
			podGroup: &schedulingv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pg1",
					Namespace: namespace,
				},
				Spec: schedulingv1beta1.PodGroupSpec{
					Queue: "c1",
				},
			},
			ExpectValue: 1,
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()

		c.addPodGroup(testcase.podGroup)

		if testcase.ExpectValue != c.queue.Len() {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, c.queue.Len())
		}
		if testcase.ExpectValue != len(c.podGroups[testcase.podGroup.Spec.Queue]) {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, len(c.podGroups[testcase.podGroup.Spec.Queue]))
		}
	}

}

func TestDeletePodGroup(t *testing.T) {
	namespace := "c1"

	testCases := []struct {
		Name        string
		podGroup    *schedulingv1beta1.PodGroup
		ExpectValue bool
	}{
		{
			Name: "deletepodgroup",
			podGroup: &schedulingv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pg1",
					Namespace: namespace,
				},
				Spec: schedulingv1beta1.PodGroupSpec{
					Queue: "c1",
				},
			},
			ExpectValue: false,
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()

		key, _ := cache.MetaNamespaceKeyFunc(testcase.podGroup)
		c.podGroups[testcase.podGroup.Spec.Queue] = make(map[string]struct{})
		c.podGroups[testcase.podGroup.Spec.Queue][key] = struct{}{}

		c.deletePodGroup(testcase.podGroup)
		if _, ok := c.podGroups[testcase.podGroup.Spec.Queue][key]; ok != testcase.ExpectValue {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, ok)
		}
	}
}

func TestUpdatePodGroup(t *testing.T) {
	namespace := "c1"

	testCases := []struct {
		Name        string
		podGroupold *schedulingv1beta1.PodGroup
		podGroupnew *schedulingv1beta1.PodGroup
		ExpectValue int
	}{
		{
			Name: "updatepodgroup",
			podGroupold: &schedulingv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pg1",
					Namespace: namespace,
				},
				Spec: schedulingv1beta1.PodGroupSpec{
					Queue: "c1",
				},
				Status: schedulingv1beta1.PodGroupStatus{
					Phase: schedulingv1beta1.PodGroupPending,
				},
			},
			podGroupnew: &schedulingv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pg1",
					Namespace: namespace,
				},
				Spec: schedulingv1beta1.PodGroupSpec{
					Queue: "c1",
				},
				Status: schedulingv1beta1.PodGroupStatus{
					Phase: schedulingv1beta1.PodGroupRunning,
				},
			},
			ExpectValue: 1,
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()

		c.updatePodGroup(testcase.podGroupold, testcase.podGroupnew)

		if testcase.ExpectValue != c.queue.Len() {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, c.queue.Len())
		}
	}
}

func TestSyncQueue(t *testing.T) {
	namespace := "c1"

	testCases := []struct {
		Name          string
		pgsInCache    []*schedulingv1beta1.PodGroup
		pgsInInformer []*schedulingv1beta1.PodGroup
		queue         *schedulingv1beta1.Queue
		ExpectStatus  schedulingv1beta1.QueueStatus
	}{
		{
			Name: "syncQueue",
			pgsInCache: []*schedulingv1beta1.PodGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg1",
						Namespace: namespace,
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "c1",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupPending,
					},
				},
			},
			pgsInInformer: []*schedulingv1beta1.PodGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg1",
						Namespace: namespace,
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "c1",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupPending,
					},
				},
			},
			queue: &schedulingv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name: "c1",
				},
				Spec: schedulingv1beta1.QueueSpec{
					Weight: 1,
				},
			},
			ExpectStatus: schedulingv1beta1.QueueStatus{
				Pending:     1,
				Reservation: schedulingv1beta1.Reservation{},
				Allocated:   v1.ResourceList{},
			},
		},
		{
			Name: "syncQueueHandlingNotFoundPg",
			pgsInCache: []*schedulingv1beta1.PodGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg1",
						Namespace: namespace,
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "c2",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg2",
						Namespace: namespace,
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "c2",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupPending,
					},
				},
			},
			pgsInInformer: []*schedulingv1beta1.PodGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pg2",
						Namespace: namespace,
					},
					Spec: schedulingv1beta1.PodGroupSpec{
						Queue: "c2",
					},
					Status: schedulingv1beta1.PodGroupStatus{
						Phase: schedulingv1beta1.PodGroupPending,
					},
				},
			},
			queue: &schedulingv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name: "c2",
				},
				Spec: schedulingv1beta1.QueueSpec{
					Weight: 1,
				},
			},
			ExpectStatus: schedulingv1beta1.QueueStatus{
				Pending:     1,
				Reservation: schedulingv1beta1.Reservation{},
				Allocated:   v1.ResourceList{},
			},
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()

		for j := range testcase.pgsInCache {
			key, _ := cache.MetaNamespaceKeyFunc(testcase.pgsInCache[j])
			if _, ok := c.podGroups[testcase.pgsInCache[j].Spec.Queue]; !ok {
				c.podGroups[testcase.pgsInCache[j].Spec.Queue] = make(map[string]struct{})
			}
			c.podGroups[testcase.pgsInCache[j].Spec.Queue][key] = struct{}{}
		}

		for j := range testcase.pgsInInformer {
			c.pgInformer.Informer().GetIndexer().Add(testcase.pgsInInformer[j])
		}

		c.queueInformer.Informer().GetIndexer().Add(testcase.queue)
		c.vcClient.SchedulingV1beta1().Queues().Create(context.TODO(), testcase.queue, metav1.CreateOptions{})

		err := c.syncQueue(testcase.queue, nil)

		item, _ := c.vcClient.SchedulingV1beta1().Queues().Get(context.TODO(), testcase.queue.Name, metav1.GetOptions{})
		if err != nil && !reflect.DeepEqual(testcase.ExpectStatus, item.Status) {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectStatus, item.Status)
		}
	}
}

func TestProcessNextWorkItem(t *testing.T) {
	testCases := []struct {
		Name        string
		ExpectValue int32
	}{
		{
			Name:        "processNextWorkItem",
			ExpectValue: 0,
		},
	}

	for i, testcase := range testCases {
		c := newFakeController()
		c.queue.Add(&apis.Request{JobName: "test"})
		bVal := c.processNextWorkItem()
		fmt.Println("The value of boolean is ", bVal)
		if c.queue.Len() != 0 {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, c.queue.Len())
		}
	}
}

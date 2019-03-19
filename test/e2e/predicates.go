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

package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/api"
)

var _ = Describe("Predicates E2E Test", func() {
	It("NodeAffinity", func() {
		context := initTestContext()
		defer cleanupTestContext(context)

		slot := oneCPU
		nodeName, rep := computeNode(context, oneCPU)
		Expect(rep).NotTo(Equal(0))

		affinity := &v1.Affinity{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchFields: []v1.NodeSelectorRequirement{
								{
									Key:      api.NodeFieldSelectorKeyNodeName,
									Operator: v1.NodeSelectorOpIn,
									Values:   []string{nodeName},
								},
							},
						},
					},
				},
			},
		}

		spec := &jobSpec{
			name: "na-spec",
			tasks: []taskSpec{
				{
					img:      defaultNginxImage,
					req:      slot,
					min:      1,
					rep:      1,
					affinity: affinity,
				},
			},
		}

		job := createJob(context, spec)
		err := waitJobReady(context, job)
		Expect(err).NotTo(HaveOccurred())

		pods := getTasksOfJob(context, job)
		for _, pod := range pods {
			Expect(pod.Spec.NodeName).To(Equal(nodeName))
		}
	})

	It("Hostport", func() {
		context := initTestContext()
		defer cleanupTestContext(context)

		nn := clusterNodeNumber(context)

		spec := &jobSpec{
			name: "hp-spec",
			tasks: []taskSpec{
				{
					img:      defaultNginxImage,
					min:      int32(nn),
					req:      oneCPU,
					rep:      int32(nn * 2),
					hostport: 28080,
				},
			},
		}

		job := createJob(context, spec)

		err := waitTasksReady(context, job, nn)
		Expect(err).NotTo(HaveOccurred())

		err = waitTasksPending(context, job, nn)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Pod Affinity", func() {
		context := initTestContext()
		defer cleanupTestContext(context)

		slot := oneCPU
		_, rep := computeNode(context, oneCPU)
		Expect(rep).NotTo(Equal(0))

		labels := map[string]string{"foo": "bar"}

		affinity := &v1.Affinity{
			PodAffinity: &v1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		}

		spec := &jobSpec{
			name: "pa-spec",
			tasks: []taskSpec{
				{
					img:      defaultNginxImage,
					req:      slot,
					min:      rep,
					rep:      rep,
					affinity: affinity,
					labels:   labels,
				},
			},
		}

		job := createJob(context, spec)
		err := waitJobReady(context, job)
		Expect(err).NotTo(HaveOccurred())

		pods := getTasksOfJob(context, job)
		// All pods should be scheduled to the same node.
		nodeName := pods[0].Spec.NodeName
		for _, pod := range pods {
			Expect(pod.Spec.NodeName).To(Equal(nodeName))
		}
	})

	It("Taints/Tolerations", func() {
		context := initTestContext()
		defer cleanupTestContext(context)

		taints := []v1.Taint{
			{
				Key:    "test-taint-key",
				Value:  "test-taint-val",
				Effect: v1.TaintEffectNoSchedule,
			},
		}

		err := taintAllNodes(context, taints)
		Expect(err).NotTo(HaveOccurred())

		spec := &jobSpec{
			name: "tt-spec",
			tasks: []taskSpec{
				{
					img: defaultNginxImage,
					req: oneCPU,
					min: 1,
					rep: 1,
				},
			},
		}

		job := createJob(context, spec)
		err = waitJobPending(context, job)
		Expect(err).NotTo(HaveOccurred())

		err = removeTaintsFromAllNodes(context, taints)
		Expect(err).NotTo(HaveOccurred())

		err = waitJobReady(context, job)
		Expect(err).NotTo(HaveOccurred())
	})

})

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
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Job E2E Test: Test Admission service", func() {

	ginkgo.It("Default queue would be added", func() {
		jobName := "job-default-queue"
		namespace := "test"
		context := initTestContext()
		defer cleanupTestContext(context)

		_, err := createJobInner(context, &jobSpec{
			min:       1,
			namespace: namespace,
			name:      jobName,
			tasks: []taskSpec{
				{
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  1,
					rep:  1,
					name: "taskname",
				},
			},
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		createdJob, err := context.vkclient.BatchV1alpha1().Jobs(namespace).Get(jobName, v1.GetOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(createdJob.Spec.Queue).Should(gomega.Equal("default"),
			"Job queue attribute would default to 'default' ")
	})

})

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

	vkv1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

var _ = Describe("MPI E2E Test", func() {
	cleanupResources := CleanupResources{}
	var context *context

	BeforeEach(func() {
		context = gContext
	})

	AfterEach(func() {
		deleteResources(gContext, cleanupResources)
	})
	It("will run and complete finally", func() {
		jobName := "mpi"
		cleanupResources.Jobs = []string{jobName}

		slot := oneCPU

		spec := &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.CompleteJobAction,
					Event:  vkv1.TaskCompletedEvent,
				},
			},
			plugins: map[string][]string{
				"ssh": {},
				"env": {},
			},
			tasks: []taskSpec{
				{
					name:       "mpimaster",
					img:        defaultMPIImage,
					req:        slot,
					min:        1,
					rep:        1,
					workingDir: "/home",
					//Need sometime waiting for worker node ready
					command: `sleep 5;
mkdir -p /var/run/sshd; /usr/sbin/sshd;
mpiexec --allow-run-as-root --hostfile /etc/volcano/mpiworker.host -np 2 mpi_hello_world > /home/re`,
				},
				{
					name:       "mpiworker",
					img:        defaultMPIImage,
					req:        slot,
					min:        2,
					rep:        2,
					workingDir: "/home",
					command:    "mkdir -p /var/run/sshd; /usr/sbin/sshd -D;",
				},
			},
		}

		job := createJob(context, spec)

		err := waitJobStates(context, job, []vkv1.JobPhase{
			vkv1.Pending, vkv1.Running, vkv1.Completing, vkv1.Completed})
		Expect(err).NotTo(HaveOccurred())
	})

})

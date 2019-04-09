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
	vkv1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	jobutil "volcano.sh/volcano/pkg/controllers/job"
)

var _ = Describe("Job Error Handling", func() {
	cleanupResources := CleanupResources{}
	var context *context

	BeforeEach(func() {
		context = gContext
	})

	AfterEach(func() {
		deleteResources(gContext, cleanupResources)
	})

	It("job level LifecyclePolicy, Event: PodFailed; Action: RestartJob", func() {
		By("init test context")
		jobName := "failed-restart-job"
		cleanupResources.Jobs = []string{jobName}
		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.RestartJobAction,
					Event:  vkv1.PodFailedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name:          "fail",
					img:           defaultNginxImage,
					min:           2,
					rep:           2,
					command:       "sleep 10s && xxx",
					restartPolicy: v1.RestartPolicyNever,
				},
			},
		})

		// job phase: pending -> running -> restarting
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running, vkv1.Restarting})
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: PodFailed; Action: TerminateJob", func() {
		By("init test context")
		jobName := "failed-terminate-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.TerminateJobAction,
					Event:  vkv1.PodFailedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name:          "fail",
					img:           defaultNginxImage,
					min:           2,
					rep:           2,
					command:       "sleep 10s && xxx",
					restartPolicy: v1.RestartPolicyNever,
				},
			},
		})

		// job phase: pending -> running -> Terminating -> Terminated
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running, vkv1.Terminating, vkv1.Terminated})
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: PodFailed; Action: AbortJob", func() {
		By("init test context")
		jobName := "failed-abort-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.AbortJobAction,
					Event:  vkv1.PodFailedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name:          "fail",
					img:           defaultNginxImage,
					min:           2,
					rep:           2,
					command:       "sleep 10s && xxx",
					restartPolicy: v1.RestartPolicyNever,
				},
			},
		})

		// job phase: pending -> running -> Aborting -> Aborted
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running, vkv1.Aborting, vkv1.Aborted})
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: PodEvicted; Action: RestartJob", func() {
		By("init test context")
		jobName := "evicted-restart-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.RestartJobAction,
					Event:  vkv1.PodEvictedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name: "delete",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
			},
		})

		// job phase: pending -> running
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running})
		Expect(err).NotTo(HaveOccurred())

		By("delete one pod of job")
		podName := jobutil.MakePodName(job.Name, "delete", 0)
		err = context.kubeclient.CoreV1().Pods(job.Namespace).Delete(podName, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// job phase: Restarting -> Running
		err = waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Restarting, vkv1.Running})
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: PodEvicted; Action: TerminateJob", func() {
		By("init test context")
		jobName := "evicted-terminate-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.TerminateJobAction,
					Event:  vkv1.PodEvictedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name: "delete",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
			},
		})

		// job phase: pending -> running
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running})
		Expect(err).NotTo(HaveOccurred())

		By("delete one pod of job")
		podName := jobutil.MakePodName(job.Name, "delete", 0)
		err = context.kubeclient.CoreV1().Pods(job.Namespace).Delete(podName, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// job phase: Terminating -> Terminated
		err = waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Terminating, vkv1.Terminated})
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: PodEvicted; Action: AbortJob", func() {
		By("init test context")
		jobName := "evicted-abort-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.AbortJobAction,
					Event:  vkv1.PodEvictedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name: "delete",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
			},
		})

		// job phase: pending -> running
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running})
		Expect(err).NotTo(HaveOccurred())

		By("delete one pod of job")
		podName := jobutil.MakePodName(job.Name, "delete", 0)
		err = context.kubeclient.CoreV1().Pods(context.namespace).Delete(podName, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// job phase: Aborting -> Aborted
		err = waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Aborting, vkv1.Aborted})
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: Any; Action: RestartJob", func() {
		By("init test context")
		jobName := "any-restart-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.RestartJobAction,
					Event:  vkv1.AnyEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "success",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
				{
					name: "delete",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
			},
		})

		// job phase: pending -> running
		err := waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Pending, vkv1.Running})
		Expect(err).NotTo(HaveOccurred())

		By("delete one pod of job")
		podName := jobutil.MakePodName(job.Name, "delete", 0)
		err = context.kubeclient.CoreV1().Pods(context.namespace).Delete(podName, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// job phase: Restarting -> Running
		err = waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Restarting, vkv1.Running})
		Expect(err).NotTo(HaveOccurred())
	})

	It("Job error handling: Restart job when job is unschedulable", func() {
		By("init test context")
		jobName := "job-restart-when-unschedulable"
		cleanupResources.Jobs = []string{jobName}

		rep := clusterSize(context, oneCPU)

		jobSpec := &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Event:  vkv1.JobUnknownEvent,
					Action: vkv1.RestartJobAction,
				},
			},
			tasks: []taskSpec{
				{
					name: "test",
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  rep,
					rep:  rep,
				},
			},
		}
		By("Create the Job")
		job := createJob(context, jobSpec)
		err := waitJobReady(context, job)
		Expect(err).NotTo(HaveOccurred())

		By("Taint all nodes")
		taints := []v1.Taint{
			{
				Key:    "unschedulable-taint-key",
				Value:  "unschedulable-taint-val",
				Effect: v1.TaintEffectNoSchedule,
			},
		}
		err = taintAllNodes(context, taints)
		Expect(err).NotTo(HaveOccurred())

		podName := jobutil.MakePodName(job.Name, "test", 0)
		By("Kill one of the pod in order to trigger unschedulable status")
		err = context.kubeclient.CoreV1().Pods(context.namespace).Delete(podName, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Job is restarting")
		err = waitJobPhases(context, job, []vkv1.JobPhase{
			vkv1.Restarting, vkv1.Pending})
		Expect(err).NotTo(HaveOccurred())

		By("Untaint all nodes")
		err = removeTaintsFromAllNodes(context, taints)
		Expect(err).NotTo(HaveOccurred())
		By("Job is running again")
		err = waitJobPhases(context, job, []vkv1.JobPhase{vkv1.Running})
		Expect(err).NotTo(HaveOccurred())
	})

	It("Job error handling: Abort job when job is unschedulable", func() {
		By("init test context")
		jobName := "job-abort-when-unschedulable"
		cleanupResources.Jobs = []string{jobName}

		rep := clusterSize(context, oneCPU)

		jobSpec := &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Event:  vkv1.JobUnknownEvent,
					Action: vkv1.AbortJobAction,
				},
			},
			tasks: []taskSpec{
				{
					name: "test",
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  rep,
					rep:  rep,
				},
			},
		}
		By("Create the Job")
		job := createJob(context, jobSpec)
		err := waitJobReady(context, job)
		Expect(err).NotTo(HaveOccurred())

		By("Taint all nodes")
		taints := []v1.Taint{
			{
				Key:    "unschedulable-taint-key",
				Value:  "unschedulable-taint-val",
				Effect: v1.TaintEffectNoSchedule,
			},
		}
		err = taintAllNodes(context, taints)
		Expect(err).NotTo(HaveOccurred())

		podName := jobutil.MakePodName(job.Name, "test", 0)
		By("Kill one of the pod in order to trigger unschedulable status")
		err = context.kubeclient.CoreV1().Pods(context.namespace).Delete(podName, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Job is aborted")
		err = waitJobPhases(context, job, []vkv1.JobPhase{
			vkv1.Aborting, vkv1.Aborted})
		Expect(err).NotTo(HaveOccurred())

		err = removeTaintsFromAllNodes(context, taints)
		Expect(err).NotTo(HaveOccurred())
	})

	It("job level LifecyclePolicy, Event: TaskCompleted; Action: CompletedJob", func() {
		By("init test context")
		jobName := "any-restart-job"
		cleanupResources.Jobs = []string{jobName}

		By("create job")
		job := createJob(context, &jobSpec{
			name: jobName,
			policies: []vkv1.LifecyclePolicy{
				{
					Action: vkv1.CompleteJobAction,
					Event:  vkv1.TaskCompletedEvent,
				},
			},
			tasks: []taskSpec{
				{
					name: "completed-task",
					img:  defaultBusyBoxImage,
					min:  2,
					rep:  2,
					//Sleep 5 seconds ensure job in running state
					command: "sleep 5",
				},
				{
					name: "terminating-task",
					img:  defaultNginxImage,
					min:  2,
					rep:  2,
				},
			},
		})

		By("job scheduled, then task 'completed_task' finished and job finally complete")
		// job phase: pending -> running -> completing -> completed
		err := waitJobStates(context, job, []vkv1.JobPhase{
			vkv1.Pending, vkv1.Running, vkv1.Completing, vkv1.Completed})
		Expect(err).NotTo(HaveOccurred())

	})

})

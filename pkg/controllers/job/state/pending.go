/*
Copyright 2017 The Volcano Authors.

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

package state

import (
	vcbatch "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	"volcano.sh/volcano/pkg/controllers/apis"
)

type pendingState struct {
	job *apis.JobInfo
}

func (ps *pendingState) Execute(action vcbatch.Action) error {
	switch action {
	case vcbatch.RestartJobAction:
		return KillJob(ps.job, func(status *vcbatch.JobStatus) bool {
			status.RetryCount++
			status.State.Phase = vcbatch.Restarting
			return true
		})

	case vcbatch.AbortJobAction:
		return KillJob(ps.job, func(status *vcbatch.JobStatus) bool {
			status.State.Phase = vcbatch.Aborting
			return true
		})
	case vcbatch.CompleteJobAction:
		return KillJob(ps.job, func(status *vcbatch.JobStatus) bool {
			status.State.Phase = vcbatch.Completing
			return true
		})
	case vcbatch.TerminateJobAction:
		return KillJob(ps.job, func(status *vcbatch.JobStatus) bool {
			status.State.Phase = vcbatch.Terminating
			return true
		})
	default:
		return SyncJob(ps.job, func(status *vcbatch.JobStatus) bool {
			phase := vcbatch.Pending

			if ps.job.Job.Spec.MinAvailable <= status.Running+status.Succeeded+status.Failed {
				phase = vcbatch.Running
			}

			status.State.Phase = phase
			return true
		})
	}
}

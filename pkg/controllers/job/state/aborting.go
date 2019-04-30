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
	vkv1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	"volcano.sh/volcano/pkg/controllers/apis"
)

type abortingState struct {
	job *apis.JobInfo
}

func (ps *abortingState) Execute(action vkv1.Action) error {
	switch action {
	case vkv1.ResumeJobAction:
		// Already in Restarting phase, just sync it
		return SyncJob(ps.job, func(status *vkv1.JobStatus) {
			status.State.Phase = vkv1.Restarting
			status.RetryCount++
		})
	default:
		return KillJob(ps.job, func(status *vkv1.JobStatus) {
			// If any "alive" pods, still in Aborting phase
			phase := vkv1.Aborted
			if status.Terminating != 0 || status.Pending != 0 || status.Running != 0 {
				phase = vkv1.Aborting
			}

			status.State.Phase = phase
		})
	}
}

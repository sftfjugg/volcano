/*
Copyright 2018 The Volcano Authors.

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

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Job struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of a cron job, including the minAvailable
	// +optional
	Spec JobSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Current status of Job
	// +optional
	Status JobStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// JobSpec describes how the job execution will look like and when it will actually run
type JobSpec struct {
	// SchedulerName is the default value of `tasks.template.spec.schedulerName`.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,1,opt,name=schedulerName"`

	// The minimal available pods to run for this Job
	// +optional
	MinAvailable int32 `json:"minAvailable,omitempty" protobuf:"bytes,2,opt,name=minAvailable"`

	// The volume mount for input of Job
	Input *VolumeSpec `json:"input,omitempty" protobuf:"bytes,3,opt,name=input"`

	// The volume mount for output of Job
	Output *VolumeSpec `json:"output,omitempty" protobuf:"bytes,4,opt,name=output"`

	// Tasks specifies the task specification of Job
	// +optional
	Tasks []TaskSpec `json:"tasks,omitempty" protobuf:"bytes,5,opt,name=tasks"`

	// Specifies the default lifecycle of tasks
	// +optional
	Policies []LifecyclePolicy `json:"policies,omitempty" protobuf:"bytes,6,opt,name=policies"`

	//Specifies the queue that will be used in the scheduler, "default" queue is used this leaves empty.
	Queue string `json:"queue,omitempty" protobuf:"bytes,7,opt,name=queue"`

	// Specifies the plugin of job
	// Key is plugin name, value is the arguments of the plugin
	// +optional
	Plugins map[string][]string `json:"plugins,omitempty" protobuf:"bytes,7,opt,name=plugins"`
}

// VolumeSpec defines the specification of Volume, e.g. PVC
type VolumeSpec struct {
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mountPath" protobuf:"bytes,1,opt,name=mountPath"`

	// VolumeClaim defines the PVC used by the VolumeMount.
	VolumeClaim *v1.PersistentVolumeClaimSpec `json:"volumeClaim,omitempty" protobuf:"bytes,1,opt,name=volumeClaim"`
}

type JobEvent string

const (
	CommandIssued JobEvent = "CommandIssued"
	PluginError   JobEvent = "PluginError"
)

// Event represent the phase of Job, e.g. pod-failed.
type Event string

const (
	// AllEvent means all event
	AnyEvent Event = "*"
	// PodFailedEvent is triggered if Pod was failed
	PodFailedEvent Event = "PodFailed"
	// PodEvictedEvent is triggered if Pod was deleted
	PodEvictedEvent Event = "PodEvicted"
	// These below are several events can lead to job 'Unknown'
	// 1. Task Unschedulable, this is triggered when part of
	//    pods can't be scheduled while some are already running in gang-scheduling case.
	JobUnknownEvent Event = "Unknown"

	// OutOfSyncEvent is triggered if Pod/Job were updated
	OutOfSyncEvent Event = "OutOfSync"
	// CommandIssuedEvent is triggered if a command is raised by user
	CommandIssuedEvent Event = "CommandIssued"
	// TaskCompletedEvent is triggered if the 'Replicas' amount of pods in one task are succeed
	TaskCompletedEvent Event = "TaskCompleted"
)

// Action is the action that Job controller will take according to the event.
type Action string

const (
	// AbortJobAction if this action is set, the whole job will be aborted:
	// all Pod of Job will be evicted, and no Pod will be recreated
	AbortJobAction Action = "AbortJob"
	// RestartJobAction if this action is set, the whole job will be restarted
	RestartJobAction Action = "RestartJob"
	// RestartTaskAction if this action is set, only the task will be restarted; default action.
	// This action can not work together with job level events, e.g. JobUnschedulable
	RestartTaskAction Action = "RestartTask"
	// TerminateJobAction if this action is set, the whole job wil be terminated
	// and can not be resumed: all Pod of Job will be evicted, and no Pod will be recreated.
	TerminateJobAction Action = "TerminateJob"
	// CompleteJobAction if this action is set, the unfinished pods will be killed, job completed.
	CompleteJobAction Action = "CompleteJob"

	// ResumeJobAction is the action to resume an aborted job.
	ResumeJobAction Action = "ResumeJob"
	// SyncJobAction is the action to sync Job/Pod status.
	SyncJobAction Action = "SyncJob"
)

// LifecyclePolicy specifies the lifecycle and error handling of task and job.
type LifecyclePolicy struct {
	// The action that will be taken to the PodGroup according to Event.
	// One of "Restart", "None".
	// Default to None.
	// +optional
	Action Action `json:"action,omitempty" protobuf:"bytes,1,opt,name=action"`

	// The Event recorded by scheduler; the controller takes actions
	// according to this Event.
	// +optional
	Event Event `json:"event,omitempty" protobuf:"bytes,2,opt,name=event"`

	// Timeout is the grace period for controller to take actions.
	// Default to nil (take action immediately).
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty" protobuf:"bytes,3,opt,name=timeout"`
}

// TaskSpec specifies the task specification of Job
type TaskSpec struct {
	// Name specifies the name of tasks
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Replicas specifies the replicas of this TaskSpec in Job
	Replicas int32 `json:"replicas,omitempty" protobuf:"bytes,2,opt,name=replicas"`

	// Specifies the pod that will be created for this TaskSpec
	// when executing a Job
	Template v1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`

	// Specifies the lifecycle of task
	// +optional
	Policies []LifecyclePolicy `json:"policies,omitempty" protobuf:"bytes,4,opt,name=policies"`
}

type JobPhase string

const (
	// Pending is the phase that job is pending in the queue, waiting for scheduling decision
	Pending JobPhase = "Pending"
	// Aborting is the phase that job is aborted, waiting for releasing pods
	Aborting JobPhase = "Aborting"
	// Aborted is the phase that job is aborted by user or error handling
	Aborted JobPhase = "Aborted"
	// Running is the phase that minimal available tasks of Job are running
	Running JobPhase = "Running"
	// Restarting is the phase that the Job is restarted, waiting for pod releasing and recreating
	Restarting JobPhase = "Restarting"
	// Completing is the phase that required tasks of job are completed, job starts to clean up
	Completing JobPhase = "Completing"
	// Completed is the phase that all tasks of Job are completed
	Completed JobPhase = "Completed"
	// Terminating is the phase that the Job is terminated, waiting for releasing pods
	Terminating JobPhase = "Terminating"
	// Terminated is the phase that the job is finished unexpected, e.g. events
	Terminated JobPhase = "Terminated"
)

// JobState contains details for the current state of the job.
type JobState struct {
	// The phase of Job.
	// +optional
	Phase JobPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`

	// Unique, one-word, CamelCase reason for the phase's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,2,opt,name=reason"`

	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
}

// JobStatus represents the current status of a Job
type JobStatus struct {
	// Current state of Job.
	State JobState `json:"state,omitempty" protobuf:"bytes,1,opt,name=state"`

	// The minimal available pods to run for this Job
	// +optional
	MinAvailable int32 `json:"minAvailable,omitempty" protobuf:"bytes,2,opt,name=minAvailable"`

	// The number of pending pods.
	// +optional
	Pending int32 `json:"pending,omitempty" protobuf:"bytes,3,opt,name=pending"`

	// The number of running pods.
	// +optional
	Running int32 `json:"running,omitempty" protobuf:"bytes,4,opt,name=running"`

	// The number of pods which reached phase Succeeded.
	// +optional
	Succeeded int32 `json:"Succeeded,omitempty" protobuf:"bytes,5,opt,name=succeeded"`

	// The number of pods which reached phase Failed.
	// +optional
	Failed int32 `json:"failed,omitempty" protobuf:"bytes,6,opt,name=failed"`

	// The number of pods which reached phase Terminating.
	// +optional
	Terminating int32 `json:"terminating,omitempty" protobuf:"bytes,7,opt,name=terminating"`
	//Current version of job
	Version int32 `json:"version,omitempty" protobuf:"bytes,8,opt,name=version"`
	// The resources that controlled by this job, e.g. Service, ConfigMap
	ControlledResources map[string]string `json:"controlledResources,omitempty" protobuf:"bytes,8,opt,name=controlledResources"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Job `json:"items" protobuf:"bytes,2,rep,name=items"`
}

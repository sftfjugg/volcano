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

package admission

import (
	"fmt"

	"github.com/golang/glog"

	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	v1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

const (
	AdmitJobPath  = "/jobs"
	MutateJobPath = "/mutating-jobs"
	PVCInputName  = "volcano.sh/job-input"
	PVCOutputName = "volcano.sh/job-output"
)

type AdmitFunc func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

var scheme = runtime.NewScheme()
var Codecs = serializer.NewCodecFactory(scheme)

func init() {
	addToScheme(scheme)
}

func addToScheme(scheme *runtime.Scheme) {
	corev1.AddToScheme(scheme)
	admissionregistrationv1beta1.AddToScheme(scheme)
}

func ToAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	glog.Error(err)
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func CheckPolicyDuplicate(policies []v1alpha1.LifecyclePolicy) (string, bool) {
	policyEvents := map[v1alpha1.Event]v1alpha1.Event{}
	hasDuplicate := false
	var duplicateInfo string

	for _, policy := range policies {
		if _, found := policyEvents[policy.Event]; found {
			hasDuplicate = true
			duplicateInfo = fmt.Sprintf("%v", policy.Event)
			break
		} else {
			policyEvents[policy.Event] = policy.Event
		}
	}

	if _, found := policyEvents[v1alpha1.AnyEvent]; found && len(policyEvents) > 1 {
		hasDuplicate = true
		duplicateInfo = "if there's * here, no other policy should be here"
	}

	return duplicateInfo, hasDuplicate
}

func DecodeJob(object runtime.RawExtension, resource metav1.GroupVersionResource) (v1alpha1.Job, error) {
	jobResource := metav1.GroupVersionResource{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "jobs"}
	raw := object.Raw
	job := v1alpha1.Job{}

	if resource != jobResource {
		err := fmt.Errorf("expect resource to be %s", jobResource)
		return job, err
	}

	deserializer := Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &job); err != nil {
		return job, err
	}
	glog.V(3).Infof("the job struct is %+v", job)

	return job, nil
}

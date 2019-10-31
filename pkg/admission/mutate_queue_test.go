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
	"encoding/json"
	"reflect"
	"testing"

	"volcano.sh/volcano/pkg/apis/scheduling/v1alpha2"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMutateQueues(t *testing.T) {
	stateNotSet := v1alpha2.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name: "normal-case-refresh-default-state",
		},
		Spec: v1alpha2.QueueSpec{
			Weight: 1,
		},
	}

	stateNotSetJSON, err := json.Marshal(stateNotSet)
	if err != nil {
		t.Errorf("Marshal queue without state set failed for %v", err)
	}

	openState := v1alpha2.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name: "normal-case-set-open",
		},
		Spec: v1alpha2.QueueSpec{
			Weight: 1,
			State:  v1alpha2.QueueStateOpen,
		},
	}

	openStateJSON, err := json.Marshal(openState)
	if err != nil {
		t.Errorf("Marshal queue with open state failed for %v", err)
	}

	pt := v1beta1.PatchTypeJSONPatch

	var refreshPatch []patchOperation
	refreshPatch = append(refreshPatch, patchOperation{
		Op:    "add",
		Path:  "/spec/state",
		Value: v1alpha2.QueueStateOpen,
	})

	refreshPatchJSON, err := json.Marshal(refreshPatch)
	if err != nil {
		t.Errorf("Marshal queue patch failed for %v", err)
	}

	var openStatePatch []patchOperation
	openStatePatchJSON, err := json.Marshal(openStatePatch)
	if err != nil {
		t.Errorf("Marshal null patch failed for %v", err)
	}

	testCases := []struct {
		Name           string
		AR             v1beta1.AdmissionReview
		reviewResponse *v1beta1.AdmissionResponse
	}{
		{
			Name: "Normal Case Refresh Default Open State For Queue",
			AR: v1beta1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1beta1",
				},
				Request: &v1beta1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   "scheduling.sigs.dev",
						Version: "v1alpha2",
						Kind:    "Queue",
					},
					Resource: metav1.GroupVersionResource{
						Group:    "scheduling.sigs.dev",
						Version:  "v1alpha2",
						Resource: "queues",
					},
					Name:      "normal-case-refresh-default-state",
					Operation: "CREATE",
					Object: runtime.RawExtension{
						Raw: stateNotSetJSON,
					},
				},
			},
			reviewResponse: &v1beta1.AdmissionResponse{
				Allowed:   true,
				PatchType: &pt,
				Patch:     refreshPatchJSON,
			},
		},
		{
			Name: "Normal Case Without Queue State Patch ",
			AR: v1beta1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1beta1",
				},
				Request: &v1beta1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   "scheduling.sigs.dev",
						Version: "v1alpha2",
						Kind:    "Queue",
					},
					Resource: metav1.GroupVersionResource{
						Group:    "scheduling.sigs.dev",
						Version:  "v1alpha2",
						Resource: "queues",
					},
					Name:      "normal-case-set-open",
					Operation: "CREATE",
					Object: runtime.RawExtension{
						Raw: openStateJSON,
					},
				},
			},
			reviewResponse: &v1beta1.AdmissionResponse{
				Allowed:   true,
				PatchType: &pt,
				Patch:     openStatePatchJSON,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			reviewResponse := MutateQueues(testCase.AR)
			if false == reflect.DeepEqual(reviewResponse, testCase.reviewResponse) {
				t.Errorf("Test case %s failed, expect %v, got %v", testCase.Name,
					reviewResponse, testCase.reviewResponse)
			}
		})
	}
}

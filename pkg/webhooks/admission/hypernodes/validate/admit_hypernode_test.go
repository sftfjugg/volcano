/*
Copyright 2024 The Volcano Authors.

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

package validate

import (
	"testing"

	admissionv1 "k8s.io/api/admission/v1"

	hypernodev1alpha1 "volcano.sh/apis/pkg/apis/topology/v1alpha1"
)

func TestValidateHyperNode(t *testing.T) {
	testCases := []struct {
		Name           string
		HyperNode      hypernodev1alpha1.HyperNode
		ExpectErr      bool
		reviewResponse admissionv1.AdmissionResponse
		ret            string
	}{
		{
			Name: "validate valid hypernode",
			HyperNode: hypernodev1alpha1.HyperNode{
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Members: []hypernodev1alpha1.MemberSpec{
						{
							Selector: hypernodev1alpha1.MemberSelector{
								Type:       hypernodev1alpha1.ExactMatchMemberSelectorType,
								ExactMatch: &hypernodev1alpha1.ExactMatch{Name: "node-1"},
							},
						},
					},
				},
			},
			ExpectErr:      false,
			reviewResponse: admissionv1.AdmissionResponse{},
		},
		{
			Name: "validate invalid hypernode with empty exactMatch",
			HyperNode: hypernodev1alpha1.HyperNode{
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Members: []hypernodev1alpha1.MemberSpec{
						{
							Selector: hypernodev1alpha1.MemberSelector{
								Type:       hypernodev1alpha1.ExactMatchMemberSelectorType,
								ExactMatch: &hypernodev1alpha1.ExactMatch{Name: ""},
							},
						},
					},
				},
			},
			ExpectErr:      true,
			reviewResponse: admissionv1.AdmissionResponse{},
		},
		{
			Name: "validate invalid hypernode with empty regexMatch",
			HyperNode: hypernodev1alpha1.HyperNode{
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Members: []hypernodev1alpha1.MemberSpec{
						{
							Selector: hypernodev1alpha1.MemberSelector{
								Type: hypernodev1alpha1.RegexMatchMemberSelectorType,
								RegexMatch: &hypernodev1alpha1.RegexMatch{
									Pattern: "",
								},
							},
						},
					},
				},
			},
			ExpectErr:      true,
			reviewResponse: admissionv1.AdmissionResponse{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			err := validateHyperNode(&testCase.HyperNode)
			if err != nil {
				t.Errorf("validateHyperNode failed: %v", err)
			}
		})
	}

}

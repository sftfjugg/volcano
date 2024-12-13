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
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	whv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/klog/v2"

	hypernodev1alpha1 "volcano.sh/apis/pkg/apis/topology/v1alpha1"
	"volcano.sh/volcano/pkg/webhooks/router"
	"volcano.sh/volcano/pkg/webhooks/schema"
	"volcano.sh/volcano/pkg/webhooks/util"
)

func init() {
	router.RegisterAdmission(service)
}

var config = &router.AdmissionServiceConfig{}

var service = &router.AdmissionService{
	Path: "/hypernodes/validate",
	Func: AdmitHyperNode,

	Config: config,

	ValidatingConfig: &whv1.ValidatingWebhookConfiguration{
		Webhooks: []whv1.ValidatingWebhook{{
			Name: "validatehypernode.volcano.sh",
			Rules: []whv1.RuleWithOperations{
				{
					Operations: []whv1.OperationType{whv1.Create, whv1.Update},
					Rule: whv1.Rule{
						APIGroups:   []string{hypernodev1alpha1.SchemeGroupVersion.Group},
						APIVersions: []string{hypernodev1alpha1.SchemeGroupVersion.Version},
						Resources:   []string{"hypernodes"},
					},
				},
			},
		}},
	},
}

// AdmitHyperNode is to admit hypernode and return response.
func AdmitHyperNode(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	klog.V(3).Infof("admitting hypernode -- %s", ar.Request.Operation)

	hypernode, err := schema.DecodeHyperNode(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return util.ToAdmissionResponse(err)
	}

	switch ar.Request.Operation {
	case admissionv1.Create, admissionv1.Update:
		err = validateHyperNode(hypernode)
		if err != nil {
			return util.ToAdmissionResponse(err)
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

// validateHyperNode is to validate hypernode.
func validateHyperNode(hypernode *hypernodev1alpha1.HyperNode) error {
	if len(hypernode.Spec.Members) == 0 {
		return nil
	}

	for _, member := range hypernode.Spec.Members {
		if member.Selector.Type == "" {
			continue
		}

		if member.Selector.Type == hypernodev1alpha1.ExactMatchMemberSelectorType && member.Selector.ExactMatch == nil {
			return fmt.Errorf("exactMatch is required when type is Exact")
		}
		if member.Selector.Type == hypernodev1alpha1.RegexMatchMemberSelectorType && member.Selector.RegexMatch == nil {
			return fmt.Errorf("regexMatch is required when type is Regex")
		}
	}
	return nil
}

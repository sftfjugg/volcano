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
	"regexp"

	admissionv1 "k8s.io/api/admission/v1"
	whv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

// validateHyperNodeName is to validate hypernode name.
func validateHyperNodeName(value string, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if errMsgs := validation.IsQualifiedName(value); len(errMsgs) > 0 {
		err := field.Invalid(fldPath, value, fmt.Sprintf("hypernode name must validate failed %v", errMsgs))
		errs = append(errs, err)
	}
	return errs
}

// validateHyperNodeMemberSelector is to validate hypernode member selector.
func validateHyperNodeMemberSelector(selector hypernodev1alpha1.MemberSelector, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	switch selector.Type {
	case hypernodev1alpha1.ExactMatchMemberSelectorType:
		if selector.ExactMatch == nil {
			err := field.Invalid(fldPath.Child("exactMatch"), selector.ExactMatch,
				fmt.Sprintf("exactMatch is required when type is Exact"))
			errs = append(errs, err)
		}
		if errMsgs := validation.IsQualifiedName(selector.ExactMatch.Name); len(errMsgs) > 0 {
			err := field.Invalid(fldPath.Child("exactMatch").Child("name"),
				selector.ExactMatch.Name, fmt.Sprintf("member exactMatch validate failed %v", errMsgs))
			errs = append(errs, err)
		}
	case hypernodev1alpha1.RegexMatchMemberSelectorType:
		if selector.RegexMatch == nil {
			err := field.Invalid(fldPath, selector.RegexMatch,
				fmt.Sprintf("regexMatch is required when type is Regex"))
			errs = append(errs, err)
		}
		if selector.RegexMatch.Pattern == "" {
			err := field.Invalid(fldPath.Child("regexMatch").Child("pattern"),
				selector.RegexMatch.Pattern, "member regexMatch pattern is required")
			errs = append(errs, err)
		}
		if _, err := regexp.Compile(selector.RegexMatch.Pattern); err != nil {
			err := field.Invalid(fldPath.Child("regexMatch").Child("pattern"),
				selector.RegexMatch.Pattern, fmt.Sprintf("member regexMatch pattern is invalid: %v", err))
			errs = append(errs, err)
		}
	default:
		err := field.Invalid(fldPath.Child("type"), selector.Type, fmt.Sprintf("unknown type %s", selector.Type))
		errs = append(errs, err)
	}

	return errs
}

// validateHyperNodeMemberType is to validate hypernode member type.
func validateHyperNodeMemberType(memberType hypernodev1alpha1.MemberType, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	switch memberType {
	case hypernodev1alpha1.MemberTypeNode, "HyperNode":
	default:
		err := field.Invalid(fldPath, memberType, fmt.Sprintf("unknown member type %s", memberType))
		errs = append(errs, err)
	}
	return errs
}

// validateHyperNode is to validate hypernode.
func validateHyperNode(hypernode *hypernodev1alpha1.HyperNode) error {

	errs := field.ErrorList{}
	resourcePath := field.NewPath("")

	errs = append(errs, validateHyperNodeName(hypernode.Name,
		resourcePath.Child("metadata").Child("name"))...)

	for _, member := range hypernode.Spec.Members {
		errs = append(errs, validateHyperNodeMemberType(member.Type,
			resourcePath.Child("spec").Child("members").Child("type"))...)
		errs = append(errs, validateHyperNodeMemberSelector(member.Selector,
			resourcePath.Child("spec").Child("members").Child("selector"))...)
	}

	if len(errs) > 0 {
		return errs.ToAggregate()
	}

	return nil
}

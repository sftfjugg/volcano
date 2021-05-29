/*
Copyright 2021 The Volcano Authors.

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

package policy

import "k8s.io/klog"

type policyBestEffort struct {
	numaNodes []int
}

// NewPolicyBestEffort return a new policy interface
func NewPolicyBestEffort(numaNodes []int) Policy {
	return &policyBestEffort{numaNodes: numaNodes}
}

func (p *policyBestEffort) canAdmitPodResult(hint *TopologyHint) bool {
	return true
}

func (p *policyBestEffort) Predicate(providersHints []map[string][]TopologyHint) (TopologyHint, bool) {
	filteredProvidersHints := filterProvidersHints(providersHints)
	bestHint := mergeFilteredHints(p.numaNodes, filteredProvidersHints)
	admit := p.canAdmitPodResult(&bestHint)

	klog.V(4).Infof("bestHint: %v admit %v\n", bestHint, admit)
	return bestHint, admit
}

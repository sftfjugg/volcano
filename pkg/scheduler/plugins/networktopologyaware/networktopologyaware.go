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

package networktopologyaware

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/util"
)

const (
	// PluginName indicates name of volcano scheduler plugin.
	PluginName = "networktopologyaware"
)

type networkTopologyAwarePlugin struct{}

// New function returns prioritizePlugin object
func New(aruguments framework.Arguments) framework.Plugin {
	return &networkTopologyAwarePlugin{}
}

func (nta *networkTopologyAwarePlugin) Name() string {
	return PluginName
}

func (nta *networkTopologyAwarePlugin) OnSessionOpen(ssn *framework.Session) {
	klog.V(5).Infof("Enter networkTopologyAwarePlugin plugin ...")
	defer func() {
		klog.V(5).Infof("Leaving networkTopologyAware plugin ...")
	}()
	ntaFn := func(jobInfo *api.JobInfo, hyperNodes map[string][]*api.NodeInfo, hyperNodesTiers []int, hyperNodesListByTier map[int][]string, hyperNodesMap map[string]sets.Set[string]) (map[string]float64, error) {
		hyperNodeScores := make(map[string]float64)
		for hyperNode := range hyperNodes {
			score := networkTopologyAwareScore(hyperNode, jobInfo, hyperNodesTiers, hyperNodesListByTier, hyperNodesMap)
			hyperNodeScores[hyperNode] = score
		}

		klog.V(4).Infof("networkTopologyAware score for job %s is: %v", jobInfo.Name, hyperNodeScores)
		return hyperNodeScores, nil
	}
	ssn.AddHyperNodeOrederFn(nta.Name(), ntaFn)
}

func (bp *networkTopologyAwarePlugin) OnSessionClose(ssn *framework.Session) {
}

// networkTopologyAwareScore use the best fit polices during scheduling.

// Explanation:
// The RootHypernode property of a job is the hypernode that serves as the smallest root in the hypernode tree.
// A job has multiple tasks, each belonging to a hypernode. This RootHypernode is the topmost and lowest common ancestor among the hypernodes of all tasks within the job.

// Goals:
// - The tier to which the rootHypernode of a job belongs should be as low as possible.
func networkTopologyAwareScore(hyperNode string, job *api.JobInfo, hyperNodesTiers []int, hyperNodesListByTier map[int][]string, hyperNodesMap map[string]sets.Set[string]) float64 {
	score := 0.0
	sumTier := 0
	for _, tier := range hyperNodesTiers {
		sumTier += tier
	}

	rootHypernode, index := util.FindOutRootHyperNode(hyperNode, job, hyperNodesTiers, hyperNodesListByTier, hyperNodesMap)
	if index == -1 {
		return score
	}
	// hyperNode is the rootHypernode
	if job.RootHyperNode != "" && rootHypernode == hyperNode {
		return 1.0
	}

	for tier, nodes := range hyperNodesListByTier {
		for _, node := range nodes {
			if node == rootHypernode {
				tierWeight := 1 - float64(tier)/float64(sumTier)
				return score + tierWeight
			}
		}
	}
	return score
}

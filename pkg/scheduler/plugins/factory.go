/*
Copyright 2019 The Kubernetes Authors.

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

package plugins

import (
	"volcano.sh/volcano/pkg/scheduler/framework"

	"volcano.sh/volcano/pkg/scheduler/plugins/conformance"
	"volcano.sh/volcano/pkg/scheduler/plugins/drf"
	"volcano.sh/volcano/pkg/scheduler/plugins/gang"
	"volcano.sh/volcano/pkg/scheduler/plugins/nodeorder"
	"volcano.sh/volcano/pkg/scheduler/plugins/predicates"
	"volcano.sh/volcano/pkg/scheduler/plugins/priority"
	"volcano.sh/volcano/pkg/scheduler/plugins/proportion"
)

func init() {
	// Plugins for Jobs
	framework.RegisterPluginBuilder("drf", drf.New)
	framework.RegisterPluginBuilder("gang", gang.New)
	framework.RegisterPluginBuilder("predicates", predicates.New)
	framework.RegisterPluginBuilder("priority", priority.New)
	framework.RegisterPluginBuilder("nodeorder", nodeorder.New)
	framework.RegisterPluginBuilder("conformance", conformance.New)

	// Plugins for Queues
	framework.RegisterPluginBuilder("proportion", proportion.New)
}

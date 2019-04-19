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

package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

const defaultWorkers = 5

// ServerOption is the main context object for the controller manager.
type ServerOption struct {
	Master               string
	Kubeconfig           string
	EnableLeaderElection bool
	LockObjectNamespace  string
	// WorkerThreads is the number of threads syncing operations
	// that will be done concurrently. Larger number = faster job updating,
	// but more CPU (and network) load.
	WorkerThreads int32
}

// NewServerOption creates a new CMServer with a default config.
func NewServerOption() *ServerOption {
	s := ServerOption{
		// default setting
		WorkerThreads: defaultWorkers,
	}
	return &s
}

// AddFlags adds flags for a specific CMServer to the specified FlagSet
func (s *ServerOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")
	fs.BoolVar(&s.EnableLeaderElection, "leader-elect", s.EnableLeaderElection, "Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated kar-scheduler for high availability.")
	fs.StringVar(&s.LockObjectNamespace, "lock-object-namespace", s.LockObjectNamespace, "Define the namespace of the lock object.")
	fs.Int32Var(&s.WorkerThreads, "worker-threads", s.WorkerThreads, "The number of job syncing operations that will be done concurrently. "+
		"Larger number = faster job updating, but more CPU (and network) load")
}

func (s *ServerOption) CheckOptionOrDie() error {
	if s.EnableLeaderElection && s.LockObjectNamespace == "" {
		return fmt.Errorf("lock-object-namespace must not be nil when LeaderElection is enabled")
	}

	if s.WorkerThreads <= 0 {
		return fmt.Errorf("worker-threads must be greater than 0")
	}

	return nil
}

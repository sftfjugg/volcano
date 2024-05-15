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

package options

import (
	"reflect"
	"testing"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/component-base/config"
	componentbaseoptions "k8s.io/component-base/config/options"

	"volcano.sh/volcano/pkg/kube"
	commonutil "volcano.sh/volcano/pkg/util"
)

func TestAddFlags(t *testing.T) {
	fs := pflag.NewFlagSet("addflagstest", pflag.ContinueOnError)
	s := NewServerOption()

	commonutil.LeaderElectionDefault(&s.LeaderElection)
	s.LeaderElection.ResourceName = "vc-controller-manager"
	componentbaseoptions.BindLeaderElectionFlags(&s.LeaderElection, fs)
	s.AddFlags(fs)

	args := []string{
		"--master=127.0.0.1",
		"--kube-api-burst=200",
		"--scheduler-name=volcano",
		"--scheduler-name=volcano2",
		"--leader-elect-lease-duration=60s",
		"--leader-elect-renew-deadline=20s",
		"--leader-elect-retry-period=10s",
	}
	fs.Parse(args)

	// This is a snapshot of expected options parsed by args.
	expected := &ServerOption{
		KubeClientOptions: kube.ClientOptions{
			Master:     "127.0.0.1",
			KubeConfig: "",
			QPS:        defaultQPS,
			Burst:      200,
		},
		PrintVersion:            false,
		WorkerThreads:           defaultWorkers,
		SchedulerNames:          []string{"volcano", "volcano2"},
		MaxRequeueNum:           defaultMaxRequeueNum,
		HealthzBindAddress:      ":11251",
		InheritOwnerAnnotations: true,
		LeaderElection: config.LeaderElectionConfiguration{
			LeaderElect:       true,
			LeaseDuration:     metav1.Duration{60 * time.Second},
			RenewDeadline:     metav1.Duration{20 * time.Second},
			RetryPeriod:       metav1.Duration{10 * time.Second},
			ResourceLock:      resourcelock.LeasesResourceLock,
			ResourceNamespace: defaultLockObjectNamespace,
			ResourceName:      "vc-controller-manager",
		},
		LockObjectNamespace: defaultLockObjectNamespace,
		WorkerThreadsForPG:  5,
	}

	if !reflect.DeepEqual(expected, s) {
		t.Errorf("Got different run options than expected.\nGot: %+v\nExpected: %+v\n", s, expected)
	}

	err := s.CheckOptionOrDie()
	if err != nil {
		t.Errorf("expected nil but got %v\n", err)
	}

}

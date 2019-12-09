/*
Copyright 2018 The Volcano Authors.

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
package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/flag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"volcano.sh/volcano/cmd/admission/app"
	"volcano.sh/volcano/cmd/admission/app/options"
	"volcano.sh/volcano/pkg/admission/router"
	"volcano.sh/volcano/pkg/version"

	_ "volcano.sh/volcano/pkg/admission/jobs/mutate"
	_ "volcano.sh/volcano/pkg/admission/jobs/validate"
	_ "volcano.sh/volcano/pkg/admission/pods"
)

var logFlushFreq = pflag.Duration("log-flush-frequency", 5*time.Second, "Maximum number of seconds between log flushes")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	klog.InitFlags(nil)

	config := options.NewConfig()
	config.AddFlags(pflag.CommandLine)

	flag.InitFlags()

	if config.PrintVersion {
		version.PrintVersionAndExit()
	}

	go wait.Until(klog.Flush, *logFlushFreq, wait.NeverStop)
	defer klog.Flush()

	if err := config.CheckPortOrDie(); err != nil {
		klog.Fatalf("Configured port is invalid: %v", err)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.Kubeconfig)
	if err != nil {
		klog.Fatalf("Unable to build k8s config: %v", err)
	}

	vClient := app.GetVolcanoClient(restConfig)
	router.ForEachAdmission(func(service *router.AdmissionService) {
		if service.Config != nil {
			service.Config.VolcanoClient = vClient
			service.Config.SchedulerName = config.SchedulerName
		}
		http.HandleFunc(service.Path, service.Handler)
	})

	//
	//caBundle, err := ioutil.ReadFile(config.CaCertFile)
	//if err != nil {
	//	klog.Fatalf("Unable to read cacert file: %v", err)
	//}
	////
	////err = options.RegisterWebhooks(config, app.GetClient(restConfig), caBundle)
	////if err != nil {
	////	klog.Fatalf("Unable to register webhook configs: %v", err)
	////}

	webhookServeError := make(chan struct{})
	stopChannel := make(chan os.Signal)
	signal.Notify(stopChannel, syscall.SIGTERM, syscall.SIGINT)

	server := &http.Server{
		Addr:      ":" + strconv.Itoa(config.Port),
		TLSConfig: app.ConfigTLS(config, restConfig),
	}
	go func() {
		err = server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			klog.Fatalf("ListenAndServeTLS for admission webhook failed: %v", err)
			close(webhookServeError)
		}
	}()

	select {
	case <-stopChannel:
		if err := server.Close(); err != nil {
			klog.Fatalf("Close admission server failed: %v", err)
		}
		return
	case <-webhookServeError:
		return
	}
}

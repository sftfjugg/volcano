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

package podgroup

import (
	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kbv1 "volcano.sh/volcano/pkg/apis/scheduling/v1alpha1"
	kbver "volcano.sh/volcano/pkg/client/clientset/versioned"
	kbinfoext "volcano.sh/volcano/pkg/client/informers/externalversions"
	kbinfo "volcano.sh/volcano/pkg/client/informers/externalversions/scheduling/v1alpha1"
	kblister "volcano.sh/volcano/pkg/client/listers/scheduling/v1alpha1"
)

// Controller the Podgroup Controller type
type Controller struct {
	kubeClients kubernetes.Interface
	kbClients   kbver.Interface

	podInformer     coreinformers.PodInformer
	pgInformer      kbinfo.PodGroupInformer
	sharedInformers informers.SharedInformerFactory

	// A store of pods
	podLister corelisters.PodLister
	podSynced func() bool

	// A store of podgroups
	pgLister kblister.PodGroupLister
	pgSynced func() bool

	queue workqueue.RateLimitingInterface
}

// NewPodgroupController create new Podgroup Controller
func NewPodgroupController(
	kubeClient kubernetes.Interface,
	kbClient kbver.Interface,
	schedulerName string,
) *Controller {
	cc := &Controller{
		kubeClients: kubeClient,
		kbClients:   kbClient,

		queue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	cc.sharedInformers = informers.NewSharedInformerFactory(cc.kubeClients, 0)
	cc.podInformer = cc.sharedInformers.Core().V1().Pods()
	cc.podLister = cc.podInformer.Lister()
	cc.podSynced = cc.podInformer.Informer().HasSynced
	cc.podInformer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch obj.(type) {
				case *v1.Pod:
					pod := obj.(*v1.Pod)
					if pod.Spec.SchedulerName == schedulerName && len(pod.OwnerReferences) != 0 &&
						(pod.Annotations == nil || pod.Annotations[kbv1.GroupNameAnnotationKey] == "") {
						return true
					}
					return false
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: cc.addPod,
			},
		})

	cc.pgInformer = kbinfoext.NewSharedInformerFactory(cc.kbClients, 0).Scheduling().V1alpha1().PodGroups()
	cc.pgLister = cc.pgInformer.Lister()
	cc.pgSynced = cc.pgInformer.Informer().HasSynced

	return cc
}

// Run start NewPodgroupController
func (cc *Controller) Run(stopCh <-chan struct{}) {
	go cc.sharedInformers.Start(stopCh)
	go cc.podInformer.Informer().Run(stopCh)
	go cc.pgInformer.Informer().Run(stopCh)

	cache.WaitForCacheSync(stopCh, cc.podSynced, cc.pgSynced)

	go wait.Until(cc.worker, 0, stopCh)

	glog.Infof("PodgroupController is running ...... ")
}

func (cc *Controller) worker() {
	for cc.processNextReq() {
	}
}

func (cc *Controller) processNextReq() bool {
	obj, shutdown := cc.queue.Get()
	if shutdown {
		glog.Errorf("Fail to pop item from queue")
		return false
	}

	req := obj.(podRequest)
	defer cc.queue.Done(req)

	pod, err := cc.podLister.Pods(req.pod.Namespace).Get(req.pod.Name)
	if err != nil {
		glog.Errorf("Failed to get pod by <%v> from cache: %v", req, err)
		return true
	}

	// normal pod use volcano
	if err := cc.createNormalPodPGIfNotExist(pod); err != nil {
		glog.Errorf("Failed to handle Pod <%s/%s>: %v", pod.Namespace, pod.Name, err)
		cc.queue.AddRateLimited(req)
		return true
	}

	// If no error, forget it.
	cc.queue.Forget(req)

	return true
}

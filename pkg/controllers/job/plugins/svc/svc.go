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

package svc

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	batch "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	"volcano.sh/volcano/pkg/apis/helpers"
	jobhelpers "volcano.sh/volcano/pkg/controllers/job/helpers"
	"volcano.sh/volcano/pkg/controllers/job/plugins/interface"
)

type servicePlugin struct {
	// Arguments given for the plugin
	pluginArguments []string

	Clientset pluginsinterface.PluginClientset
}

// New creates service plugin
func New(client pluginsinterface.PluginClientset, arguments []string) pluginsinterface.PluginInterface {
	servicePlugin := servicePlugin{pluginArguments: arguments, Clientset: client}

	return &servicePlugin
}

func (sp *servicePlugin) Name() string {
	return "svc"
}

func (sp *servicePlugin) OnPodCreate(pod *v1.Pod, job *batch.Job) error {
	// use podName.serviceName as default pod DNS domain
	if len(pod.Spec.Hostname) == 0 {
		pod.Spec.Hostname = pod.Name
	}
	if len(pod.Spec.Subdomain) == 0 {
		pod.Spec.Subdomain = job.Name
	}

	sp.mountConfigmap(pod, job)

	return nil
}

func (sp *servicePlugin) OnJobAdd(job *batch.Job) error {
	if job.Status.ControlledResources["plugin-"+sp.Name()] == sp.Name() {
		return nil
	}

	data := GenerateHost(job)

	if err := helpers.CreateConfigMapIfNotExist(job, sp.Clientset.KubeClients, data, sp.cmName(job)); err != nil {
		return err
	}

	if err := CreateServiceIfNotExist(sp.Clientset.KubeClients, job); err != nil {
		return err
	}

	job.Status.ControlledResources["plugin-"+sp.Name()] = sp.Name()

	return nil
}

func (sp *servicePlugin) OnJobDelete(job *batch.Job) error {
	if err := helpers.DeleteConfigmap(job, sp.Clientset.KubeClients, sp.cmName(job)); err != nil {
		return err
	}

	if err := sp.Clientset.KubeClients.CoreV1().Services(job.Namespace).Delete(job.Name, nil); err != nil {
		if !apierrors.IsNotFound(err) {
			glog.Errorf("Failed to delete Service of Job %v/%v: %v", job.Namespace, job.Name, err)
			return err
		}
	}

	return nil
}

func (sp *servicePlugin) mountConfigmap(pod *v1.Pod, job *batch.Job) {
	cmName := sp.cmName(job)
	cmVolume := v1.Volume{
		Name: cmName,
	}
	cmVolume.ConfigMap = &v1.ConfigMapVolumeSource{
		LocalObjectReference: v1.LocalObjectReference{
			Name: cmName,
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, cmVolume)

	for i, c := range pod.Spec.Containers {
		vm := v1.VolumeMount{
			MountPath: ConfigMapMountPath,
			Name:      cmName,
		}

		pod.Spec.Containers[i].VolumeMounts = append(c.VolumeMounts, vm)
	}
}

func CreateServiceIfNotExist(kubeClient kubernetes.Interface, job *batch.Job) error {
	// If Service does not exist, create one for Job.
	if _, err := kubeClient.CoreV1().Services(job.Namespace).Get(job.Name, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			glog.V(3).Infof("Failed to get Service for Job <%s/%s>: %v",
				job.Namespace, job.Name, err)
			return err
		}

		svc := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: job.Namespace,
				Name:      job.Name,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(job, helpers.JobKind),
				},
			},
			Spec: v1.ServiceSpec{
				ClusterIP: "None",
				Selector: map[string]string{
					batch.JobNameKey:      job.Name,
					batch.JobNamespaceKey: job.Namespace,
				},
				Ports: []v1.ServicePort{
					{
						Name:       "placeholder-volcano",
						Port:       1,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(1),
					},
				},
			},
		}

		if _, e := kubeClient.CoreV1().Services(job.Namespace).Create(svc); e != nil {
			glog.V(3).Infof("Failed to create Service for Job <%s/%s>: %v", job.Namespace, job.Name, e)
			return e
		}

	}

	return nil
}

func (sp *servicePlugin) cmName(job *batch.Job) string {
	return fmt.Sprintf("%s-%s", job.Name, sp.Name())
}

func GenerateHost(job *batch.Job) map[string]string {
	data := make(map[string]string, len(job.Spec.Tasks))

	for _, ts := range job.Spec.Tasks {
		hosts := make([]string, 0, ts.Replicas)

		for i := 0; i < int(ts.Replicas); i++ {
			hostName := ts.Template.Spec.Hostname
			subdomain := ts.Template.Spec.Subdomain
			if len(hostName) == 0 {
				hostName = jobhelpers.MakePodName(job.Name, ts.Name, i)
			}
			if len(subdomain) == 0 {
				subdomain = job.Name
			}
			hosts = append(hosts, hostName+"."+subdomain)
			if len(ts.Template.Spec.Hostname) != 0 {
				break
			}
		}

		key := fmt.Sprintf(ConfigMapTaskHostFmt, ts.Name)
		data[key] = strings.Join(hosts, "\n")
	}

	return data
}

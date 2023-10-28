/*
 Copyright 2023 The Volcano Authors.

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

package source

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const NODE_METRICS_PERIOD = "10m"

type NodeMetrics struct {
	MetricsTime time.Time
	CPU         float64
	Memory      float64
}

type MetricsClient interface {
	NodesMetricsAvg(ctx context.Context, nodeMetricsMap map[string]*NodeMetrics) error
}

func NewMetricsClient(restConfig *rest.Config, metricsConf map[string]string) (MetricsClient, error) {
	klog.V(3).Infof("New metrics client begin, resconfig is %v, metricsConf is %v", restConfig, metricsConf)
	metricsType := metricsConf["type"]
	if metricsType == "elasticsearch" {
		return NewElasticsearchMetricsClient(metricsConf)
	} else if metricsType == "prometheus" {
		return NewPrometheusMetricsClient(metricsConf)
	} else if metricsType == "prometheus_adapt" {
		return NewCustomMetricsClient(restConfig)
	} else {
		return nil, fmt.Errorf("Data cannot be collected from the %s monitoring system. "+
			"The supported monitoring systems are elasticsearch, prometheus, and prometheus_adapt.", metricsType)
	}
}

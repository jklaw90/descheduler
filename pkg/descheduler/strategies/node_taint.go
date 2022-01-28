/*
Copyright 2017 The Kubernetes Authors.

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

package strategies

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/descheduler/pkg/api"
	"sigs.k8s.io/descheduler/pkg/descheduler/evictions"
	podutil "sigs.k8s.io/descheduler/pkg/descheduler/pod"
	"sigs.k8s.io/descheduler/pkg/utils"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func validateRemovePodsViolatingNodeTaintsParams(params *api.StrategyParameters) error {
	if params == nil {
		return nil
	}

	// At most one of include/exclude can be set
	if params.Namespaces != nil && len(params.Namespaces.Include) > 0 && len(params.Namespaces.Exclude) > 0 {
		return fmt.Errorf("only one of Include/Exclude namespaces can be set")
	}
	if params.ThresholdPriority != nil && params.ThresholdPriorityClassName != "" {
		return fmt.Errorf("only one of thresholdPriority and thresholdPriorityClassName can be set")
	}

	return nil
}

type RemovePodsViolatingNodeTaintsStrategy struct {
	strategy api.DeschedulerStrategy
	client   clientset.Interface
}

func NewRemovePodsViolatingNodeTaintsStrategy(client clientset.Interface, strategyList api.StrategyList) (*RemovePodsViolatingNodeTaintsStrategy, error) {
	s := &RemovePodsViolatingNodeTaintsStrategy{}
	strategy, ok := strategyList[s.Name()]
	if !ok {
		return nil, errors.New("")
	}
	s.strategy = strategy

	return s, nil
}

func (s *RemovePodsViolatingNodeTaintsStrategy) Name() api.StrategyName {
	return RemovePodsViolatingNodeTaints
}

func (s *RemovePodsViolatingNodeTaintsStrategy) Enabled() bool {
	return s.strategy.Enabled
}

func (s *RemovePodsViolatingNodeTaintsStrategy) Validate() error {
	err := validateRemovePodsViolatingNodeTaintsParams(s.strategy.Params)
	if err != nil {
		klog.ErrorS(err, "Invalid RemovePodsViolatingNodeTaints parameters")
		return err
	}
	return nil
}

// Run evicts pods on the node which violate NoSchedule Taints on nodes
func (s *RemovePodsViolatingNodeTaintsStrategy) Run(ctx context.Context, client clientset.Interface, strategy api.DeschedulerStrategy, nodes []*v1.Node, podEvictor *evictions.PodEvictor, getPodsAssignedToNode podutil.GetPodsAssignedToNodeFunc) {
	if err := validateRemovePodsViolatingNodeTaintsParams(strategy.Params); err != nil {
		klog.ErrorS(err, "Invalid RemovePodsViolatingNodeTaints parameters")
		return
	}

	var includedNamespaces, excludedNamespaces sets.String
	var labelSelector *metav1.LabelSelector
	if strategy.Params != nil {
		if strategy.Params.Namespaces != nil {
			includedNamespaces = sets.NewString(strategy.Params.Namespaces.Include...)
			excludedNamespaces = sets.NewString(strategy.Params.Namespaces.Exclude...)
		}
		labelSelector = strategy.Params.LabelSelector
	}

	thresholdPriority, err := utils.GetPriorityFromStrategyParams(ctx, client, strategy.Params)
	if err != nil {
		klog.ErrorS(err, "Failed to get threshold priority from strategy's params")
		return
	}

	nodeFit := false
	if strategy.Params != nil {
		nodeFit = strategy.Params.NodeFit
	}

	evictable := podEvictor.Evictable(evictions.WithPriorityThreshold(thresholdPriority), evictions.WithNodeFit(nodeFit))

	podFilter, err := podutil.NewOptions().
		WithFilter(evictable.IsEvictable).
		WithNamespaces(includedNamespaces).
		WithoutNamespaces(excludedNamespaces).
		WithLabelSelector(labelSelector).
		BuildFilterFunc()
	if err != nil {
		klog.ErrorS(err, "Error initializing pod filter function")
		return
	}

	for _, node := range nodes {
		klog.V(1).InfoS("Processing node", "node", klog.KObj(node))
		pods, err := podutil.ListAllPodsOnANode(node.Name, getPodsAssignedToNode, podFilter)
		if err != nil {
			//no pods evicted as error encountered retrieving evictable Pods
			return
		}
		totalPods := len(pods)
		for i := 0; i < totalPods; i++ {
			if !utils.TolerationsTolerateTaintsWithFilter(
				pods[i].Spec.Tolerations,
				node.Spec.Taints,
				func(taint *v1.Taint) bool { return taint.Effect == v1.TaintEffectNoSchedule },
			) {
				klog.V(2).InfoS("Not all taints with NoSchedule effect are tolerated after update for pod on node", "pod", klog.KObj(pods[i]), "node", klog.KObj(node))
				if _, err := podEvictor.EvictPod(ctx, pods[i], node, "NodeTaint"); err != nil {
					klog.ErrorS(err, "Error evicting pod")
					break
				}
			}
		}
	}
}

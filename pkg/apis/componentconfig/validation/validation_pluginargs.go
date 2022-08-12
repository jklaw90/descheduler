/*
Copyright 2022 The Kubernetes Authors.

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

package validation

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/descheduler/pkg/api"
	"sigs.k8s.io/descheduler/pkg/apis/componentconfig"
)

// ValidateRemovePodsViolatingNodeTaintsArgs validates RemovePodsViolatingNodeTaints arguments
func ValidateRemovePodsViolatingNodeTaintsArgs(args *componentconfig.RemovePodsViolatingNodeTaintsArgs) error {
	return errorsAggregate(
		validateNamespaceArgs(args.Namespaces),
		validateLabelSelectorArgs(args.LabelSelector),
	)
}

// ValidateRemoveFailedPodsArgs validates RemoveFailedPods arguments
func ValidateRemoveFailedPodsArgs(args *componentconfig.RemoveFailedPodsArgs) error {
	return errorsAggregate(
		validateNamespaceArgs(args.Namespaces),
		validateLabelSelectorArgs(args.LabelSelector),
	)
}

// ValidateRemovePodsHavingTooManyRestartsArgs validates RemovePodsHavingTooManyRestarts arguments
func ValidateRemovePodsHavingTooManyRestartsArgs(args *componentconfig.RemovePodsHavingTooManyRestartsArgs) error {
	return errorsAggregate(
		validateNamespaceArgs(args.Namespaces),
		validateLabelSelectorArgs(args.LabelSelector),
		validatePodRestartThreshold(args.PodRestartThreshold),
	)
}

// ValidateRemovePodsViolatingNodeAffinityArgs validates RemovePodsViolatingNodeAffinity arguments
func ValidateRemovePodsViolatingNodeAffinityArgs(args *componentconfig.RemovePodsViolatingNodeAffinityArgs) error {
	var err error
	if args == nil || len(args.NodeAffinityType) == 0 {
		err = fmt.Errorf("nodeAffinityType needs to be set")
	}

	return errorsAggregate(
		err,
		validateNamespaceArgs(args.Namespaces),
		validateLabelSelectorArgs(args.LabelSelector),
	)
}

// ValidatePodLifeTimeArgs validates PodLifeTime arguments
func ValidatePodLifeTimeArgs(args *componentconfig.PodLifeTimeArgs) error {
	var err error
	if args.MaxPodLifeTimeSeconds == nil {
		err = fmt.Errorf("MaxPodLifeTimeSeconds not set")
	}

	return errorsAggregate(
		err,
		validateNamespaceArgs(args.Namespaces),
		validateLabelSelectorArgs(args.LabelSelector),
		validatePodLifeTimeStates(args.States),
	)
}

func ValidateRemoveDuplicatesArgs(args *componentconfig.RemoveDuplicatesArgs) error {
	return validateNamespaceArgs(args.Namespaces)
}

// errorsAggregate converts all arg validation errors to a single error interface.
// if no errors, it will return nil.
func errorsAggregate(errors ...error) error {
	return utilerrors.NewAggregate(errors)
}

func validateNamespaceArgs(namespaces *api.Namespaces) error {
	// At most one of include/exclude can be set
	if namespaces != nil && len(namespaces.Include) > 0 && len(namespaces.Exclude) > 0 {
		return fmt.Errorf("only one of Include/Exclude namespaces can be set")
	}

	return nil
}

func validateLabelSelectorArgs(labelSelector *metav1.LabelSelector) error {
	if labelSelector != nil {
		if _, err := metav1.LabelSelectorAsSelector(labelSelector); err != nil {
			return fmt.Errorf("failed to get label selectors from strategy's params: %+v", err)
		}
	}

	return nil
}

func validatePodRestartThreshold(podRestartThreshold int32) error {
	if podRestartThreshold < 1 {
		return fmt.Errorf("PodsHavingTooManyRestarts threshold not set")
	}
	return nil
}

func validatePodLifeTimeStates(states []string) error {
	podLifeTimeAllowedStates := sets.NewString(
		string(v1.PodRunning),
		string(v1.PodPending),

		// Container state reasons: https://github.com/kubernetes/kubernetes/blob/release-1.24/pkg/kubelet/kubelet_pods.go#L76-L79
		"PodInitializing",
		"ContainerCreating",
	)

	if !podLifeTimeAllowedStates.HasAll(states...) {
		return fmt.Errorf("states must be one of %v", podLifeTimeAllowedStates.List())
	}

	return nil
}

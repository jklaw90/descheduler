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

package descheduler

import (
	"sigs.k8s.io/descheduler/pkg/api"
	"sigs.k8s.io/descheduler/pkg/apis/componentconfig"
	"sigs.k8s.io/descheduler/pkg/apis/componentconfig/validation"
	"sigs.k8s.io/descheduler/pkg/framework"
	"sigs.k8s.io/descheduler/pkg/framework/plugins/removepodsviolatingnodetaints"
)

func RemovePodsViolatingNodeTaints2plugin(params *api.StrategyParameters, handle *handleImpl) (framework.DeschedulePlugin, error) {
	// Once all strategies are migrated the arguments get read from the configuration file
	// without any wiring. Keeping the wiring here so the descheduler can still use
	// the v1alpha1 configuration during the strategy migration to plugins.
	args := &componentconfig.RemovePodsViolatingNodeTaintsArgs{
		Namespaces:              params.Namespaces,
		LabelSelector:           params.LabelSelector,
		IncludePreferNoSchedule: params.IncludePreferNoSchedule,
		ExcludedTaints:          params.ExcludedTaints,
	}
	if err := validation.ValidateRemovePodsViolatingNodeTaintsArgs(args); err != nil {
		return nil, err
	}
	pg, err := removepodsviolatingnodetaints.New(args, handle)
	if err != nil {
		return nil, err
	}
	return pg.(framework.DeschedulePlugin), nil
}

/*
Copyright 2025 Red Hat, Inc.

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

package util

import (
	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// controlPlaneMachineSetKind is the kind used by the ControlPlaneMachineSet resource.
	controlPlaneMachineSetKind = "ControlPlaneMachineSet"

	// machineRoleLabelName is the label for specifying the role of a machine.
	machineRoleLabelName = "machine.openshift.io/cluster-api-machine-role"
)

// IsControlPlaneMAPIMachine returns true if the given MAPI Machine is a control plane machine.
func IsControlPlaneMAPIMachine(machine *mapiv1beta1.Machine) bool {
	if machine == nil {
		return false
	}

	if hasOwnerKind(machine.OwnerReferences, controlPlaneMachineSetKind) {
		return true
	}

	if role, exists := machine.Labels[machineRoleLabelName]; exists && role == "master" {
		return true
	}

	return false
}

func hasOwnerKind(refs []metav1.OwnerReference, kind string) bool {
	for _, ownerRef := range refs {
		if ownerRef.Kind == kind {
			return true
		}
	}

	return false
}

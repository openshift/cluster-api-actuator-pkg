/*
Copyright 2024 Red Hat, Inc.

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
package capi2mapi

import mapiv1beta1 "github.com/openshift/api/machine/v1beta1"

// MachineAndInfrastructureMachine represents the conversion between a CAPI Machine and InfrastructureMachine to a MAPI Machine.
type MachineAndInfrastructureMachine interface {
	ToMachine() (*mapiv1beta1.Machine, []string, error)
}

// MachineSetAndMachineTemplate represents the conversion between a CAPI MachineSet and MachineTemplate to a MAPI MachineSet.
type MachineSetAndMachineTemplate interface {
	ToMachineSet() (*mapiv1beta1.MachineSet, []string, error)
}

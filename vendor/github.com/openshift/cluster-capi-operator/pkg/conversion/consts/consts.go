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
package consts

const (
	// MAPIMachineMetadataLabelInstanceType is the label for the instance type of the machine is used as column for kubectl.
	MAPIMachineMetadataLabelInstanceType = "machine.openshift.io/instance-type"

	// MAPIMachineMetadataLabelRegion is the label for the region of the machine is used as column for kubectl.
	MAPIMachineMetadataLabelRegion = "machine.openshift.io/region"

	// MAPIMachineMetadataLabelZone is the label for the zone of the machine and is used as column for kubectl.
	MAPIMachineMetadataLabelZone = "machine.openshift.io/zone"

	// MAPIMachineMetadataAnnotationInstanceState is the annotation for the instance state of the machine is used as column for kubectl.
	MAPIMachineMetadataAnnotationInstanceState = "machine.openshift.io/instance-state"
)

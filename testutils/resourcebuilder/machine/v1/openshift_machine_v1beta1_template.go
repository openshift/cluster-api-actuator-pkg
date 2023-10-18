/*
Copyright 2022 Red Hat, Inc.

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

package v1

import (
	machinev1 "github.com/openshift/api/machine/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder"
	machinev1beta1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/machine/v1beta1"
	"k8s.io/utils/ptr"
)

// OpenShiftMachineV1Beta1Template creates a new OpenShift machine template builder.
func OpenShiftMachineV1Beta1Template() OpenShiftMachineV1Beta1TemplateBuilder {
	return OpenShiftMachineV1Beta1TemplateBuilder{
		labels: map[string]string{
			resourcebuilder.MachineRoleLabelName: "master",
			resourcebuilder.MachineTypeLabelName: "master",
			machinev1beta1.MachineClusterIDLabel: resourcebuilder.TestClusterIDValue,
		},
		providerSpecBuilder: machinev1beta1resourcebuilder.AWSProviderSpec(),
	}
}

// OpenShiftMachineV1Beta1TemplateBuilder is used to build out an OpenShift machine template.
type OpenShiftMachineV1Beta1TemplateBuilder struct {
	failureDomainsBuilder resourcebuilder.OpenShiftMachineV1Beta1FailureDomainsBuilder
	labels                map[string]string
	providerSpecBuilder   resourcebuilder.RawExtensionBuilder
}

// BuildTemplate builds a new machine template based on the configuration provided.
func (m OpenShiftMachineV1Beta1TemplateBuilder) BuildTemplate() machinev1.ControlPlaneMachineSetTemplate {
	template := machinev1.ControlPlaneMachineSetTemplate{
		MachineType: machinev1.OpenShiftMachineV1Beta1MachineType,
		OpenShiftMachineV1Beta1Machine: &machinev1.OpenShiftMachineV1Beta1MachineTemplate{
			ObjectMeta: machinev1.ControlPlaneMachineSetTemplateObjectMeta{
				Labels: m.labels,
			},
		},
	}

	if m.failureDomainsBuilder != nil {
		template.OpenShiftMachineV1Beta1Machine.FailureDomains = ptr.To[machinev1.FailureDomains](m.failureDomainsBuilder.BuildFailureDomains())
	}

	if m.providerSpecBuilder != nil {
		template.OpenShiftMachineV1Beta1Machine.Spec.ProviderSpec.Value = m.providerSpecBuilder.BuildRawExtension()
	}

	return template
}

// WithFailureDomainsBuilder sets the failure domains builder for the machine template builder.
func (m OpenShiftMachineV1Beta1TemplateBuilder) WithFailureDomainsBuilder(fdsBuilder resourcebuilder.OpenShiftMachineV1Beta1FailureDomainsBuilder) OpenShiftMachineV1Beta1TemplateBuilder {
	m.failureDomainsBuilder = fdsBuilder
	return m
}

// WithLabel sets the label on the machine labels for the machine template builder.
func (m OpenShiftMachineV1Beta1TemplateBuilder) WithLabel(key, value string) OpenShiftMachineV1Beta1TemplateBuilder {
	if m.labels == nil {
		m.labels = make(map[string]string)
	}

	m.labels[key] = value

	return m
}

// WithLabels sets the labels for the machine template builder.
func (m OpenShiftMachineV1Beta1TemplateBuilder) WithLabels(labels map[string]string) OpenShiftMachineV1Beta1TemplateBuilder {
	m.labels = labels
	return m
}

// WithProviderSpecBuilder sets the providerSpec builder for the machine template builder.
func (m OpenShiftMachineV1Beta1TemplateBuilder) WithProviderSpecBuilder(builder resourcebuilder.RawExtensionBuilder) OpenShiftMachineV1Beta1TemplateBuilder {
	m.providerSpecBuilder = builder
	return m
}

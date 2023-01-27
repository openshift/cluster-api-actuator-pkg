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

package v1beta1

import (
	"encoding/json"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VSphereProviderSpec creates a new VSphere machine config builder.
func VSphereProviderSpec() VSphereProviderSpecBuilder {
	return VSphereProviderSpecBuilder{
		template: "test-ln-xw89i22-c1627-rvtrn-rhcos",
	}
}

// VSphereProviderSpecBuilder is used to build out a VSphere machine config object.
type VSphereProviderSpecBuilder struct {
	template string
}

// Build builds a new VSphere machine config based on the configuration provided.
func (v VSphereProviderSpecBuilder) Build() *machinev1beta1.VSphereMachineProviderSpec {
	return &machinev1beta1.VSphereMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VSphereMachineProviderSpec",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		NumCoresPerSocket: 4,
		DiskGiB:           120,
		UserDataSecret: &v1.LocalObjectReference{
			Name: "master-user-data",
		},
		MemoryMiB: 16384,
		CredentialsSecret: &v1.LocalObjectReference{
			Name: "vsphere-cloud-credentials",
		},
		Network: machinev1beta1.NetworkSpec{
			Devices: []machinev1beta1.NetworkDeviceSpec{
				{
					NetworkName: "test-segment-01",
				},
			},
		},
		NumCPUs:  4,
		Template: v.template,
	}
}

// BuildRawExtension builds a new VSphere machine config based on the configuration provided.
func (v VSphereProviderSpecBuilder) BuildRawExtension() *runtime.RawExtension {
	providerConfig := v.Build()

	raw, err := json.Marshal(providerConfig)
	if err != nil {
		// As we are building the input to json.Marshal, this should never happen.
		panic(err)
	}

	return &runtime.RawExtension{
		Raw: raw,
	}
}

// WithTemplate sets the template for the VSphere machine config builder.
func (v VSphereProviderSpecBuilder) WithTemplate(template string) VSphereProviderSpecBuilder {
	v.template = template
	return v
}

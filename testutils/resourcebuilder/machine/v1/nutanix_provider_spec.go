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
	"encoding/json"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

// NewNutanixMachineProviderConfigBuilder returns a NutanixMachineProviderConfigBuilder.
func NewNutanixMachineProviderConfigBuilder() *NutanixMachineProviderConfigBuilder {
	return &NutanixMachineProviderConfigBuilder{}
}

// NutanixMachineProviderConfigBuilder is used to build a NutanixMachineProviderConfig.
type NutanixMachineProviderConfigBuilder struct {
	// failureDomains holds the Nutanix failure domains data for the builder to use
	failureDomains []configv1.NutanixFailureDomain
	// failureDomainName configures the failure domain name the build will use
	failureDomainName string
}

// Build returns the generated NutanixMachineProviderConfig.
func (n *NutanixMachineProviderConfigBuilder) Build() *machinev1.NutanixMachineProviderConfig {
	providerConfig := &machinev1.NutanixMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: machinev1.GroupVersion.String(),
			Kind:       "NutanixMachineProviderConfig",
		},
		UserDataSecret:    &corev1.LocalObjectReference{Name: "nutanix-user-data"},
		CredentialsSecret: &corev1.LocalObjectReference{Name: "nutanix-credentials"},
		Image: machinev1.NutanixResourceIdentifier{
			Type: machinev1.NutanixIdentifierName,
			Name: ptr.To[string]("rhcos"),
		},
		Subnets:        []machinev1.NutanixResourceIdentifier{{Type: machinev1.NutanixIdentifierName, Name: ptr.To[string]("default-net")}},
		VCPUsPerSocket: int32(1),
		VCPUSockets:    int32(4),
		MemorySize:     resource.MustParse(fmt.Sprintf("%dMi", 8096)),
		Cluster: machinev1.NutanixResourceIdentifier{
			Type: machinev1.NutanixIdentifierUUID,
			UUID: ptr.To[string]("pe-uuid"),
		},
		SystemDiskSize: resource.MustParse(fmt.Sprintf("%dGi", 120)),
	}

	if len(n.failureDomainName) > 0 {
		var failureDomain *configv1.NutanixFailureDomain

		for _, fd := range n.failureDomains {
			if fd.Name == n.failureDomainName {
				failureDomain = ptr.To[configv1.NutanixFailureDomain](fd)
				break
			}
		}

		if failureDomain == nil {
			// The failureDomainName is not found in the Infrastructure resource
			panic(fmt.Sprintf("The failure domain with name %q is not configured.", n.failureDomainName))
		}

		providerConfig.FailureDomain = &machinev1.NutanixFailureDomainReference{
			Name: failureDomain.Name,
		}

		// update Cluster
		providerConfig.Cluster = machinev1.NutanixResourceIdentifier{
			Name: failureDomain.Cluster.Name,
			UUID: failureDomain.Cluster.UUID,
		}

		switch failureDomain.Cluster.Type {
		case configv1.NutanixIdentifierName:
			providerConfig.Cluster.Type = machinev1.NutanixIdentifierName
		case configv1.NutanixIdentifierUUID:
			providerConfig.Cluster.Type = machinev1.NutanixIdentifierUUID
		default:
		}

		// update Subnets
		providerConfig.Subnets = []machinev1.NutanixResourceIdentifier{}

		for _, fdSubnet := range failureDomain.Subnets {
			pcSubnet := machinev1.NutanixResourceIdentifier{
				Name: fdSubnet.Name,
				UUID: fdSubnet.UUID,
			}

			switch fdSubnet.Type {
			case configv1.NutanixIdentifierName:
				pcSubnet.Type = machinev1.NutanixIdentifierName
			case configv1.NutanixIdentifierUUID:
				pcSubnet.Type = machinev1.NutanixIdentifierUUID
			default:
			}

			providerConfig.Subnets = append(providerConfig.Subnets, pcSubnet)
		}
	}

	return providerConfig
}

// BuildRawExtension builds a new Nutanix machine config based on the configuration provided.
func (n *NutanixMachineProviderConfigBuilder) BuildRawExtension() *runtime.RawExtension {
	providerConfig := n.Build()

	raw, err := json.Marshal(providerConfig)
	if err != nil {
		// As we are building the input to json.Marshal, this should never happen.
		panic(err)
	}

	return &runtime.RawExtension{
		Raw: raw,
	}
}

// WithFailureDomains sets the failureDomains field with the input value.
func (n *NutanixMachineProviderConfigBuilder) WithFailureDomains(failureDomains []configv1.NutanixFailureDomain) *NutanixMachineProviderConfigBuilder {
	n.failureDomains = failureDomains
	return n
}

// WithFailureDomainName sets the failureDomainName field with the input value.
func (n *NutanixMachineProviderConfigBuilder) WithFailureDomainName(failureDomainName string) *NutanixMachineProviderConfigBuilder {
	n.failureDomainName = failureDomainName
	return n
}

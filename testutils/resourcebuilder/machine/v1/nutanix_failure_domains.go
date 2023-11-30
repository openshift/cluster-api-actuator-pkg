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
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
)

// NutanixFailureDomains creates a new failure domains builder for Nutanix.
func NutanixFailureDomains() NutanixFailureDomainsBuilder {
	fdsBuilder := NutanixFailureDomainsBuilder{
		failureDomainsBuilders: []NutanixFailureDomainBuilder{
			{Name: "fd-pe0"},
			{Name: "fd-pe1"},
			{Name: "fd-pe2"},
		},
	}

	return fdsBuilder
}

// NutanixFailureDomainsBuilder is used to build a failuredomains for Nutanix.
type NutanixFailureDomainsBuilder struct {
	failureDomainsBuilders []NutanixFailureDomainBuilder
}

// BuildFailureDomains builds a failuredomains from the configuration.
func (b NutanixFailureDomainsBuilder) BuildFailureDomains() machinev1.FailureDomains {
	fds := machinev1.FailureDomains{
		Platform: configv1.NutanixPlatformType,
		Nutanix:  []machinev1.NutanixFailureDomainReference{},
	}

	for _, builder := range b.failureDomainsBuilders {
		fds.Nutanix = append(fds.Nutanix, builder.Build())
	}

	return fds
}

// WithFailureDomainBuilder adds a failure domain builder to the failure domains builder's builders.
func (b NutanixFailureDomainsBuilder) WithFailureDomainBuilder(fdbuilder NutanixFailureDomainBuilder) NutanixFailureDomainsBuilder {
	b.failureDomainsBuilders = append(b.failureDomainsBuilders, fdbuilder)
	return b
}

// WithFailureDomainBuilders replaces the failure domains builder's builders with the given builders.
func (b NutanixFailureDomainsBuilder) WithFailureDomainBuilders(fdbuilders ...NutanixFailureDomainBuilder) NutanixFailureDomainsBuilder {
	b.failureDomainsBuilders = fdbuilders
	return b
}

// NewNutanixFailureDomainBuilder creates a new failure domain builder for Nutanix.
func NewNutanixFailureDomainBuilder() NutanixFailureDomainBuilder {
	return NutanixFailureDomainBuilder{}
}

// NutanixFailureDomainBuilder is used to build a Nutanix failuredomain.
type NutanixFailureDomainBuilder struct {
	Name string
}

// Build builds a Nutanix failuredomain from the configuration.
func (g NutanixFailureDomainBuilder) Build() machinev1.NutanixFailureDomainReference {
	return machinev1.NutanixFailureDomainReference{
		Name: g.Name,
	}
}

// WithName sets the zone for the Nutanix failuredomain builder.
func (g NutanixFailureDomainBuilder) WithName(name string) NutanixFailureDomainBuilder {
	g.Name = name
	return g
}

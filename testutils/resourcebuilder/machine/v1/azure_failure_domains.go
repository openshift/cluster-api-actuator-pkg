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

// Linter warns about duplicated code with OpenStack, GCP, and Azure FailureDomains builders.
// While the builders are almost identical, we need to keep them separate because they build different objects.
//
//nolint:dupl
package v1

import (
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
)

// AzureFailureDomains creates a new failure domains builder for Azure.
func AzureFailureDomains() AzureFailureDomainsBuilder {
	return AzureFailureDomainsBuilder{[]AzureFailureDomainBuilder{
		AzureFailureDomain().WithZone("1"),
		AzureFailureDomain().WithZone("2"),
		AzureFailureDomain().WithZone("3"),
	}}
}

// AzureFailureDomainsBuilder is used to build a failuredomains.
type AzureFailureDomainsBuilder struct {
	failureDomainsBuilders []AzureFailureDomainBuilder
}

// BuildFailureDomains builds a failuredomains from the configuration.
func (a AzureFailureDomainsBuilder) BuildFailureDomains() machinev1.FailureDomains {
	fds := machinev1.FailureDomains{
		Platform: configv1.AzurePlatformType,
		Azure:    &[]machinev1.AzureFailureDomain{},
	}

	for _, builder := range a.failureDomainsBuilders {
		*fds.Azure = append(*fds.Azure, builder.Build())
	}

	return fds
}

// WithFailureDomainBuilder adds a failure domain builder to the failure domains builder's builders.
func (a AzureFailureDomainsBuilder) WithFailureDomainBuilder(fdbuilder AzureFailureDomainBuilder) AzureFailureDomainsBuilder {
	a.failureDomainsBuilders = append(a.failureDomainsBuilders, fdbuilder)
	return a
}

// WithFailureDomainBuilders replaces the failure domains builder's builders with the given builders.
func (a AzureFailureDomainsBuilder) WithFailureDomainBuilders(fdbuilders ...AzureFailureDomainBuilder) AzureFailureDomainsBuilder {
	a.failureDomainsBuilders = fdbuilders
	return a
}

// AzureFailureDomain creates a new Azure failure domain builder for Azure.
func AzureFailureDomain() AzureFailureDomainBuilder {
	return AzureFailureDomainBuilder{}
}

// AzureFailureDomainBuilder is used to build an Azure failuredomain.
type AzureFailureDomainBuilder struct {
	Zone   string
	Subnet string
}

// Build builds a Azure failuredomain from the configuration.
func (a AzureFailureDomainBuilder) Build() machinev1.AzureFailureDomain {
	return machinev1.AzureFailureDomain{
		Zone:   a.Zone,
		Subnet: a.Subnet,
	}
}

// WithZone sets the zone for the Azure failuredomain builder.
func (a AzureFailureDomainBuilder) WithZone(zone string) AzureFailureDomainBuilder {
	a.Zone = zone
	return a
}

// WithSubnet sets the subnet for the Azure failuredomain builder.
func (a AzureFailureDomainBuilder) WithSubnet(subnet string) AzureFailureDomainBuilder {
	a.Subnet = subnet
	return a
}

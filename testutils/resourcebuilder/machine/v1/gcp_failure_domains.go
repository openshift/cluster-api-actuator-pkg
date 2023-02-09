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

// GCPFailureDomains creates a new failure domains builder for GCP.
func GCPFailureDomains() GCPFailureDomainsBuilder {
	return GCPFailureDomainsBuilder{[]GCPFailureDomainBuilder{
		GCPFailureDomain().WithZone("us-central1-a"),
		GCPFailureDomain().WithZone("us-central1-b"),
		GCPFailureDomain().WithZone("us-central1-c"),
	}}
}

// GCPFailureDomainsBuilder is used to build a failuredomains.
type GCPFailureDomainsBuilder struct {
	failureDomainsBuilders []GCPFailureDomainBuilder
}

// BuildFailureDomains builds a failuredomains from the configuration.
func (g GCPFailureDomainsBuilder) BuildFailureDomains() machinev1.FailureDomains {
	fds := machinev1.FailureDomains{
		Platform: configv1.GCPPlatformType,
		GCP:      &[]machinev1.GCPFailureDomain{},
	}

	for _, builder := range g.failureDomainsBuilders {
		*fds.GCP = append(*fds.GCP, builder.Build())
	}

	return fds
}

// WithFailureDomainBuilder adds a failure domain builder to the failure domains builder's builders.
func (g GCPFailureDomainsBuilder) WithFailureDomainBuilder(fdbuilder GCPFailureDomainBuilder) GCPFailureDomainsBuilder {
	g.failureDomainsBuilders = append(g.failureDomainsBuilders, fdbuilder)
	return g
}

// WithFailureDomainBuilders replaces the failure domains builder's builders with the given builders.
func (g GCPFailureDomainsBuilder) WithFailureDomainBuilders(fdbuilders ...GCPFailureDomainBuilder) GCPFailureDomainsBuilder {
	g.failureDomainsBuilders = fdbuilders
	return g
}

// GCPFailureDomain creates a new GCP failure domain builder for GCP.
func GCPFailureDomain() GCPFailureDomainBuilder {
	return GCPFailureDomainBuilder{}
}

// GCPFailureDomainBuilder is used to build a GCP failuredomain.
type GCPFailureDomainBuilder struct {
	Zone string
}

// Build builds a GCP failuredomain from the configuration.
func (g GCPFailureDomainBuilder) Build() machinev1.GCPFailureDomain {
	return machinev1.GCPFailureDomain{
		Zone: g.Zone,
	}
}

// WithZone sets the zone for the GCP failuredomain builder.
func (g GCPFailureDomainBuilder) WithZone(zone string) GCPFailureDomainBuilder {
	g.Zone = zone
	return g
}

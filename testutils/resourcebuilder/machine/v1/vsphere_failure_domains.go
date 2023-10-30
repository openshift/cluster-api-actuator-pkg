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

// Linter warns about duplicated code with OpenStack, VSphere, and Azure FailureDomains builders.
// While the builders are almost identical, we need to keep them separate because they build different objects.
//
//nolint:dupl
package v1

import (
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
)

// VSphereFailureDomains creates a new failure domains builder for VSphere.
func VSphereFailureDomains() VSphereFailureDomainsBuilder {
	return VSphereFailureDomainsBuilder{[]VSphereFailureDomainBuilder{
		VSphereFailureDomain().WithZone("us-central1-a"),
		VSphereFailureDomain().WithZone("us-central1-b"),
		VSphereFailureDomain().WithZone("us-central1-c"),
	}}
}

// VSphereFailureDomainsBuilder is used to build a failuredomains.
type VSphereFailureDomainsBuilder struct {
	failureDomainsBuilders []VSphereFailureDomainBuilder
}

// BuildFailureDomains builds a failuredomains from the configuration.
func (g VSphereFailureDomainsBuilder) BuildFailureDomains() machinev1.FailureDomains {
	fds := machinev1.FailureDomains{
		Platform: configv1.VSpherePlatformType,
		VSphere:  []machinev1.VSphereFailureDomain{},
	}

	for _, builder := range g.failureDomainsBuilders {
		fds.VSphere = append(fds.VSphere, builder.Build())
	}

	return fds
}

// WithFailureDomainBuilder adds a failure domain builder to the failure domains builder's builders.
func (g VSphereFailureDomainsBuilder) WithFailureDomainBuilder(fdbuilder VSphereFailureDomainBuilder) VSphereFailureDomainsBuilder {
	g.failureDomainsBuilders = append(g.failureDomainsBuilders, fdbuilder)
	return g
}

// WithFailureDomainBuilders replaces the failure domains builder's builders with the given builders.
func (g VSphereFailureDomainsBuilder) WithFailureDomainBuilders(fdbuilders ...VSphereFailureDomainBuilder) VSphereFailureDomainsBuilder {
	g.failureDomainsBuilders = fdbuilders
	return g
}

// VSphereFailureDomain creates a new VSphere failure domain builder for VSphere.
func VSphereFailureDomain() VSphereFailureDomainBuilder {
	return VSphereFailureDomainBuilder{}
}

// VSphereFailureDomainBuilder is used to build a VSphere failuredomain.
type VSphereFailureDomainBuilder struct {
	Name string
}

// Build builds a VSphere failuredomain from the configuration.
func (g VSphereFailureDomainBuilder) Build() machinev1.VSphereFailureDomain {
	return machinev1.VSphereFailureDomain{
		Name: g.Name,
	}
}

// WithZone sets the zone for the VSphere failuredomain builder.
func (g VSphereFailureDomainBuilder) WithZone(name string) VSphereFailureDomainBuilder {
	g.Name = name
	return g
}

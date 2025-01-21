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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// OpenStackCluster creates a new OpenStackClusterBuilder.
func OpenStackCluster() OpenStackClusterBuilder {
	return OpenStackClusterBuilder{}
}

// OpenStackClusterBuilder is used to build out an OpenStackCluster object.
type OpenStackClusterBuilder struct {
	// Object meta fields.
	annotations       map[string]string
	creationTimestamp metav1.Time
	deletionTimestamp *metav1.Time
	generateName      string
	labels            map[string]string
	name              string
	namespace         string

	// Spec fields.
	apiServerFixedIP                 *string
	apiServerFloatingIP              *string
	apiServerLoadBalancer            *capov1.APIServerLoadBalancer
	apiServerPort                    *uint16
	bastion                          *capov1.Bastion
	controlPlaneAvailabilityZones    []string
	controlPlaneOmitAvailabilityZone *bool
	controlPlaneEndpoint             *clusterv1.APIEndpoint
	disableAPIServerFloatingIP       *bool
	disableExternalNetwork           *bool
	disablePortSecurity              *bool
	externalNetwork                  *capov1.NetworkParam
	externalRouterIPs                []capov1.ExternalRouterIPParam
	identityRef                      capov1.OpenStackIdentityReference
	managedSecurityGroups            *capov1.ManagedSecurityGroups
	managedSubnets                   []capov1.SubnetSpec
	network                          *capov1.NetworkParam
	networkMTU                       *int
	router                           *capov1.RouterParam
	subnets                          []capov1.SubnetParam
	tags                             []string

	// Status fields.
	bastionStatus         *capov1.BastionStatus
	externalNetworkStatus *capov1.NetworkStatus
	failureDomains        clusterv1.FailureDomains
	networkStatus         *capov1.NetworkStatusWithSubnets
	ready                 bool
}

// Build builds a new OpenStackCluster based on the configuration provided.
func (a OpenStackClusterBuilder) Build() *capov1.OpenStackCluster {
	openstackCluster := &capov1.OpenStackCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capov1.SchemeGroupVersion.String(),
			Kind:       "OpenStackCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       a.annotations,
			CreationTimestamp: a.creationTimestamp,
			DeletionTimestamp: a.deletionTimestamp,
			GenerateName:      a.generateName,
			Labels:            a.labels,
			Name:              a.name,
			Namespace:         a.namespace,
		},
		Spec: capov1.OpenStackClusterSpec{
			APIServerFixedIP:                 a.apiServerFixedIP,
			APIServerFloatingIP:              a.apiServerFloatingIP,
			APIServerLoadBalancer:            a.apiServerLoadBalancer,
			APIServerPort:                    a.apiServerPort,
			Bastion:                          a.bastion,
			ControlPlaneAvailabilityZones:    a.controlPlaneAvailabilityZones,
			ControlPlaneOmitAvailabilityZone: a.controlPlaneOmitAvailabilityZone,
			ControlPlaneEndpoint:             a.controlPlaneEndpoint,
			DisableAPIServerFloatingIP:       a.disableAPIServerFloatingIP,
			DisableExternalNetwork:           a.disableExternalNetwork,
			DisablePortSecurity:              a.disablePortSecurity,
			ExternalNetwork:                  a.externalNetwork,
			ExternalRouterIPs:                a.externalRouterIPs,
			IdentityRef:                      a.identityRef,
			ManagedSecurityGroups:            a.managedSecurityGroups,
			ManagedSubnets:                   a.managedSubnets,
			Network:                          a.network,
			NetworkMTU:                       a.networkMTU,
			Router:                           a.router,
			Subnets:                          a.subnets,
			Tags:                             a.tags,
		},
		Status: capov1.OpenStackClusterStatus{
			Bastion:         a.bastionStatus,
			ExternalNetwork: a.externalNetworkStatus,
			FailureDomains:  a.failureDomains,
			Network:         a.networkStatus,
			Ready:           a.ready,
		},
	}

	return openstackCluster
}

// Object meta fields.

// WithAnnotations sets the annotations for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithAnnotations(annotations map[string]string) OpenStackClusterBuilder {
	a.annotations = annotations
	return a
}

// WithCreationTimestamp sets the creationTimestamp for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithCreationTimestamp(timestamp metav1.Time) OpenStackClusterBuilder {
	a.creationTimestamp = timestamp
	return a
}

// WithDeletionTimestamp sets the deletionTimestamp for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithDeletionTimestamp(timestamp *metav1.Time) OpenStackClusterBuilder {
	a.deletionTimestamp = timestamp
	return a
}

// WithGenerateName sets the generateName for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithGenerateName(generateName string) OpenStackClusterBuilder {
	a.generateName = generateName
	return a
}

// WithLabels sets the labels for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithLabels(labels map[string]string) OpenStackClusterBuilder {
	a.labels = labels
	return a
}

// WithName sets the name for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithName(name string) OpenStackClusterBuilder {
	a.name = name
	return a
}

// WithNamespace sets the namespace for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithNamespace(namespace string) OpenStackClusterBuilder {
	a.namespace = namespace
	return a
}

// Spec fields.

// WithAPIServerFixedIP sets the API server fixed IP for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithAPIServerFixedIP(fixedIP *string) OpenStackClusterBuilder {
	a.apiServerFixedIP = fixedIP
	return a
}

// WithAPIServerFloatingIP sets the API server fixed IP for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithAPIServerFloatingIP(floatingIP *string) OpenStackClusterBuilder {
	a.apiServerFloatingIP = floatingIP
	return a
}

// WithAPIServerLoadBalancer sets the API server load balancer for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithAPIServerLoadBalancer(loadBalancer *capov1.APIServerLoadBalancer) OpenStackClusterBuilder {
	a.apiServerLoadBalancer = loadBalancer
	return a
}

// WithAPIServerPort sets the API server port for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithAPIServerPort(port *uint16) OpenStackClusterBuilder {
	a.apiServerPort = port
	return a
}

// WithBastion sets the bastion for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithBastion(bastion *capov1.Bastion) OpenStackClusterBuilder {
	a.bastion = bastion
	return a
}

// WithControlPlaneOmitAvailabilityZone sets the controlPlaneOmitAvailabilityZone for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithControlPlaneOmitAvailabilityZone(omitAZ *bool) OpenStackClusterBuilder {
	a.controlPlaneOmitAvailabilityZone = omitAZ
	return a
}

// WithControlPlaneAvailabilityZones sets the controlPlaneAvailabilityZones for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithControlPlaneAvailabilityZones(azs []string) OpenStackClusterBuilder {
	a.controlPlaneAvailabilityZones = azs
	return a
}

// WithControlPlaneEndpoint sets the controlPlaneEndpoint for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithControlPlaneEndpoint(endpoint *clusterv1.APIEndpoint) OpenStackClusterBuilder {
	a.controlPlaneEndpoint = endpoint
	return a
}

// WithDisableAPIServerFloatingIP sets the disableAPIServerFloatingIP for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithDisableAPIServerFloatingIP(disableFIP *bool) OpenStackClusterBuilder {
	a.disableAPIServerFloatingIP = disableFIP
	return a
}

// WithDisableExternalNetwork sets the disableExternalNetwork for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithDisableExternalNetwork(disableExternalNetwork *bool) OpenStackClusterBuilder {
	a.disableExternalNetwork = disableExternalNetwork
	return a
}

// WithDisablePortSecurity sets the disablePortSecurity for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithDisablePortSecurity(disablePortSecurity *bool) OpenStackClusterBuilder {
	a.disablePortSecurity = disablePortSecurity
	return a
}

// WithExternalNetwork sets the externalNetwork for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithExternalNetwork(externalNetwork *capov1.NetworkParam) OpenStackClusterBuilder {
	a.externalNetwork = externalNetwork
	return a
}

// WithExternalRouterIPs sets the externalRouterIPs for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithExternalRouterIPs(externalRouterIPs []capov1.ExternalRouterIPParam) OpenStackClusterBuilder {
	a.externalRouterIPs = externalRouterIPs
	return a
}

// WithIdentityRef sets the identityRef for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithIdentityRef(identityRef capov1.OpenStackIdentityReference) OpenStackClusterBuilder {
	a.identityRef = identityRef
	return a
}

// WithManagedSecurityGroups sets the managedSecurityGroups for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithManagedSecurityGroups(managedSGs *capov1.ManagedSecurityGroups) OpenStackClusterBuilder {
	a.managedSecurityGroups = managedSGs
	return a
}

// WithManagedSubnets sets the managedSubnets for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithManagedSubnets(managedSubnets []capov1.SubnetSpec) OpenStackClusterBuilder {
	a.managedSubnets = managedSubnets
	return a
}

// WithNetwork sets the network for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithNetwork(network *capov1.NetworkParam) OpenStackClusterBuilder {
	a.network = network
	return a
}

// WithNetworkMTU sets the networkMTU for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithNetworkMTU(networkMTU *int) OpenStackClusterBuilder {
	a.networkMTU = networkMTU
	return a
}

// WithRouter sets the router for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithRouter(router *capov1.RouterParam) OpenStackClusterBuilder {
	a.router = router
	return a
}

// WithSubnets sets the subnets for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithSubnets(subnets []capov1.SubnetParam) OpenStackClusterBuilder {
	a.subnets = subnets
	return a
}

// WithTags sets the tags for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithTags(tags []string) OpenStackClusterBuilder {
	a.tags = tags
	return a
}

// Status fields.

// WithBastionStatus sets the bastion status for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithBastionStatus(bastionStatus *capov1.BastionStatus) OpenStackClusterBuilder {
	a.bastionStatus = bastionStatus
	return a
}

// WithExternalNetworkStatus sets the network status for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithExternalNetworkStatus(networkStatus *capov1.NetworkStatus) OpenStackClusterBuilder {
	a.externalNetworkStatus = networkStatus
	return a
}

// WithFailureDomains sets the failureDomains for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithFailureDomains(failureDomains clusterv1.FailureDomains) OpenStackClusterBuilder {
	a.failureDomains = failureDomains
	return a
}

// WithNetworkStatus sets the network status for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithNetworkStatus(networkStatus *capov1.NetworkStatusWithSubnets) OpenStackClusterBuilder {
	a.networkStatus = networkStatus
	return a
}

// WithReady sets the ready status for the OpenStackCluster builder.
func (a OpenStackClusterBuilder) WithReady(ready bool) OpenStackClusterBuilder {
	a.ready = ready
	return a
}

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
package controllers

import (
	"errors"

	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
)

const (
	// DefaultCAPINamespace is the default namespace for CAPI resources.
	DefaultCAPINamespace = "openshift-cluster-api"

	// DefaultMAPINamespace the default namespace for MAPI resources.
	DefaultMAPINamespace = "openshift-machine-api"

	// DefaultOperatorNamespace is the default namespace used for operator resources.
	DefaultOperatorNamespace = "openshift-cluster-api-operator"

	// DefaultCRDCompatibilityCheckerNamespace is the default namespace for CRD Compatibility Checker operator and controllers.
	DefaultCRDCompatibilityCheckerNamespace = "openshift-compatibility-requirements-operator"

	// OperatorVersionKey is the key used to store the operator version in the ClusterOperator status.
	OperatorVersionKey = "operator"

	// ClusterOperatorName is the name of the ClusterOperator resource.
	ClusterOperatorName = "cluster-api"

	// InfrastructureResourceName is the name of the cluster global infrastructure resource.
	InfrastructureResourceName = "cluster"

	// SynchronizedCondition is used to denote when a MAPI or CAPI resource is
	// synchronized. This condition should only be true when a synchronization
	// controller has successfully synchronized a non-authoritative resource.
	SynchronizedCondition mapiv1beta1.ConditionType = "Synchronized"

	// ReasonResourceSynchronized denotes that the resource is synchronized
	// successfully.
	ReasonResourceSynchronized = "ResourceSynchronized"

	// ReasonAuthoritativeAPIChanged indicates that sync state is stale due to a change of authoritativeAPI.
	ReasonAuthoritativeAPIChanged = "AuthoritativeAPIChanged"

	// MachineSetOpenshiftLabelKey is the key for label referring to a Machine API MachineSet.
	MachineSetOpenshiftLabelKey = "machine.openshift.io/cluster-api-machineset"
)

var (
	// ErrInvalidSpecAuthoritativeAPI is returned when spec.authoritativeAPI has a value which is not permitted.
	ErrInvalidSpecAuthoritativeAPI = errors.New("invalid spec.authoritativeAPI")
)

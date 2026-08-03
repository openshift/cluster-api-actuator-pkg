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
package capi2mapi

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-capi-operator/pkg/conversion/consts"
	"github.com/openshift/cluster-capi-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capiutil "sigs.k8s.io/cluster-api/util"
)

var (
	errCAPIMachineAWSMachineAWSClusterCannotBeNil            = errors.New("provided Machine, AWSMachine and AWSCluster can not be nil")
	errCAPIMachineSetAWSMachineTemplateAWSClusterCannotBeNil = errors.New("provided MachineSet, AWSMachineTemplate and AWSCluster can not be nil")
	errNilLoadBalancer                                       = errors.New("nil load balancer")
	errUnsupportedLoadBalancerType                           = errors.New("unsupported load balancer type")

	// awsDedicatedHostNamePattern is used to validate the id of a dedicated host.
	awsDedicatedHostNamePattern = regexp.MustCompile(`^h-(?:[0-9a-f]{8}|[0-9a-f]{17})$`)
)

const (
	errUnsupportedCAPATenancy      = "unable to convert tenancy, unknown value"
	errUnsupportedCAPAMarketType   = "unable to convert market type, unknown value"
	errUnsupportedHTTPTokensState  = "unable to convert httpTokens state, unknown value" //nolint:gosec // This is an error message, not a credential
	defaultIdentityName            = "default"
	defaultCredentialsSecretName   = "aws-cloud-credentials" //#nosec G101 -- False positive, not actually a credential.
	errUnsupportedHostAffinityType = "unable to convert hostAffinity, unknown value"
	errHostIDRequired              = "id is required and must start with 'h-' followed by 8 or 17 lowercase hexadecimal characters (0-9 and a-f)"
	errHostIDInvalidFormat         = "id must start with 'h-' followed by 8 or 17 lowercase hexadecimal characters (0-9 and a-f)"

	// TenancyDefault default setting for tenancy.
	TenancyDefault = "default"
	// TenancyDedicated dedicated setting for tenancy.
	TenancyDedicated = "dedicated"
	// TenancyHost host setting for tenancy.
	TenancyHost = "host"
)

// machineAndAWSMachineAndAWSCluster stores the details of a Cluster API Machine and AWSMachine and AWSCluster.
type machineAndAWSMachineAndAWSCluster struct {
	machine                               *clusterv1.Machine
	awsMachine                            *awsv1.AWSMachine
	awsCluster                            *awsv1.AWSCluster
	excludeMachineAPILabelsAndAnnotations bool
}

// machineSetAndAWSMachineTemplateAndAWSCluster stores the details of a Cluster API MachineSet and AWSMachineTemplate and AWSCluster.
type machineSetAndAWSMachineTemplateAndAWSCluster struct {
	machineSet *clusterv1.MachineSet
	template   *awsv1.AWSMachineTemplate
	awsCluster *awsv1.AWSCluster
	*machineAndAWSMachineAndAWSCluster
}

// FromMachineAndAWSMachineAndAWSCluster wraps a CAPI Machine and CAPA AWSMachine and CAPA AWSCluster into a capi2mapi MachineAndInfrastructureMachine.
func FromMachineAndAWSMachineAndAWSCluster(m *clusterv1.Machine, am *awsv1.AWSMachine, ac *awsv1.AWSCluster) MachineAndInfrastructureMachine {
	return &machineAndAWSMachineAndAWSCluster{machine: m, awsMachine: am, awsCluster: ac}
}

// FromMachineSetAndAWSMachineTemplateAndAWSCluster wraps a CAPI MachineSet and CAPA AWSMachineTemplate and CAPA AWSCluster into a capi2mapi MachineSetAndAWSMachineTemplateAndAWSCluster.
func FromMachineSetAndAWSMachineTemplateAndAWSCluster(ms *clusterv1.MachineSet, mts *awsv1.AWSMachineTemplate, ac *awsv1.AWSCluster) MachineSetAndMachineTemplate {
	return &machineSetAndAWSMachineTemplateAndAWSCluster{
		machineSet: ms,
		template:   mts,
		awsCluster: ac,
		machineAndAWSMachineAndAWSCluster: &machineAndAWSMachineAndAWSCluster{
			machine: &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      ms.Spec.Template.Labels,
					Annotations: ms.Spec.Template.Annotations,
				},
				Spec: ms.Spec.Template.Spec,
			},
			awsMachine: &awsv1.AWSMachine{
				Spec: mts.Spec.Template.Spec,
			},
			awsCluster:                            ac,
			excludeMachineAPILabelsAndAnnotations: true,
		},
	}
}

// toProviderSpec converts a capi2mapi MachineAndAWSMachineTemplateAndAWSCluster into a MAPI AWSMachineProviderConfig.
//
//nolint:funlen
func (m machineAndAWSMachineAndAWSCluster) toProviderSpec() (*mapiv1beta1.AWSMachineProviderConfig, []string, field.ErrorList) {
	var errors field.ErrorList

	fldPath := field.NewPath("spec")

	mapaTenancy, err := convertAWSTenancyToMAPI(fldPath.Child("tenancy"), m.awsMachine.Spec.Tenancy)
	if err != nil {
		errors = append(errors, err)
	}

	mapiAWSMetadataOptions, warnings, errs := convertAWSMetadataOptionsToMAPI(fldPath.Child("instanceMetadataOptions"), m.awsMachine.Spec.InstanceMetadataOptions)
	if errs != nil {
		errors = append(errors, errs...)
	}

	mapiAWSMarketType, err := convertAWSMarketTypeToMAPI(fldPath.Child("marketType"), m.awsMachine.Spec.MarketType)
	if err != nil {
		errors = append(errors, err)
	}

	mapiLoadBalancers, lbErrs := convertAWSClusterLoadBalancersToMAPI(fldPath, m.machine, m.awsCluster)
	if len(lbErrs) > 0 {
		errors = append(errors, lbErrs...)
	}

	var placementGroupPartition *int32

	if converted, ok := int64ToInt32(m.awsMachine.Spec.PlacementGroupPartition); !ok {
		errors = append(errors, field.Invalid(fldPath.Child("placementGroupPartition"), m.awsMachine.Spec.PlacementGroupPartition, "placementGroupPartition exceeds maximum int32 value"))
	} else if converted > 0 {
		placementGroupPartition = &converted
	}

	mapaProviderConfig := mapiv1beta1.AWSMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: "AWSMachineProviderConfig",
			// In the machineSets both "awsproviderconfig.openshift.io/v1beta1" and "machine.openshift.io/v1beta1" can be found.
			// Here we always settle on one of the two.
			APIVersion: "machine.openshift.io/v1beta1",
		},
		// ObjectMeta - Only present because it's needed to form part of the runtime.RawExtension, not actually used by MAPA.
		AMI: mapiv1beta1.AWSResourceReference{
			// The use of ARN and Filters to reference AMIs was present
			// in CAPA but has been deprecated and then removed
			// ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/pull/3257
			ID: m.awsMachine.Spec.AMI.ID,
		},
		InstanceType: m.awsMachine.Spec.InstanceType,
		CPUOptions:   ConvertAWSCPUOptionsToMAPI(m.awsMachine.Spec.CPUOptions),
		Tags:         convertAWSTagsToMAPI(m.awsMachine.Spec.AdditionalTags),
		IAMInstanceProfile: &mapiv1beta1.AWSResourceReference{
			ID: &m.awsMachine.Spec.IAMInstanceProfile,
		},
		// UserDataSecret - Populated below.
		// CredentialsSecret - Handled below.
		KeyName: m.awsMachine.Spec.SSHKeyName,
		// DeviceIndex - OCPCLOUD-2707: Value must always be zero. No other values are valid in MAPA even though the value is configurable.
		PublicIP:             m.awsMachine.Spec.PublicIP,
		SecurityGroups:       convertAWSSecurityGroupstoMAPI(m.awsMachine.Spec.AdditionalSecurityGroups), // This is the way we want to convert security groups, as the AdditionalSecurity Groups are what gets added to MAPI SGs.
		NetworkInterfaceType: convertAWSNetworkInterfaceTypeToMAPI(m.awsMachine.Spec.NetworkInterfaceType),
		Subnet:               convertAWSResourceReferenceToMAPI(ptr.Deref(m.awsMachine.Spec.Subnet, awsv1.AWSResourceReference{})),
		Placement: mapiv1beta1.Placement{
			AvailabilityZone: m.machine.Spec.FailureDomain,
			Tenancy:          mapaTenancy,
			Region:           m.awsCluster.Spec.Region,
		},
		// HostPlacement - Populated below.
		LoadBalancers: mapiLoadBalancers,
		// BlockDevices - Populated below.
		SpotMarketOptions:       convertAWSSpotMarketOptionsToMAPI(m.awsMachine.Spec.SpotMarketOptions),
		MetadataServiceOptions:  mapiAWSMetadataOptions,
		PlacementGroupName:      m.awsMachine.Spec.PlacementGroupName,
		PlacementGroupPartition: placementGroupPartition,
		CapacityReservationID:   ptr.Deref(m.awsMachine.Spec.CapacityReservationID, ""),
		MarketType:              mapiAWSMarketType,
	}

	// Dedicated host support
	mapaProviderConfig.Placement.Host, errs = convertAWSDedicatedHostToMAPI(m.awsMachine.Spec, mapaTenancy, fldPath)
	if len(errs) > 0 {
		errors = append(errors, errs...)
	}

	secretRef, errs := handleAWSIdentityRef(fldPath.Child("identityRef"), m.awsCluster.Spec.IdentityRef)

	if len(errs) > 0 {
		errors = append(errors, errs...)
	} else {
		mapaProviderConfig.CredentialsSecret = secretRef
	}

	userDataSecretName := ptr.Deref(m.machine.Spec.Bootstrap.DataSecretName, "")
	if userDataSecretName != "" {
		mapaProviderConfig.UserDataSecret = &corev1.LocalObjectReference{
			Name: userDataSecretName,
		}
	}

	mapaProviderConfig.BlockDevices, errs = convertAWSVolumesToMAPI(fldPath, m.awsMachine.Spec.RootVolume, m.awsMachine.Spec.NonRootVolumes)
	if len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Below this line are fields not used from the CAPI AWSMachine.

	// ProviderID - Populated at a different level.
	// InstanceID - Ignore - Is a subset of providerID.
	// Ignition - Ignore - Only has a version field and we force this to a particular value.

	if m.awsMachine.Spec.NetworkInterfaceType != "" && m.awsMachine.Spec.NetworkInterfaceType != awsv1.NetworkInterfaceTypeEFAWithENAInterface && m.awsMachine.Spec.NetworkInterfaceType != awsv1.NetworkInterfaceTypeENI {
		errors = append(errors, field.Invalid(fldPath.Child("networkInterfaceType"), m.awsMachine.Spec.NetworkInterfaceType, "networkInterface type must be one of interface, efa or omitted, unsupported value"))
	}
	// There are quite a few unsupported fields, so break them out for now.
	errors = append(errors, handleUnsupportedAWSMachineFields(fldPath, m.awsMachine.Spec)...)

	if len(errors) > 0 {
		return nil, warnings, errors
	}

	return &mapaProviderConfig, warnings, nil
}

func (m machineAndAWSMachineAndAWSCluster) toProviderStatus() *mapiv1beta1.AWSMachineProviderStatus {
	s := &mapiv1beta1.AWSMachineProviderStatus{
		InstanceState: ptr.To(string(ptr.Deref(m.awsMachine.Status.InstanceState, ""))),
		InstanceID:    m.awsMachine.Spec.InstanceID,
		Conditions:    convertCAPAMachineConditionsToMAPIMachineAWSProviderConditions(m.awsMachine),
	}

	// Convert DedicatedHost status if present
	if m.awsMachine.Status.DedicatedHost != nil {
		dedicatedHostID := ptr.Deref(m.awsMachine.Status.DedicatedHost.ID, "")
		if dedicatedHostID != "" {
			s.DedicatedHost = &mapiv1beta1.DedicatedHostStatus{
				ID: dedicatedHostID,
			}
		}
	}

	return s
}

func convertDynamicHostAllocationTags(dynamicHostAllocation *awsv1.DynamicHostAllocationSpec) *mapiv1beta1.DynamicHostAllocationSpec {
	if dynamicHostAllocation == nil || dynamicHostAllocation.Tags == nil {
		return nil
	}

	// Collect keys first to ensure deterministic ordering
	keys := make([]string, 0, len(dynamicHostAllocation.Tags))
	for key := range dynamicHostAllocation.Tags {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	// Build tags slice in sorted order
	tags := make([]mapiv1beta1.TagSpecification, 0, len(keys))
	for _, key := range keys {
		tags = append(tags, mapiv1beta1.TagSpecification{
			Name:  key,
			Value: dynamicHostAllocation.Tags[key],
		})
	}

	// Only create DynamicHostAllocationSpec if there are tags.
	// The MAPI API has MinProperties=1 validation, so an empty struct is invalid.
	if len(tags) == 0 {
		return nil
	}

	return &mapiv1beta1.DynamicHostAllocationSpec{
		Tags: &tags,
	}
}

func convertHostAffinityHost(spec awsv1.AWSMachineSpec, fldPath *field.Path) (*mapiv1beta1.HostPlacement, field.ErrorList) {
	var errorList field.ErrorList

	hasDynamicHostAllocation := spec.DynamicHostAllocation != nil
	hasHostID := spec.HostID != nil

	// For "host" affinity, either HostID (user-provided) or DynamicHostAllocation must be set
	if !hasHostID && !hasDynamicHostAllocation {
		errorList = append(errorList, field.Required(fldPath.Child("dedicatedHost"), "either id or dynamicHostAllocation is required when hostAffinity is host"))
		return nil, errorList
	}

	if hasHostID && hasDynamicHostAllocation {
		errorList = append(errorList, field.Invalid(fldPath.Child("dedicatedHost"), spec.HostID, "id and dynamicHostAllocation are mutually exclusive"))
		return nil, errorList
	}

	if hasHostID {
		// User-provided host ID
		if !awsDedicatedHostNamePattern.MatchString(*spec.HostID) {
			errorList = append(errorList, field.Invalid(fldPath.Child("dedicatedHost").Child("id"), *spec.HostID, errHostIDInvalidFormat))
			return nil, errorList
		}

		return &mapiv1beta1.HostPlacement{
			Affinity: ptr.To(mapiv1beta1.HostAffinityDedicatedHost),
			DedicatedHost: &mapiv1beta1.DedicatedHost{
				AllocationStrategy: ptr.To(mapiv1beta1.AllocationStrategyUserProvided),
				ID:                 *spec.HostID,
			},
		}, nil
	}

	// Dynamic host allocation
	dynamicAlloc := convertDynamicHostAllocationTags(spec.DynamicHostAllocation)

	return &mapiv1beta1.HostPlacement{
		Affinity: ptr.To(mapiv1beta1.HostAffinityDedicatedHost),
		DedicatedHost: &mapiv1beta1.DedicatedHost{
			AllocationStrategy:    ptr.To(mapiv1beta1.AllocationStrategyDynamic),
			DynamicHostAllocation: dynamicAlloc,
		},
	}, nil
}

func convertHostAffinityDefault(spec awsv1.AWSMachineSpec, tenancy mapiv1beta1.InstanceTenancy, fldPath *field.Path) (*mapiv1beta1.HostPlacement, field.ErrorList) {
	var errorList field.ErrorList

	hasDynamicHostAllocation := spec.DynamicHostAllocation != nil
	hasHostID := spec.HostID != nil

	// DynamicHostAllocation is not allowed with "default" affinity
	if hasDynamicHostAllocation {
		errorList = append(errorList, field.Invalid(fldPath.Child("dynamicHostAllocation"), spec.DynamicHostAllocation, "dynamicHostAllocation is only allowed when hostAffinity is host"))
		return nil, errorList
	}

	// Only create a HostPlacement when:
	// 1. Tenancy is "host" (even without a specific HostID), OR
	// 2. There's an explicit HostID (user-provided dedicated host)
	// Otherwise, return nil to indicate no dedicated host configuration
	if tenancy != mapiv1beta1.HostTenancy && !hasHostID {
		return nil, errorList
	}

	host := &mapiv1beta1.HostPlacement{
		Affinity: ptr.To(mapiv1beta1.HostAffinityAnyAvailable),
	}

	// For "default", host ID is optional
	if hasHostID {
		if !awsDedicatedHostNamePattern.MatchString(*spec.HostID) {
			errorList = append(errorList, field.Invalid(fldPath.Child("dedicatedHost").Child("id"), *spec.HostID, errHostIDInvalidFormat))
			return nil, errorList
		}

		host.DedicatedHost = &mapiv1beta1.DedicatedHost{
			AllocationStrategy: ptr.To(mapiv1beta1.AllocationStrategyUserProvided),
			ID:                 *spec.HostID,
		}
	}

	return host, errorList
}

func convertAWSDedicatedHostToMAPI(spec awsv1.AWSMachineSpec, tenancy mapiv1beta1.InstanceTenancy, fldPath *field.Path) (*mapiv1beta1.HostPlacement, field.ErrorList) {
	var (
		errorList field.ErrorList
		host      *mapiv1beta1.HostPlacement
	)

	if spec.HostAffinity == nil {
		// When HostAffinity is not set, validate that dedicated host fields are also not set.  Default host affinity is default so host ID does not need to be checked.
		if spec.DynamicHostAllocation != nil {
			errorList = append(errorList, field.Invalid(fldPath.Child("dynamicHostAllocation"), spec.DynamicHostAllocation, "dynamicHostAllocation is only allowed when hostAffinity is host"))
		}

		return host, errorList
	}

	switch *spec.HostAffinity {
	case "host":
		host, errorList = convertHostAffinityHost(spec, fldPath)
	case "default":
		host, errorList = convertHostAffinityDefault(spec, tenancy, fldPath)
	default:
		errorList = append(errorList, field.Invalid(fldPath.Child("hostAffinity"), spec.HostAffinity, errUnsupportedHostAffinityType))
	}

	return host, errorList
}

func convertCAPAMachineConditionsToMAPIMachineAWSProviderConditions(awsMachine *awsv1.AWSMachine) []metav1.Condition {
	if ptr.Deref(awsMachine.Status.InstanceState, "") == awsv1.InstanceStateRunning {
		// Set conditionSuccess
		return []metav1.Condition{{
			Type:    string(mapiv1beta1.MachineCreation),
			Status:  metav1.ConditionTrue,
			Reason:  mapiv1beta1.MachineCreationSucceededConditionReason,
			Message: "Machine successfully created",
			// LastTransitionTime will be set by the condition utilities.
		}}
	}

	// Set conditionFailed
	return []metav1.Condition{{
		Type:    string(mapiv1beta1.MachineCreation),
		Status:  metav1.ConditionFalse,
		Reason:  mapiv1beta1.MachineCreationFailedConditionReason,
		Message: "See AWSMachine conditions.",
		// LastTransitionTime will be set by the condition utilities.
	}}
}

// ToMachine converts a capi2mapi MachineAndAWSMachineTemplate into a MAPI Machine.
func (m machineAndAWSMachineAndAWSCluster) ToMachine() (*mapiv1beta1.Machine, []string, error) {
	if m.machine == nil || m.awsMachine == nil || m.awsCluster == nil {
		return nil, nil, errCAPIMachineAWSMachineAWSClusterCannotBeNil
	}

	var errors field.ErrorList

	mapaSpec, warnings, err := m.toProviderSpec()
	if err != nil {
		errors = append(errors, err...)
	}

	awsSpecRawExt, errRaw := RawExtensionFromInterface(mapaSpec)
	if errRaw != nil {
		return nil, nil, fmt.Errorf("unable to convert AWS providerSpec to raw extension: %w", errRaw)
	}

	awsStatusRawExt, errRaw := RawExtensionFromInterface(m.toProviderStatus())
	if errRaw != nil {
		return nil, nil, fmt.Errorf("unable to convert AWS providerStatus to raw extension: %w", errRaw)
	}

	var additionalMachineAPIMetadataLabels, additionalMachineAPIMetadataAnnotations map[string]string
	if !m.excludeMachineAPILabelsAndAnnotations {
		additionalMachineAPIMetadataLabels = map[string]string{
			consts.MAPIMachineMetadataLabelInstanceType: m.awsMachine.Spec.InstanceType,
			consts.MAPIMachineMetadataLabelRegion:       m.awsCluster.Spec.Region,
			consts.MAPIMachineMetadataLabelZone:         m.machine.Spec.FailureDomain,
		}

		additionalMachineAPIMetadataAnnotations = map[string]string{
			consts.MAPIMachineMetadataAnnotationInstanceState: string(ptr.Deref(m.awsMachine.Status.InstanceState, "")),
		}
	}

	mapiMachine, err := fromCAPIMachineToMAPIMachine(m.machine, additionalMachineAPIMetadataLabels, additionalMachineAPIMetadataAnnotations)
	if err != nil {
		errors = append(errors, err...)
	}

	mapiMachine.Spec.ProviderSpec.Value = awsSpecRawExt
	mapiMachine.Status.ProviderStatus = awsStatusRawExt

	if len(errors) > 0 {
		return nil, warnings, errors.ToAggregate()
	}

	return mapiMachine, warnings, nil
}

// ToMachineSet converts a capi2mapi MachineAndAWSMachineTemplate into a MAPI MachineSet.
func (m machineSetAndAWSMachineTemplateAndAWSCluster) ToMachineSet() (*mapiv1beta1.MachineSet, []string, error) {
	if m.machineSet == nil || m.template == nil || m.awsCluster == nil || m.machineAndAWSMachineAndAWSCluster == nil {
		return nil, nil, errCAPIMachineSetAWSMachineTemplateAWSClusterCannotBeNil
	}

	var errors []error

	// Run the full ToMachine conversion so that we can check for
	// any Machine level conversion errors in the spec translation.
	mapiMachine, warnings, err := m.ToMachine()
	if err != nil {
		errors = append(errors, err)
	}

	mapiMachineSet, err := fromCAPIMachineSetToMAPIMachineSet(m.machineSet)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, warnings, utilerrors.NewAggregate(errors)
	}

	mapiMachineSet.Spec.Template.Spec = mapiMachine.Spec

	// Copy the labels and annotations from the Machine to the template.
	mapiMachineSet.Spec.Template.Annotations = mapiMachine.Annotations
	mapiMachineSet.Spec.Template.Labels = mapiMachine.Labels

	return mapiMachineSet, warnings, nil
}

// Conversion helpers.
//
//nolint:unparam // Return empty warnings for consistency
func convertAWSMetadataOptionsToMAPI(fldPath *field.Path, capiMetadataOpts *awsv1.InstanceMetadataOptions) (mapiv1beta1.MetadataServiceOptions, []string, field.ErrorList) {
	var errors field.ErrorList

	if capiMetadataOpts == nil {
		return mapiv1beta1.MetadataServiceOptions{}, nil, nil
	}

	var auth mapiv1beta1.MetadataServiceAuthentication

	switch capiMetadataOpts.HTTPTokens {
	case "":
		// Defaults to optional on both sides.
	case awsv1.HTTPTokensStateOptional:
		auth = mapiv1beta1.MetadataServiceAuthenticationOptional
	case awsv1.HTTPTokensStateRequired:
		auth = mapiv1beta1.MetadataServiceAuthenticationRequired
	default:
		errors = append(errors, field.Invalid(fldPath.Child("httpTokens"), capiMetadataOpts.HTTPTokens, errUnsupportedHTTPTokensState))
	}

	if capiMetadataOpts.HTTPEndpoint != "" && capiMetadataOpts.HTTPEndpoint != awsv1.InstanceMetadataEndpointStateEnabled {
		// This defaults to "enabled" in CAPI and on the AWS side, so if it's not "enabled", the user explicitly chose another option.
		// TODO(OCPCLOUD-2710): We should implement this within MAPI to create feature parity.
		errors = append(errors, field.Invalid(fldPath.Child("httpEndpoint"), capiMetadataOpts.HTTPEndpoint, fmt.Sprintf("httpEndpoint values other than %q are not supported", awsv1.InstanceMetadataEndpointStateEnabled)))
	}

	if capiMetadataOpts.HTTPPutResponseHopLimit != 0 && capiMetadataOpts.HTTPPutResponseHopLimit != 1 {
		// This defaults to 1 in CAPI and on the AWS side, so if it's not 1, the user explicitly chose another option.
		// TODO(OCPCLOUD-2710): We should implement this within MAPI to create feature parity.
		errors = append(errors, field.Invalid(fldPath.Child("httpPutResponseHopLimit"), capiMetadataOpts.HTTPPutResponseHopLimit, "httpPutResponseHopLimit values other than 1 are not supported"))
	}

	if capiMetadataOpts.HTTPProtocolIPv6 != "" && capiMetadataOpts.HTTPProtocolIPv6 != awsv1.InstanceMetadataEndpointStateDisabled {
		// This defaults to "disabled" in CAPI and on the AWS side, so if it's not "disabled", the user explicitly chose another option.
		// TODO(OCPCLOUD-2710): We should implement this within MAPI to create feature parity.
		errors = append(errors, field.Invalid(fldPath.Child("httpProtocolIpv6"), capiMetadataOpts.HTTPProtocolIPv6, fmt.Sprintf("httpProtocolIpv6 values other than %q are not supported", awsv1.InstanceMetadataEndpointStateDisabled)))
	}

	if capiMetadataOpts.InstanceMetadataTags != "" && capiMetadataOpts.InstanceMetadataTags != awsv1.InstanceMetadataEndpointStateDisabled {
		// This defaults to "disabled" in CAPI and on the AWS side, so if it's not "disabled", the user explicitly chose another option.
		// TODO(OCPCLOUD-2710): We should implement this within MAPI to create feature parity.
		errors = append(errors, field.Invalid(fldPath.Child("instanceMetadataTags"), capiMetadataOpts.InstanceMetadataTags, fmt.Sprintf("instanceMetadataTags values other than %q are not supported", awsv1.InstanceMetadataEndpointStateDisabled)))
	}

	metadataOpts := mapiv1beta1.MetadataServiceOptions{
		Authentication: auth,
	}

	if len(errors) > 0 {
		return mapiv1beta1.MetadataServiceOptions{}, nil, errors
	}

	return metadataOpts, nil, nil
}

func convertAWSResourceReferenceToMAPI(capiReference awsv1.AWSResourceReference) mapiv1beta1.AWSResourceReference {
	filters := convertAWSFiltersToMAPI(capiReference.Filters)

	return mapiv1beta1.AWSResourceReference{
		ID:      capiReference.ID,
		Filters: filters,
	}
}

func convertAWSFiltersToMAPI(capiFilters []awsv1.Filter) []mapiv1beta1.Filter {
	return util.SliceMap(capiFilters, func(filter awsv1.Filter) mapiv1beta1.Filter {
		return mapiv1beta1.Filter{
			Name:   filter.Name,
			Values: filter.Values,
		}
	})
}

func convertAWSTagsToMAPI(capiTags awsv1.Tags) []mapiv1beta1.TagSpecification {
	if len(capiTags) == 0 {
		return []mapiv1beta1.TagSpecification{}
	}

	// Collect keys first to ensure deterministic ordering
	keys := make([]string, 0, len(capiTags))
	for key := range capiTags {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	// Build tags slice in sorted order
	mapiTags := make([]mapiv1beta1.TagSpecification, 0, len(capiTags))
	for _, key := range keys {
		mapiTags = append(mapiTags, mapiv1beta1.TagSpecification{
			Name:  key,
			Value: capiTags[key],
		})
	}

	return mapiTags
}

func convertAWSSecurityGroupstoMAPI(sgs []awsv1.AWSResourceReference) []mapiv1beta1.AWSResourceReference {
	return util.SliceMap(sgs, convertAWSResourceReferenceToMAPI)
}

func convertAWSSpotMarketOptionsToMAPI(capiSpotMarketOptions *awsv1.SpotMarketOptions) *mapiv1beta1.SpotMarketOptions {
	if capiSpotMarketOptions == nil {
		return nil
	}

	return &mapiv1beta1.SpotMarketOptions{
		MaxPrice: capiSpotMarketOptions.MaxPrice,
	}
}

func convertAWSTenancyToMAPI(fldPath *field.Path, capiTenancy string) (mapiv1beta1.InstanceTenancy, *field.Error) {
	switch capiTenancy {
	case "default":
		return mapiv1beta1.DefaultTenancy, nil
	case "dedicated":
		return mapiv1beta1.DedicatedTenancy, nil
	case "host":
		return mapiv1beta1.HostTenancy, nil
	case "":
		return "", nil
	default:
		return "", field.Invalid(fldPath, capiTenancy, errUnsupportedCAPATenancy)
	}
}

func convertAWSMarketTypeToMAPI(fldPath *field.Path, marketType awsv1.MarketType) (mapiv1beta1.MarketType, *field.Error) {
	switch marketType {
	case awsv1.MarketTypeOnDemand:
		return mapiv1beta1.MarketTypeOnDemand, nil
	case awsv1.MarketTypeSpot:
		return mapiv1beta1.MarketTypeSpot, nil
	case awsv1.MarketTypeCapacityBlock:
		return mapiv1beta1.MarketTypeCapacityBlock, nil
	case "":
		return "", nil
	default:
		return "", field.Invalid(fldPath, marketType, errUnsupportedCAPAMarketType)
	}
}

func convertAWSVolumesToMAPI(fldPath *field.Path, rootVolume *awsv1.Volume, nonRootVolumes []awsv1.Volume) ([]mapiv1beta1.BlockDeviceMappingSpec, field.ErrorList) {
	var (
		blockDeviceMapping []mapiv1beta1.BlockDeviceMappingSpec
		errors             field.ErrorList
	)

	if rootVolume != nil && *rootVolume != (awsv1.Volume{}) {
		bdm, err := convertAWSVolumeToMAPI(fldPath.Child("rootVolume"), *rootVolume)
		if err != nil {
			errors = append(errors, err)
		} else {
			blockDeviceMapping = append(blockDeviceMapping, bdm)
		}
	}

	for i, volume := range nonRootVolumes {
		bdm, err := convertAWSVolumeToMAPI(fldPath.Child("nonRootVolumes").Index(i), volume)
		if err != nil {
			errors = append(errors, err)
		} else {
			blockDeviceMapping = append(blockDeviceMapping, bdm)
		}
	}

	return blockDeviceMapping, errors
}

func convertAWSVolumeToMAPI(fldPath *field.Path, volume awsv1.Volume) (mapiv1beta1.BlockDeviceMappingSpec, *field.Error) {
	bdm := mapiv1beta1.BlockDeviceMappingSpec{
		EBS: &mapiv1beta1.EBSBlockDeviceSpec{
			VolumeSize: ptr.To(volume.Size),
			Encrypted:  volume.Encrypted,
			KMSKey:     convertAWSKMSKeyToMAPI(volume.EncryptionKey),
		},
	}

	if volume.DeviceName != "" {
		bdm.DeviceName = ptr.To(volume.DeviceName)
	}

	if volume.Type != "" {
		bdm.EBS.VolumeType = ptr.To(string(volume.Type))
	}

	if volume.IOPS != 0 {
		bdm.EBS.Iops = ptr.To(volume.IOPS)
	}

	if volume.Throughput != nil {
		throughput, ok := int64ToInt32(*volume.Throughput)
		if !ok {
			return mapiv1beta1.BlockDeviceMappingSpec{}, field.Invalid(fldPath.Child("throughput"), *volume.Throughput, "throughput exceeds maximum int32 value")
		}

		bdm.EBS.ThroughputMib = ptr.To(throughput)
	}

	return bdm, nil
}

func convertAWSKMSKeyToMAPI(kmsKey string) mapiv1beta1.AWSResourceReference {
	if strings.HasPrefix(kmsKey, "arn:") {
		return mapiv1beta1.AWSResourceReference{
			ARN: &kmsKey,
		}
	}

	return mapiv1beta1.AWSResourceReference{
		ID: &kmsKey,
	}
}

// int64ToInt32 converts an int64 to an int32 if it is within the int32 range.
func int64ToInt32(in int64) (int32, bool) {
	if in > math.MaxInt32 || in < math.MinInt32 {
		return 0, false
	}

	return int32(in), true
}

func convertAWSNetworkInterfaceTypeToMAPI(networkInterfaceType awsv1.NetworkInterfaceType) mapiv1beta1.AWSNetworkInterfaceType {
	switch networkInterfaceType {
	case awsv1.NetworkInterfaceTypeEFAWithENAInterface:
		return mapiv1beta1.AWSEFANetworkInterfaceType
	case awsv1.NetworkInterfaceTypeENI:
		return mapiv1beta1.AWSENANetworkInterfaceType
	}

	return ""
}

// handleUnsupportedAWSMachineFields returns an error for every present field in the AWSMachineSpec that
// we are currently, or indefinitely not supporting.
// These are protected by VAPs so should never actually cause an error here.
func handleUnsupportedAWSMachineFields(fldPath *field.Path, spec awsv1.AWSMachineSpec) field.ErrorList {
	errs := field.ErrorList{}

	if spec.AMI.EKSOptimizedLookupType != nil {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("ami", "eksOptimizedLookupType"), spec.AMI.EKSOptimizedLookupType, "eksOptimizedLookupType is not supported"))
	}

	if spec.ImageLookupFormat != "" {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("imageLookupFormat"), spec.ImageLookupFormat, "imageLookupFormat is not supported"))
	}

	if spec.ImageLookupOrg != "" {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("imageLookupOrg"), spec.ImageLookupOrg, "imageLookupOrg is not supported"))
	}

	if spec.ImageLookupBaseOS != "" {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("imageLookupBaseOS"), spec.ImageLookupBaseOS, "imageLookupBaseOS is not supported"))
	}

	if len(spec.SecurityGroupOverrides) > 0 {
		// We do not support SecurityGroupOverrides being used, because the externally managed annotation that we add updates the behaviour to stop this.
		errs = append(errs, field.Invalid(fldPath.Child("securityGroupOverrides"), spec.SecurityGroupOverrides, "securityGroupOverrides are not supported"))
	}

	if len(spec.NetworkInterfaces) > 0 {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("networkInterfaces"), spec.NetworkInterfaces, "networkInterfaces are not supported"))
	}

	if spec.UncompressedUserData != nil {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("uncompressedUserData"), spec.UncompressedUserData, "uncompressedUserData is not supported"))
	}

	if (spec.CloudInit != awsv1.CloudInit{}) {
		// Not required for our use case.
		errs = append(errs, field.Invalid(fldPath.Child("cloudInit"), spec.CloudInit, "cloudInit is not supported"))
	}

	// privateDNSName is not checked here because the relevant information is available on the
	// infrastructure CR, so nothing additional needs to be added to the MAPI machine spec.

	// assignPrimaryIPv6 is not checked here because the relevant information is available on the
	// infrastructure CR, so nothing additional needs to be added to the MAPI machine spec.

	if spec.Ignition != nil {
		if spec.Ignition.Proxy != nil {
			// Ignition proxy is not configurable in MAPI. Not required for our use case.
			errs = append(errs, field.Invalid(fldPath.Child("ignition", "proxy"), spec.Ignition.Proxy, "ignition proxy is not supported"))
		}

		if spec.Ignition.TLS != nil {
			// Ignition TLS is not configurable in MAPI. Not required for our use case.
			errs = append(errs, field.Invalid(fldPath.Child("ignition", "tls"), spec.Ignition.TLS, "ignition tls is not supported"))
		}
	}

	return errs
}

// handleAWSIdentityRef returns errors if the configuration IdentityRef is different from OCP defaults, and the default credential reference otherwise.
// We only support the ControllerIdentityKind, which is the upstream default, when converting.
// This default is what will happen when no IdentityRef is defined, so support both hard-coded values and the empty reference.
func handleAWSIdentityRef(fldPath *field.Path, identityRef *awsv1.AWSIdentityReference) (*corev1.LocalObjectReference, field.ErrorList) {
	errs := field.ErrorList{}

	ref := &corev1.LocalObjectReference{
		Name: defaultCredentialsSecretName,
	}

	// An unset identityref will use the default values.
	// This also protects against nil lookups below.
	if identityRef == nil {
		return ref, nil
	}

	if identityRef.Kind != awsv1.ControllerIdentityKind && identityRef.Kind != "" {
		errs = append(errs, field.Invalid(fldPath.Child("kind"), identityRef.Kind, fmt.Sprintf("kind %q cannot be converted to CredentialsSecret. Please see https://access.redhat.com/articles/7116313 for more details.", identityRef.Kind)))
	}

	if identityRef.Name != defaultIdentityName && identityRef.Name != "" {
		errs = append(errs, field.Invalid(fldPath.Child("name"), identityRef.Name, fmt.Sprintf("name %q must be %q when using an AWSClusterControllerIdentity. Please see https://access.redhat.com/articles/7116313 for more details.", identityRef.Name, defaultIdentityName)))
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Assume we're using the defaults.
	return ref, nil
}

// convertAWSClusterLoadBalancersToMAPI convert CAPI LoadBalancers from the AWSCluster spec to MAPI LoadBalancerReferences on the Machine.
func convertAWSClusterLoadBalancersToMAPI(fldPath *field.Path, machine *clusterv1.Machine, awsCluster *awsv1.AWSCluster) ([]mapiv1beta1.LoadBalancerReference, field.ErrorList) {
	var loadBalancers []mapiv1beta1.LoadBalancerReference

	errs := field.ErrorList{}

	if !capiutil.IsControlPlaneMachine(machine) {
		// No loadbalancer on non-control plane machines.
		return nil, nil
	}

	internalLoadBalancerRef, err := ConvertAWSLoadBalancerToMAPI(awsCluster.Spec.ControlPlaneLoadBalancer)
	if err != nil {
		errs = append(errs, field.Invalid(fldPath.Child("controlPlaneLoadBalancer"), awsCluster.Spec.ControlPlaneLoadBalancer, fmt.Errorf("failed to convert load balancer: %w", err).Error()))
	} else {
		loadBalancers = append(loadBalancers, internalLoadBalancerRef)
	}

	if awsCluster.Spec.SecondaryControlPlaneLoadBalancer != nil {
		externalLoadBalancerRef, err := ConvertAWSLoadBalancerToMAPI(awsCluster.Spec.SecondaryControlPlaneLoadBalancer)
		if err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("secondaryControlPlaneLoadBalancer"), awsCluster.Spec.SecondaryControlPlaneLoadBalancer, fmt.Errorf("failed to convert load balancer: %w", err).Error()))
		} else {
			loadBalancers = append(loadBalancers, externalLoadBalancerRef)
		}
	}

	return loadBalancers, errs
}

// ConvertAWSLoadBalancerToMAPI converts CAPI AWSLoadBalancerSpec to MAPI LoadBalancerReference.
func ConvertAWSLoadBalancerToMAPI(loadBalancer *awsv1.AWSLoadBalancerSpec) (mapiv1beta1.LoadBalancerReference, error) {
	if loadBalancer == nil {
		return mapiv1beta1.LoadBalancerReference{}, errNilLoadBalancer
	}

	switch loadBalancer.LoadBalancerType {
	case awsv1.LoadBalancerTypeClassic, awsv1.LoadBalancerTypeELB:
		return mapiv1beta1.LoadBalancerReference{
			Name: ptr.Deref(loadBalancer.Name, ""),
			Type: mapiv1beta1.ClassicLoadBalancerType,
		}, nil
	case awsv1.LoadBalancerTypeNLB:
		return mapiv1beta1.LoadBalancerReference{
			Name: ptr.Deref(loadBalancer.Name, ""),
			Type: mapiv1beta1.NetworkLoadBalancerType,
		}, nil
	default:
		return mapiv1beta1.LoadBalancerReference{}, errUnsupportedLoadBalancerType
	}
}

// ConvertAWSCPUOptionsToMAPI converts CAPI CPUOptions to MAPI CPUOptions.
func ConvertAWSCPUOptionsToMAPI(cpuOptions awsv1.CPUOptions) *mapiv1beta1.CPUOptions {
	mapiCPUOptions := &mapiv1beta1.CPUOptions{}

	switch cpuOptions.ConfidentialCompute {
	case awsv1.AWSConfidentialComputePolicyDisabled:
		mapiCPUOptions.ConfidentialCompute = ptr.To(mapiv1beta1.AWSConfidentialComputePolicyDisabled)
	case awsv1.AWSConfidentialComputePolicySEVSNP:
		mapiCPUOptions.ConfidentialCompute = ptr.To(mapiv1beta1.AWSConfidentialComputePolicySEVSNP)
	}

	if *mapiCPUOptions == (mapiv1beta1.CPUOptions{}) {
		return nil
	}

	return mapiCPUOptions
}

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
package capi2mapi

import (
	"strings"

	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	mapiNamespace = "openshift-machine-api"
)

// fromCAPIMachineToMAPIMachine translates a core CAPI Machine to its MAPI Machine correspondent.
func fromCAPIMachineToMAPIMachine(capiMachine *clusterv1.Machine, additionalMachineAPIMetadataLabels, additionalMachineAPIMetadataAnnotations map[string]string) (*mapiv1beta1.Machine, field.ErrorList) {
	errs := field.ErrorList{}

	lifecycleHooks, capiMachineNonHookAnnotations := convertCAPILifecycleHookAnnotationsToMAPILifecycleHooksAndAnnotations(capiMachine.Annotations)

	mapiMachineStatus, machineStatusErrs := convertCAPIMachineStatusToMAPI(capiMachine.Status)
	if len(machineStatusErrs) > 0 {
		errs = append(errs, machineStatusErrs...)
	}

	var providerID *string
	if capiMachine.Spec.ProviderID != "" {
		providerID = ptr.To(capiMachine.Spec.ProviderID)
	}

	mapiMachine := &mapiv1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:            capiMachine.Name,
			Namespace:       mapiNamespace,
			Labels:          convertCAPILabelsToMAPILabels(capiMachine.Labels, additionalMachineAPIMetadataLabels),
			Annotations:     convertCAPIAnnotationsToMAPIAnnotations(capiMachineNonHookAnnotations, additionalMachineAPIMetadataAnnotations),
			Finalizers:      []string{mapiv1beta1.MachineFinalizer},
			OwnerReferences: nil, // OwnerReferences not populated here. They are added later by the machineSync controller.
		},
		Spec: mapiv1beta1.MachineSpec{
			ObjectMeta: mapiv1beta1.ObjectMeta{
				Labels:      convertCAPIMachineLabelsToMAPIMachineSpecObjectMetaLabels(capiMachine.Labels),
				Annotations: convertCAPIMachineAnnotationsToMAPIMachineSpecObjectMetaAnnotations(capiMachineNonHookAnnotations),
			},
			ProviderID:     providerID,
			LifecycleHooks: lifecycleHooks,
			// ProviderSpec: // ProviderSpec MUST NOT be populated here. It is added later by higher level fuctions.
			// Taints: // TODO(OCPCLOUD-2861): Taint propagation from Machines to Nodes is not yet implemented in CAPI.
		},
		Status: mapiMachineStatus,
	}

	// Unused fields - Below this line are fields not used from the CAPI Machine.
	// capiMachine.ObjectMeta.OwnerReferences - handled by the machineSync controller.

	// capiMachine.Spec.ClusterName - Ignore this as it can be reconstructed from the infra object.
	// capiMachine.Spec.Bootstrap.ConfigRef - Ignore as we use DataSecretName for the MAPI side.
	// capiMachine.Spec.InfrastructureRef - Ignore as this is the split between 1 to 2 resources from MAPI to CAPI.
	// capiMachine.Spec.FailureDomain - Ignore because we use this to populate the providerSpec.

	if capiMachine.Spec.Version != "" {
		// TODO(OCPCLOUD-2714): We should prevent this using a VAP until and unless we need to support the field.
		errs = append(errs, field.Invalid(field.NewPath("spec", "version"), capiMachine.Spec.Version, "version is not supported"))
	}

	if capiMachine.Spec.Deletion.NodeDrainTimeoutSeconds != nil {
		// TODO(OCPCLOUD-2715): We should implement this within MAPI to create feature parity.
		errs = append(errs, field.Invalid(field.NewPath("spec", "deletion", "nodeDrainTimeoutSeconds"), capiMachine.Spec.Deletion.NodeDrainTimeoutSeconds, "nodeDrainTimeoutSeconds is not supported"))
	}

	if capiMachine.Spec.Deletion.NodeVolumeDetachTimeoutSeconds != nil {
		// TODO(OCPCLOUD-2715): We should implement this within MAPI to create feature parity.
		errs = append(errs, field.Invalid(field.NewPath("spec", "deletion", "nodeVolumeDetachTimeoutSeconds"), capiMachine.Spec.Deletion.NodeVolumeDetachTimeoutSeconds, "nodeVolumeDetachTimeoutSeconds is not supported"))
	}

	if capiMachine.Spec.Deletion.NodeDeletionTimeoutSeconds != nil {
		// TODO(docs): document this.
		// We tolerate if the NodeDeletionTimeout is set to the CAPI default of 10s,
		// as CAPI automatically sets this on the machine when we convert MAPI->CAPI.
		// Otherwise if it is set to a non-default value we fail.
		if *capiMachine.Spec.Deletion.NodeDeletionTimeoutSeconds != 10 {
			// TODO(OCPCLOUD-2715): We should implement this within MAPI to create feature parity.
			errs = append(errs, field.Invalid(field.NewPath("spec", "deletion", "nodeDeletionTimeoutSeconds"), capiMachine.Spec.Deletion.NodeDeletionTimeoutSeconds, "nodeDeletionTimeoutSeconds is not supported"))
		}
	}

	if len(errs) > 0 {
		// Return the mapiMachine so that the logic continues and collects all possible conversion errors.
		return mapiMachine, errs
	}

	return mapiMachine, nil
}

// convertCAPIMachineStatusToMAPI converts a CAPI MachineStatus to MAPI format.
func convertCAPIMachineStatusToMAPI(capiStatus clusterv1.MachineStatus) (mapiv1beta1.MachineStatus, field.ErrorList) {
	errs := field.ErrorList{}

	addresses, addressesErr := convertCAPIMachineAddressesToMAPI(capiStatus.Addresses)
	if addressesErr != nil {
		errs = append(errs, addressesErr...)
	}

	var nodeRef *corev1.ObjectReference
	if capiStatus.NodeRef.IsDefined() {
		nodeRef = &corev1.ObjectReference{
			Name:       capiStatus.NodeRef.Name,
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Node",
		}
	}

	var lastUpdated *metav1.Time
	if !capiStatus.LastUpdated.Equal(&metav1.Time{}) {
		lastUpdated = ptr.To(capiStatus.LastUpdated)
	}

	mapiStatus := mapiv1beta1.MachineStatus{
		NodeRef:     nodeRef,
		LastUpdated: lastUpdated,
		// Conditions:   // TODO(OCPCLOUD-3193): Add MAPI conditions when they are implemented.
		ErrorReason:  convertCAPIMachineFailureReasonToMAPIErrorReason(capiStatus.Deprecated),
		ErrorMessage: convertCAPIMachineFailureMessageToMAPIErrorMessage(capiStatus.Deprecated),
		Phase:        convertCAPIMachinePhaseToMAPI(capiStatus.Phase),
		Addresses:    addresses,

		// LastOperation // this is MAPI-specific and not used in CAPI.
		// ObservedGeneration: // We don't set the observed generation at this stage as it is handled by the machineSync controller.
		// AuthoritativeAPI: // Ignore, this field as it is not present in CAPI.
		// SynchronizedGeneration: // Ignore, this field as it is not present in CAPI.
	}

	// unused fields from CAPI MachineStatus
	// - NodeInfo: not present on the MAPI Machine status.
	// - CertificatesExpiryDate: not present on the MAPI Machine status.
	// - BootstrapReady: this is derived and not stored directly in MAPI.
	// - InfrastructureReady: this is derived and not stored directly in MAPI.
	// - Deletion: not present on the MAPI Machine status.
	// - V1Beta2: for now we use the V1Beta1 status fields to obtain the status of the MAPI Machine.

	return mapiStatus, errs
}

// convertCAPIMachineAddressesToMAPI converts CAPI machine addresses to MAPI format.
func convertCAPIMachineAddressesToMAPI(capiAddresses clusterv1.MachineAddresses) ([]corev1.NodeAddress, field.ErrorList) {
	if capiAddresses == nil {
		return nil, nil
	}

	errs := field.ErrorList{}
	mapiAddresses := make([]corev1.NodeAddress, 0, len(capiAddresses))

	// Addresses are slightly different between MAPI/CAPI.
	for _, addr := range capiAddresses {
		switch addr.Type {
		case clusterv1.MachineHostName:
			mapiAddresses = append(mapiAddresses, corev1.NodeAddress{Type: corev1.NodeHostName, Address: addr.Address})
		case clusterv1.MachineExternalIP:
			mapiAddresses = append(mapiAddresses, corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: addr.Address})
		case clusterv1.MachineInternalIP:
			mapiAddresses = append(mapiAddresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: addr.Address})
		case clusterv1.MachineExternalDNS:
			mapiAddresses = append(mapiAddresses, corev1.NodeAddress{Type: corev1.NodeExternalDNS, Address: addr.Address})
		case clusterv1.MachineInternalDNS:
			mapiAddresses = append(mapiAddresses, corev1.NodeAddress{Type: corev1.NodeInternalDNS, Address: addr.Address})
		default:
			errs = append(errs, field.Invalid(field.NewPath("status", "addresses"), string(addr.Type), string(addr.Type)+" unrecognized address type"))
		}
	}

	return mapiAddresses, errs
}

// convertCAPIMachinePhaseToMAPI converts CAPI machine phase to MAPI format.
func convertCAPIMachinePhaseToMAPI(capiPhase string) *string {
	// Phase is slightly different between MAPI/CAPI.
	// In CAPI can be one of: Pending;Provisioning;Provisioned;Running;Deleting;Deleted;Failed;Unknown
	// In MAPI can be one of: Provisioning;Provisioned;Running;Deleting;Failed (missing Pending,Deleted,Unknown)
	switch capiPhase {
	case "":
		return nil // Empty is equivalent to nil in MAPI.
	case "Pending":
		return nil // Pending is not supported in MAPI but is is a very early state so we don't need to represent it.
	case "Deleted":
		return ptr.To("Deleting") // Deleted is not supported in MAPI but we can stay in Deleting until the machine is fully removed.
	case "Unknown":
		return nil // Unknown is not supported in MAPI but we can set it to nil until we know more.
	case "Provisioning", "Provisioned", "Running", "Deleting", "Failed":
		return &capiPhase // This is a supported phase so we can represent it in MAPI.
	default:
		return nil // This is an unknown phase so we can't represent it in MAPI.
	}
}

// convertCAPIMachineFailureReasonToMAPIErrorReason converts CAPI MachineStatusError to MAPI MachineStatusError.
func convertCAPIMachineFailureReasonToMAPIErrorReason(capiDeprecatedStatus *clusterv1.MachineDeprecatedStatus) *mapiv1beta1.MachineStatusError {
	if capiDeprecatedStatus == nil || capiDeprecatedStatus.V1Beta1 == nil || capiDeprecatedStatus.V1Beta1.FailureReason == nil {
		return nil
	}

	mapiErrorReason := mapiv1beta1.MachineStatusError(*capiDeprecatedStatus.V1Beta1.FailureReason)

	return &mapiErrorReason
}

// convertCAPIMachineFailureMessageToMAPIErrorMessage converts CAPI MachineStatusError to MAPI MachineStatusError.
func convertCAPIMachineFailureMessageToMAPIErrorMessage(capiDeprecatedStatus *clusterv1.MachineDeprecatedStatus) *string {
	if capiDeprecatedStatus == nil || capiDeprecatedStatus.V1Beta1 == nil || capiDeprecatedStatus.V1Beta1.FailureMessage == nil {
		return nil
	}

	mapiErrorMessage := *capiDeprecatedStatus.V1Beta1.FailureMessage

	return &mapiErrorMessage
}

const (
	// Note the trailing slash here is important when we are trimming the prefix.
	capiPreDrainAnnotationPrefix     = clusterv1.PreDrainDeleteHookAnnotationPrefix + "/"
	capiPreTerminateAnnotationPrefix = clusterv1.PreTerminateDeleteHookAnnotationPrefix + "/"
)

// convertCAPILifecycleHookAnnotationsToMAPILifecycleHooksAndAnnotations extracts the lifecycle hooks from the CAPI Machine annotations.
func convertCAPILifecycleHookAnnotationsToMAPILifecycleHooksAndAnnotations(capiAnnotations map[string]string) (mapiv1beta1.LifecycleHooks, map[string]string) {
	hooks := mapiv1beta1.LifecycleHooks{}
	newAnnotations := make(map[string]string)

	for k, v := range capiAnnotations {
		switch {
		case strings.HasPrefix(k, capiPreDrainAnnotationPrefix):
			hooks.PreDrain = append(hooks.PreDrain, mapiv1beta1.LifecycleHook{
				Name:  strings.TrimPrefix(k, capiPreDrainAnnotationPrefix),
				Owner: v,
			})
		case strings.HasPrefix(k, capiPreTerminateAnnotationPrefix):
			hooks.PreTerminate = append(hooks.PreTerminate, mapiv1beta1.LifecycleHook{
				Name:  strings.TrimPrefix(k, capiPreTerminateAnnotationPrefix),
				Owner: v,
			})
		default:
			// Carry over only the non lifecycleHooks annotations.
			newAnnotations[k] = v
		}
	}

	return hooks, newAnnotations
}

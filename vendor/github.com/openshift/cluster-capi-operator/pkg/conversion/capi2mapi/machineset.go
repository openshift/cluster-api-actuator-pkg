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
	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capierrors "sigs.k8s.io/cluster-api/errors"
)

// fromCAPIMachineSetToMAPIMachineSet takes a CAPI MachineSet and returns a converted MAPI MachineSet.
func fromCAPIMachineSetToMAPIMachineSet(capiMachineSet *clusterv1.MachineSet) (*mapiv1beta1.MachineSet, error) {
	errs := field.ErrorList{}

	mapiMachineSet := &mapiv1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            capiMachineSet.Name,
			Namespace:       capiMachineSet.Namespace,
			Labels:          convertCAPILabelsToMAPILabels(capiMachineSet.Labels, nil),
			Annotations:     convertCAPIAnnotationsToMAPIAnnotations(capiMachineSet.Annotations, nil),
			Finalizers:      nil, // MAPI MachineSet does not have finalizers.
			OwnerReferences: nil, // OwnerReferences not populated here. They are added later by the machineSetSync controller.
		},
		Spec: mapiv1beta1.MachineSetSpec{
			Selector:        convertCAPIMachineSetSelectorToMAPI(capiMachineSet.Spec.Selector),
			Replicas:        capiMachineSet.Spec.Replicas,
			MinReadySeconds: ptr.Deref(capiMachineSet.Spec.Template.Spec.MinReadySeconds, 0),
			DeletePolicy:    string(capiMachineSet.Spec.Deletion.Order),
			Template: mapiv1beta1.MachineTemplateSpec{
				ObjectMeta: mapiv1beta1.ObjectMeta{
					Labels:      convertCAPILabelsToMAPILabels(capiMachineSet.Spec.Template.Labels, nil),
					Annotations: convertCAPIAnnotationsToMAPIAnnotations(capiMachineSet.Spec.Template.Annotations, nil),
				},
			},
		},
		Status: convertCAPIMachineSetStatusToMAPI(capiMachineSet.Status),
	}

	// Unused fields - Below this line are fields not used from the CAPI Machine.
	// metadata.OwnerReferences - handled by the machineSetSync controller.
	// capiMachineSet.Spec.ClusterName - Ignore this as it can be reconstructed from the infra object.
	// capiMachineSet.Spec.Template.Spec - Ignore as we convert this at a higher level using the Machine conversion logic.
	// capiMachineSet.Spec.MachineNamingStrategy - Not supported in MAPI, no equivalent field exists.

	if len(errs) > 0 {
		// Return the mapiMachine so that the logic continues and collects all possible conversion errors.
		return mapiMachineSet, errs.ToAggregate()
	}

	return mapiMachineSet, nil
}

// convertCAPIMachineSetStatusToMAPI converts a CAPI MachineSetStatus to MAPI format.
func convertCAPIMachineSetStatusToMAPI(capiStatus clusterv1.MachineSetStatus) mapiv1beta1.MachineSetStatus {
	deprecatedStatus := ptr.Deref(capiStatus.Deprecated, clusterv1.MachineSetDeprecatedStatus{})
	v1beta1Status := ptr.Deref(deprecatedStatus.V1Beta1, clusterv1.MachineSetV1Beta1DeprecatedStatus{})
	mapiStatus := mapiv1beta1.MachineSetStatus{
		Replicas:             ptr.Deref(capiStatus.Replicas, 0),
		FullyLabeledReplicas: v1beta1Status.FullyLabeledReplicas,
		ReadyReplicas:        v1beta1Status.ReadyReplicas,
		AvailableReplicas:    v1beta1Status.AvailableReplicas,
		LabelSelector:        capiStatus.Selector,
		// ObservedGeneration: // We don't set the observed generation at this stage as it is handled by the machineSetSync controller.
		// AuthoritativeAPI: // Ignore, this field as it is not present in CAPI.
		// SynchronizedGeneration: // Ignore, this field as it is not present in CAPI.

		// The only two conditions normally used for MAPI MachineSets are Paused and Synchronized.
		// We do not convert these conditions to MAPI conditions as they are managed directly by the machineSet sync and migration controllers.
		Conditions: nil,
	}

	// Convert FailureReason/FailureMessage to ErrorReason/ErrorMessage
	if v1beta1Status.FailureReason != nil {
		mapiStatus.ErrorReason = convertCAPIFailureReasonToMAPIErrorReason(*v1beta1Status.FailureReason)
	}

	if v1beta1Status.FailureMessage != nil {
		mapiStatus.ErrorMessage = v1beta1Status.FailureMessage
	}

	// unused fields from CAPI MachineSetStatus
	// - V1Beta2: for now we use the V1Beta1 status fields to obtain the status of the MAPI MachineSet.

	return mapiStatus
}

// convertCAPIFailureReasonToMAPIErrorReason converts CAPI MachineSetStatusError to MAPI MachineSetStatusError.
func convertCAPIFailureReasonToMAPIErrorReason(capiFailureReason capierrors.MachineSetStatusError) *mapiv1beta1.MachineSetStatusError {
	mapiErrorReason := mapiv1beta1.MachineSetStatusError(capiFailureReason)
	return &mapiErrorReason
}

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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift/cluster-capi-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// RawExtensionFromInterface marshals the machine provider spec.
func RawExtensionFromInterface(spec interface{}) (*runtime.RawExtension, error) {
	if spec == nil {
		return nil, nil //nolint:nilnil
	}

	rawBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from RawExtension: %w", err)
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

func convertCAPIMachineSetSelectorToMAPI(capiSelector metav1.LabelSelector) metav1.LabelSelector {
	mapiSelector := capiSelector.DeepCopy()
	mapiSelector.MatchLabels = convertCAPILabelsToMAPILabels(capiSelector.MatchLabels, nil)

	return *mapiSelector
}

func convertCAPILabelsToMAPILabels(capiLabels map[string]string, machineAPILabels map[string]string) map[string]string {
	if len(capiLabels) == 0 && len(machineAPILabels) == 0 {
		return nil
	}

	mapiLabels := make(map[string]string)

	for k, v := range capiLabels {
		// Transform specific node-role label.
		if strings.HasPrefix(k, clusterv1.NodeRoleLabelPrefix) {
			if _, role, found := strings.Cut(k, "/"); found {
				mapiLabels["machine.openshift.io/cluster-api-machine-type"] = role
				mapiLabels["machine.openshift.io/cluster-api-machine-role"] = role

				continue
			} // Otherwise if it is a non conformant node-role label, fallthrough.
		}

		// Ignore CAPI-specific labels that are not explicitly handled.
		if strings.Contains(k, "cluster.x-k8s.io/") {
			continue
		}

		// Default case - copy over the label as-is to MAPI.
		mapiLabels[k] = v
	}

	for k, v := range machineAPILabels {
		// Ignore empty labels to ensure to not overwrite potentially existing labels with empty values.
		if v == "" {
			continue
		}

		mapiLabels[k] = v
	}

	// On the original MAPI object some label fields are nil rather than empty.
	// So return nil instead to avoid unnecessary diff being picked up by the diff checker.
	if len(mapiLabels) == 0 {
		return nil
	}

	return mapiLabels
}

func convertCAPIMachineLabelsToMAPIMachineSpecObjectMetaLabels(capiLabels map[string]string) map[string]string {
	if len(capiLabels) == 0 {
		return nil
	}

	mapiLabels := make(map[string]string)

	for k, v := range capiLabels {
		// Ignore CAPI-specific labels that are not explicitly handled.
		if strings.Contains(k, "cluster.x-k8s.io/") {
			continue
		}

		mapiLabels[k] = v
	}

	return mapiLabels
}

func convertCAPIMachineAnnotationsToMAPIMachineSpecObjectMetaAnnotations(capiAnnotations map[string]string) map[string]string {
	if len(capiAnnotations) == 0 {
		return nil
	}

	mapiAnnotations := make(map[string]string)

	for k, v := range capiAnnotations {
		// Ignore CAPI-specific annotations that are not explicitly handled.
		if strings.Contains(k, "cluster.x-k8s.io/") {
			continue
		}

		mapiAnnotations[k] = v
	}

	return mapiAnnotations
}

func convertCAPIAnnotationsToMAPIAnnotations(capiAnnotations map[string]string, machineAPIAnnotations map[string]string) map[string]string {
	if len(capiAnnotations) == 0 && len(machineAPIAnnotations) == 0 {
		return nil
	}

	mapiAnnotations := make(map[string]string)

	toNotConvertAnnotations := sets.New(
		// We want to skip the CAPI paused annotation to be copied over to MAPI.
		clusterv1.PausedAnnotation,
	)

	for k, v := range capiAnnotations {
		if toNotConvertAnnotations.Has(k) {
			// Skip this annotation.
			continue
		}

		if k == clusterv1.DeleteMachineAnnotation {
			mapiAnnotations[util.MapiDeleteMachineAnnotation] = v
			continue
		}

		// Ignore CAPI-specific annotations that are not explicitly handled.
		if strings.Contains(k, "cluster.x-k8s.io/") {
			continue
		}

		mapiAnnotations[k] = v
	}

	for k, v := range machineAPIAnnotations {
		// Ignore empty annotations to ensure to not overwrite potentially existing annotations with empty values.
		if v == "" {
			continue
		}

		mapiAnnotations[k] = v
	}

	// On the original MAPI object some label fields are nil rather than empty.
	// So return nil instead to avoid unnecessary diff being picked up by the diff checker.
	if len(mapiAnnotations) == 0 {
		return nil
	}

	return mapiAnnotations
}

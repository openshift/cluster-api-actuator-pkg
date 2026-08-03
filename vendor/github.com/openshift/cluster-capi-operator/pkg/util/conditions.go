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
package util

import (
	"errors"
	"fmt"
	"time"

	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/conditions"
	deprecatedv1beta1conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	v1beta2conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errStatusNotMap                 = errors.New("unable to assert status structure to map")
	errStatusConditionsNotInterface = errors.New("unable to assert status.condition structure to interface")
)

// GetCondition retrieves a specific condition from a client.Object.
func GetCondition(obj client.Object, conditionType string) (*metav1.Condition, error) {
	// Convert client.Object to unstructured.Unstructured
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	data := unstructuredObj.UnstructuredContent()

	status, ok := data["status"].(map[string]interface{})
	if !ok {
		return nil, errStatusNotMap
	}

	conditions, ok := status["conditions"].([]interface{})
	if !ok {
		return nil, errStatusConditionsNotInterface
	}

	for _, c := range conditions {
		condMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		status, ok := condMap["status"].(string)
		if !ok {
			continue
		}

		if condMap["type"] == conditionType {
			return &metav1.Condition{
				Type:               conditionType,
				Status:             metav1.ConditionStatus(status),
				Reason:             getString(condMap, "reason"),
				Message:            getString(condMap, "message"),
				LastTransitionTime: getTime(condMap, "lastTransitionTime"),
			}, nil
		}
	}

	return nil, nil //nolint:nilnil
}

// GetConditionStatus returns the status for the condition.
func GetConditionStatus(obj client.Object, conditionType string) (corev1.ConditionStatus, error) {
	cond, err := GetCondition(obj, conditionType)
	if err != nil {
		return corev1.ConditionUnknown, fmt.Errorf("unable to get condition %q for the object: %w", conditionType, err)
	}

	if cond == nil {
		return corev1.ConditionUnknown, nil
	}

	return corev1.ConditionStatus(cond.Status), nil
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}

	return ""
}

func getTime(data map[string]interface{}, key string) metav1.Time {
	if val, ok := data[key].(string); ok {
		parsedTime, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return metav1.Time{Time: parsedTime}
		}
	}

	return metav1.Time{}
}

// GetMAPICondition retrieves a specific condition from a list of MAPI conditions.
func GetMAPICondition(conditions []mapiv1beta1.Condition, conditionType string) *mapiv1beta1.Condition {
	for i := range conditions {
		if string(conditions[i].Type) == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// SetMAPICondition sets a condition in a list of MAPI conditions.
// If the condition already exists and state (Status, Reason, Message) has changed:
// - if the lasttransitiontime is not set, it sets it to the current time
// - if the lasttransitiontime is set, it updates it with the one of the newly provided condition lasttransitiontime.
// If the condition state has not changed, it preserves the existing LastTransitionTime.
// If the condition does not exist, it adds it.
// This function behaves similarly to conditions.Set() for CAPI conditions.
//
//nolint:dupl
func SetMAPICondition(conditions []mapiv1beta1.Condition, condition *mapiv1beta1.Condition) []mapiv1beta1.Condition {
	for i, currCondition := range conditions {
		if currCondition.Type != condition.Type {
			continue
		}

		updatedCondition := *condition

		// Check if the condition state has changed (Status, Reason, Message)
		if currCondition.Status != condition.Status || currCondition.Reason != condition.Reason || currCondition.Message != condition.Message {
			// State has changed, update the condition with the new LastTransitionTime
			if updatedCondition.LastTransitionTime.IsZero() {
				updatedCondition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
			}
		} else {
			// State hasn't changed, preserve the existing LastTransitionTime
			updatedCondition.LastTransitionTime = currCondition.LastTransitionTime
		}

		conditions[i] = updatedCondition

		// Condition found and updated, return the updated conditions.
		return conditions
	}

	// Ensure LastTransitionTime is set also for new conditions.
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}

	condition.LastTransitionTime.Time = condition.LastTransitionTime.Truncate(1 * time.Second)

	// Condition doesn't exist, add it
	return append(conditions, *condition)
}

// SetMAPIProviderCondition sets a condition in a list of Machine API conditions.
// If the condition already exists and state (Status, Reason, Message) has changed:
// - if the lasttransitiontime is not set, it sets it to the current time
// - if the lasttransitiontime is set, it updates it with the one of the newly provided condition lasttransitiontime.
// If the condition state has not changed, it preserves the existing LastTransitionTime.
// If the condition does not exist, it adds it.
// This function behaves similarly to conditions.Set() for Cluster API conditions.
//
//nolint:dupl
func SetMAPIProviderCondition(conditions []metav1.Condition, condition *metav1.Condition) []metav1.Condition {
	for i, currCondition := range conditions {
		if currCondition.Type != condition.Type {
			continue
		}

		updatedCondition := *condition

		// Check if the condition state has changed (Status, Reason, Message)
		if currCondition.Status != condition.Status || currCondition.Reason != condition.Reason || currCondition.Message != condition.Message {
			// State has changed, update the condition with the new LastTransitionTime
			if updatedCondition.LastTransitionTime.IsZero() {
				updatedCondition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
			}
		} else {
			// State hasn't changed, preserve the existing LastTransitionTime
			updatedCondition.LastTransitionTime = currCondition.LastTransitionTime
		}

		conditions[i] = updatedCondition

		// Condition found and updated, return the updated conditions.
		return conditions
	}

	// Ensure LastTransitionTime is set also for new conditions.
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}

	condition.LastTransitionTime.Time = condition.LastTransitionTime.Truncate(1 * time.Second)

	// Condition doesn't exist, add it
	return append(conditions, *condition)
}

// EnsureCAPIV1Beta1Conditions iterates over all CAPI v1beta1 conditions and sets them on the converted object.
func EnsureCAPIV1Beta1Conditions(existing v1beta1conditions.Setter, converted v1beta1conditions.Setter) {
	// Merge the v1beta1 conditions.
	convertedConditions := converted.GetConditions()
	for i := range convertedConditions {
		v1beta1conditions.Set(existing, &convertedConditions[i])
	}

	// Copy them back to the convertedCAPIMachine.
	converted.SetConditions(existing.GetConditions())
}

// EnsureCAPIV1Beta2Conditions iterates over all CAPI v1beta2 conditions and sets them on the converted object.
func EnsureCAPIV1Beta2Conditions(existing v1beta2conditions.Setter, converted v1beta2conditions.Setter) {
	// Merge the v1beta2 conditions.
	convertedConditions := converted.GetV1Beta2Conditions()
	for i := range convertedConditions {
		v1beta2conditions.Set(existing, convertedConditions[i])
	}

	// Copy them back to the convertedCAPIMachine.
	converted.SetV1Beta2Conditions(existing.GetV1Beta2Conditions())
}

// EnsureCAPIDeprecatedV1Beta1Conditions iterates over all CAPI v1beta1 conditions and sets them on the converted object.
func EnsureCAPIDeprecatedV1Beta1Conditions(existing deprecatedv1beta1conditions.Setter, converted deprecatedv1beta1conditions.Setter) {
	// Merge the v1beta1 conditions.
	convertedConditions := converted.GetV1Beta1Conditions()
	for i := range convertedConditions {
		deprecatedv1beta1conditions.Set(existing, &convertedConditions[i])
	}

	// Copy them back to the convertedCAPIMachine.
	converted.SetV1Beta1Conditions(existing.GetV1Beta1Conditions())
}

// EnsureCAPIConditions iterates over all CAPI v1beta2 conditions and sets them on the converted object.
func EnsureCAPIConditions(existing conditions.Setter, converted conditions.Setter) {
	// Merge the v1beta2 conditions.
	convertedConditions := converted.GetConditions()
	for i := range convertedConditions {
		conditions.Set(existing, convertedConditions[i])
	}

	// Copy them back to the convertedCAPIMachine.
	converted.SetConditions(existing.GetConditions())
}

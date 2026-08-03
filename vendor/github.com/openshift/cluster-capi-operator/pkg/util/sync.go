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
package util

import (
	"reflect"

	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
	machinev1applyconfigs "github.com/openshift/client-go/machine/applyconfigurations/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1applyconfig "k8s.io/client-go/applyconfigurations/meta/v1"
)

// SetLastTransitionTime determines if the last transition time should be set or updated for a given condition type.
func SetLastTransitionTime(condType mapiv1beta1.ConditionType, conditions []mapiv1beta1.Condition, conditionAc *machinev1applyconfigs.ConditionApplyConfiguration) {
	for _, condition := range conditions {
		if condition.Type == condType {
			if !hasSameState(&condition, conditionAc) {
				conditionAc.WithLastTransitionTime(metav1.Now())

				return
			}

			conditionAc.WithLastTransitionTime(condition.LastTransitionTime)

			return
		}
	}
	// Condition does not exist; set the transition time
	conditionAc.WithLastTransitionTime(metav1.Now())
}

// SetLastTransitionTimeMetaV1 sets the last transition time of a condition
// apply configuration. It retains the last transition time of the current
// condition if it exists and matches new status, reason, and message values.
// If it does not exist, it sets the last transition time to the current time.
func SetLastTransitionTimeMetaV1(now metav1.Time, currentConditions []metav1.Condition, conditionAC *metav1applyconfig.ConditionApplyConfiguration) *metav1applyconfig.ConditionApplyConfiguration {
	if !conditionHasRequiredFields(conditionAC) {
		panic("conditionAC must set type, status, reason, and message")
	}

	matchingCondition := func(condition *metav1.Condition, conditionAC *metav1applyconfig.ConditionApplyConfiguration) bool {
		return (condition.Status == *conditionAC.Status) &&
			(condition.Reason == *conditionAC.Reason) &&
			(condition.Message == *conditionAC.Message) &&
			(conditionAC.ObservedGeneration == nil || condition.ObservedGeneration == *conditionAC.ObservedGeneration) // ObservedGeneration is optional so check it's not nil
	}

	for _, condition := range currentConditions {
		if condition.Type == *conditionAC.Type {
			// Condition has not changed, retain the last transition time
			if matchingCondition(&condition, conditionAC) {
				return conditionAC.WithLastTransitionTime(condition.LastTransitionTime)
			}

			// Condition has changed, set the last transition time to the current time
			return conditionAC.WithLastTransitionTime(now)
		}
	}

	// Condition was not previously set, set the last transition time to the current time
	return conditionAC.WithLastTransitionTime(now)
}

// type, status, reason, and message are required fields for a condition apply configuration.
// If any of these are empty this is a programming error.
func conditionHasRequiredFields(conditionAC *metav1applyconfig.ConditionApplyConfiguration) bool {
	return conditionAC.Type != nil && conditionAC.Status != nil && conditionAC.Reason != nil && conditionAC.Message != nil
}

// HasSameState returns true if a condition has the same state as a condition
// apply config; state is defined by the union of following fields: Type,
// Status.
func hasSameState(i *mapiv1beta1.Condition, j *machinev1applyconfigs.ConditionApplyConfiguration) bool {
	return i.Type == *j.Type &&
		i.Status == *j.Status
}

// GetResourceVersion returns the object ResourceVersion or the zero value for it.
func GetResourceVersion(obj client.Object) string {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return "0"
	}

	return obj.GetResourceVersion()
}

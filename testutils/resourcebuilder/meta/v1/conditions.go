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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition creates a new condition builder.
func Condition() ConditionBuilder {
	return ConditionBuilder{}
}

// ConditionBuilder is used to build out a condition object.
type ConditionBuilder struct {
	conditionType   string
	conditionStatus metav1.ConditionStatus
	reason          string
	message         string
}

// Build builds a new condition based on the configuration provided.
func (c ConditionBuilder) Build() metav1.Condition {
	return metav1.Condition{
		Type:               c.conditionType,
		Status:             c.conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             c.reason,
		ObservedGeneration: 1,
		Message:            c.message,
	}
}

// WithType sets the type for the condition builder.
func (c ConditionBuilder) WithType(conditionType string) ConditionBuilder {
	c.conditionType = conditionType
	return c
}

// WithStatus sets the status for the condition builder.
func (c ConditionBuilder) WithStatus(conditionStatus metav1.ConditionStatus) ConditionBuilder {
	c.conditionStatus = conditionStatus
	return c
}

// WithReason sets the reason for the condition builder.
func (c ConditionBuilder) WithReason(reason string) ConditionBuilder {
	c.reason = reason
	return c
}

// WithMessage sets the message for the condition builder.
func (c ConditionBuilder) WithMessage(message string) ConditionBuilder {
	c.message = message
	return c
}

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

// StatusCondition creates a new status condition builder.
func StatusCondition() StatusConditionBuilder {
	return StatusConditionBuilder{}
}

// StatusConditionBuilder is used to build out a status condition object.
type StatusConditionBuilder struct {
	conditionType   string
	conditionStatus metav1.ConditionStatus
	reason          string
	message         string
}

// Build builds a new status condition based on the configuration provided.
func (c StatusConditionBuilder) Build() metav1.Condition {
	return metav1.Condition{
		Type:               c.conditionType,
		Status:             c.conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             c.reason,
		ObservedGeneration: 1,
		Message:            c.message,
	}
}

// WithType sets the type for the status condition builder.
func (c StatusConditionBuilder) WithType(conditionType string) StatusConditionBuilder {
	c.conditionType = conditionType
	return c
}

// WithStatus sets the status for the status condition builder.
func (c StatusConditionBuilder) WithStatus(conditionStatus metav1.ConditionStatus) StatusConditionBuilder {
	c.conditionStatus = conditionStatus
	return c
}

// WithReason sets the reason for the status condition builder.
func (c StatusConditionBuilder) WithReason(reason string) StatusConditionBuilder {
	c.reason = reason
	return c
}

// WithMessage sets the message for the status condition builder.
func (c StatusConditionBuilder) WithMessage(message string) StatusConditionBuilder {
	c.message = message
	return c
}

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

package testutils

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Conditions", func() {
	Context("MatchConditions", func() {
		var condAvailable, condUpgradable metav1.Condition

		BeforeEach(func() {
			condAvailable = metav1.Condition{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				Message:            "Operating as expected",
				Reason:             "OperatorAvailable",
				LastTransitionTime: metav1.Now(),
			}

			condUpgradable = metav1.Condition{
				Type:               "Upgradeable",
				Status:             metav1.ConditionTrue,
				Message:            "Operating as expected",
				Reason:             "OperatorUpgradable",
				LastTransitionTime: metav1.Now(),
			}
		})

		It("matches when the conditions match", func() {
			// Order should not matter so switch them around a few times
			availableUpgradable := []metav1.Condition{condAvailable, condUpgradable}
			upgradableAvailable := []metav1.Condition{condUpgradable, condAvailable}

			Expect(availableUpgradable).To(MatchConditions(availableUpgradable))
			Expect(availableUpgradable).To(MatchConditions(upgradableAvailable))
			Expect(upgradableAvailable).To(MatchConditions(availableUpgradable))
			Expect(upgradableAvailable).To(MatchConditions(upgradableAvailable))
		})

		It("does not match when the conditions do not match", func() {
			// Order should not matter so switch them around a few times
			condUpgradableFalse := condUpgradable.DeepCopy()
			condUpgradableFalse.Status = metav1.ConditionFalse

			actual := []metav1.Condition{condAvailable, *condUpgradableFalse}
			expected := []metav1.Condition{condUpgradable, condAvailable}

			Expect(actual).ToNot(MatchConditions(expected))
		})

		It("does not match with too few conditions", func() {
			// Order should not matter so switch them around a few times
			actual := []metav1.Condition{condAvailable}
			expected := []metav1.Condition{condUpgradable, condAvailable}

			Expect(actual).ToNot(MatchConditions(expected))
		})

		It("does not match with too many conditions", func() {
			// Order should not matter so switch them around a few times
			condUpgradableFalse := condUpgradable.DeepCopy()
			condUpgradableFalse.Status = metav1.ConditionFalse

			actual := []metav1.Condition{condAvailable, condUpgradable, *condUpgradableFalse}
			expected := []metav1.Condition{condUpgradable, condAvailable}

			Expect(actual).ToNot(MatchConditions(expected))
		})

		It("does not match with an invalid actual", func() {
			expected := []metav1.Condition{condUpgradable, condAvailable}

			ok, err := MatchConditions(expected).Match(metav1.Condition{})
			Expect(err).To(MatchError(HavePrefix("conditions did not match: ConsistOf matcher expects an array/slice/map/iter")))
			Expect(ok).To(BeFalse())
		})
	})

	Context("MatchCondition", func() {
		var cond metav1.Condition

		BeforeEach(func() {
			cond = metav1.Condition{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				Message:            "Operating as expected",
				Reason:             "OperatorAvailable",
				LastTransitionTime: metav1.Now(),
			}
		})

		It("matches when all fields match", func() {
			Expect(cond).To(MatchCondition(cond))
		})

		It("matches when the timestamps differ", func() {
			condLater := cond.DeepCopy()
			condLater.LastTransitionTime = metav1.NewTime(time.Now().Add(1 * time.Hour))
			Expect(cond).To(MatchCondition(*condLater))
		})

		It("does not match when the type is mismatched", func() {
			condUpgradable := cond.DeepCopy()
			condUpgradable.Type = "Upgradable"
			Expect(cond).ToNot(MatchCondition(*condUpgradable))
		})

		It("does not match when the status is mismatched", func() {
			condFalse := cond.DeepCopy()
			condFalse.Status = metav1.ConditionFalse
			Expect(cond).ToNot(MatchCondition(*condFalse))
		})

		It("does not match when the message is mismatched", func() {
			condEmptyMessage := cond.DeepCopy()
			condEmptyMessage.Message = ""
			Expect(cond).ToNot(MatchCondition(*condEmptyMessage))
		})

		It("does not match when the reason is mismatched", func() {
			condEmptyReason := cond.DeepCopy()
			condEmptyReason.Reason = ""
			Expect(cond).ToNot(MatchCondition(*condEmptyReason))
		})

		It("does not match with an invalid actual", func() {
			ok, err := MatchCondition(cond).Match(configv1.ClusterOperatorStatusCondition{})
			Expect(err).To(Equal(errActualTypeMismatchCondition))
			Expect(ok).To(BeFalse())
		})
	})

	Context("MatchClusterOperatorStatusConditions", func() {
		var condAvailable, condUpgradable configv1.ClusterOperatorStatusCondition

		BeforeEach(func() {
			condAvailable = configv1.ClusterOperatorStatusCondition{
				Type:               configv1.OperatorAvailable,
				Status:             configv1.ConditionTrue,
				Message:            "Operating as expected",
				Reason:             "OperatorAvailable",
				LastTransitionTime: metav1.Now(),
			}

			condUpgradable = configv1.ClusterOperatorStatusCondition{
				Type:               configv1.OperatorUpgradeable,
				Status:             configv1.ConditionTrue,
				Message:            "Operating as expected",
				Reason:             "OperatorUpgradable",
				LastTransitionTime: metav1.Now(),
			}
		})

		It("matches when the conditions match", func() {
			// Order should not matter so switch them around a few times
			availableUpgradable := []configv1.ClusterOperatorStatusCondition{condAvailable, condUpgradable}
			upgradableAvailable := []configv1.ClusterOperatorStatusCondition{condUpgradable, condAvailable}

			Expect(availableUpgradable).To(MatchClusterOperatorStatusConditions(availableUpgradable))
			Expect(availableUpgradable).To(MatchClusterOperatorStatusConditions(upgradableAvailable))
			Expect(upgradableAvailable).To(MatchClusterOperatorStatusConditions(availableUpgradable))
			Expect(upgradableAvailable).To(MatchClusterOperatorStatusConditions(upgradableAvailable))
		})

		It("does not match when the conditions do not match", func() {
			// Order should not matter so switch them around a few times
			condUpgradableFalse := condUpgradable.DeepCopy()
			condUpgradableFalse.Status = configv1.ConditionFalse

			actual := []configv1.ClusterOperatorStatusCondition{condAvailable, *condUpgradableFalse}
			expected := []configv1.ClusterOperatorStatusCondition{condUpgradable, condAvailable}

			Expect(actual).ToNot(MatchClusterOperatorStatusConditions(expected))
		})

		It("does not match with too few conditions", func() {
			// Order should not matter so switch them around a few times
			actual := []configv1.ClusterOperatorStatusCondition{condAvailable}
			expected := []configv1.ClusterOperatorStatusCondition{condUpgradable, condAvailable}

			Expect(actual).ToNot(MatchClusterOperatorStatusConditions(expected))
		})

		It("does not match with too many conditions", func() {
			// Order should not matter so switch them around a few times
			condUpgradableFalse := condUpgradable.DeepCopy()
			condUpgradableFalse.Status = configv1.ConditionFalse

			actual := []configv1.ClusterOperatorStatusCondition{condAvailable, condUpgradable, *condUpgradableFalse}
			expected := []configv1.ClusterOperatorStatusCondition{condUpgradable, condAvailable}

			Expect(actual).ToNot(MatchClusterOperatorStatusConditions(expected))
		})

		It("does not match with an invalid actual", func() {
			expected := []configv1.ClusterOperatorStatusCondition{condUpgradable, condAvailable}

			ok, err := MatchClusterOperatorStatusConditions(expected).Match(metav1.Condition{})
			Expect(err).To(MatchError(HavePrefix("conditions did not match: ConsistOf matcher expects an array/slice/map/iter")))
			Expect(ok).To(BeFalse())
		})
	})

	Context("MatchClusterOperatorCondition", func() {
		var cond configv1.ClusterOperatorStatusCondition

		BeforeEach(func() {
			cond = configv1.ClusterOperatorStatusCondition{
				Type:               configv1.OperatorAvailable,
				Status:             configv1.ConditionTrue,
				Message:            "Operating as expected",
				Reason:             "OperatorAvailable",
				LastTransitionTime: metav1.Now(),
			}
		})

		It("matches when all fields match", func() {
			Expect(cond).To(MatchClusterOperatorStatusCondition(cond))
		})

		It("matches when the timestamps differ", func() {
			condLater := cond.DeepCopy()
			condLater.LastTransitionTime = metav1.NewTime(time.Now().Add(1 * time.Hour))
			Expect(cond).To(MatchClusterOperatorStatusCondition(*condLater))
		})

		It("does not match when the type is mismatched", func() {
			condUpgradable := cond.DeepCopy()
			condUpgradable.Type = configv1.OperatorUpgradeable
			Expect(cond).ToNot(MatchClusterOperatorStatusCondition(*condUpgradable))
		})

		It("does not match when the status is mismatched", func() {
			condFalse := cond.DeepCopy()
			condFalse.Status = configv1.ConditionFalse
			Expect(cond).ToNot(MatchClusterOperatorStatusCondition(*condFalse))
		})

		It("does not match when the message is mismatched", func() {
			condEmptyMessage := cond.DeepCopy()
			condEmptyMessage.Message = ""
			Expect(cond).ToNot(MatchClusterOperatorStatusCondition(*condEmptyMessage))
		})

		It("does not match when the reason is mismatched", func() {
			condEmptyReason := cond.DeepCopy()
			condEmptyReason.Reason = ""
			Expect(cond).ToNot(MatchClusterOperatorStatusCondition(*condEmptyReason))
		})

		It("does not match with an invalid actual", func() {
			ok, err := MatchClusterOperatorStatusCondition(cond).Match(metav1.Condition{})
			Expect(err).To(Equal(errActualTypeMismatchClusterOperatorStatusCondition))
			Expect(ok).To(BeFalse())
		})
	})
})

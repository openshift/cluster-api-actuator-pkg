package operators

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("Cluster autoscaler operator should", framework.LabelOperators, framework.LabelAutoscaler, func() {
	var client runtimeclient.Client
	var ctx context.Context
	var gatherer *gatherer.StateGatherer

	BeforeEach(func() {
		var err error

		ctx = framework.GetContext()

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")

		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ok := framework.WaitForValidatingWebhook(ctx, client, "autoscaling.openshift.io")
		Expect(ok).To(BeTrue(), "Failed to wait for ValidatingWebhook")
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to GatherAll")
		}
	})

	It("reject invalid ClusterAutoscaler resources early via webhook", func() {
		invalidCA := &caov1.ClusterAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterAutoscaler",
				APIVersion: "autoscaling.openshift.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				// Only "default" is allowed.
				Name: "invalid-name",
			},
		}

		Expect(client.Create(context.TODO(), invalidCA)).ToNot(Succeed(), "Failed to create invalid ClusterAutoscaler")
	})

	It("reject invalid MachineAutoscaler resources early via webhook", func() {
		invalidMA := &caov1beta1.MachineAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "MachineAutoscaler",
				APIVersion: "autoscaling.openshift.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-%d", time.Now().Unix()),
				Namespace: framework.MachineAPINamespace,
			},
			Spec: caov1beta1.MachineAutoscalerSpec{
				// Min is greater than max, which is invalid.
				MinReplicas: 8,
				MaxReplicas: 2,
				ScaleTargetRef: caov1beta1.CrossVersionObjectReference{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
					Name:       "test",
				},
			},
		}

		Expect(client.Create(context.TODO(), invalidMA)).ToNot(Succeed(), "Failed to create invalid MachineAutoscaler")
	})
})

var _ = Describe("Cluster autoscaler operator deployment should", framework.LabelOperators, framework.LabelAutoscaler, func() {
	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ctx := framework.GetContext()
		Expect(framework.IsDeploymentAvailable(ctx, client, "cluster-autoscaler-operator", framework.MachineAPINamespace)).To(BeTrue(),
			"Failed to wait for cluster-autoscaler-operator Deployment to be available")
	})
})

var _ = Describe("Cluster autoscaler cluster operator status should", framework.LabelOperators, framework.LabelAutoscaler, func() {
	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ctx := framework.GetContext()

		Expect(framework.WaitForStatusAvailableShort(ctx, client, "cluster-autoscaler")).To(BeTrue(),
			"Failed to wait for cluster-autoscaler Cluster Operator to be available")
	})
})

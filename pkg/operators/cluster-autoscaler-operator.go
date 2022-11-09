package operators

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("[Feature:Operators] Cluster autoscaler operator should", func() {
	var client runtimeclient.Client
	var gatherer *gatherer.StateGatherer

	defer GinkgoRecover()

	BeforeEach(func() {
		var err error

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred())

		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		ok := framework.WaitForValidatingWebhook(client, "autoscaling.openshift.io")
		Expect(ok).To(BeTrue())
	})

	AfterEach(func() {
		testDescription := CurrentGinkgoTestDescription()
		if testDescription.Failed == true {
			Expect(gatherer.WithTestDescription(testDescription).GatherAll()).To(Succeed())
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

		err := client.Create(context.TODO(), invalidCA)
		Expect(err).To(HaveOccurred())
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

		err := client.Create(context.TODO(), invalidMA)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("[Feature:Operators] Cluster autoscaler operator deployment should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		var err error
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.IsDeploymentAvailable(client, "cluster-autoscaler-operator", framework.MachineAPINamespace)).To(BeTrue())
	})
})

var _ = Describe("[Feature:Operators] Cluster autoscaler cluster operator status should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		var err error
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.WaitForStatusAvailableShort(client, "cluster-autoscaler")).To(BeTrue())
	})
})

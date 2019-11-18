package operators

import (
	"context"
	"fmt"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	e2e "github.com/openshift/cluster-api-actuator-pkg/pkg/e2e/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = g.Describe("[Feature:Operators] Cluster autoscaler operator should", func() {
	var client runtimeclient.Client

	defer g.GinkgoRecover()

	g.BeforeEach(func() {
		var err error

		client, err = e2e.LoadClient()
		o.Expect(err).NotTo(o.HaveOccurred())

		ok := waitForValidatingWebhook(client, "autoscaling.openshift.io")
		o.Expect(ok).To(o.BeTrue())
	})

	g.It("reject invalid ClusterAutoscaler resources early via webhook", func() {
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
		o.Expect(err).To(o.HaveOccurred())
	})

	g.It("reject invalid MachineAutoscaler resources early via webhook", func() {
		invalidMA := &caov1beta1.MachineAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "MachineAutoscaler",
				APIVersion: "autoscaling.openshift.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-%d", time.Now().Unix()),
				Namespace: e2e.MachineAPINamespace,
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
		o.Expect(err).To(o.HaveOccurred())
	})
})

var _ = g.Describe("[Feature:Operators] Cluster autoscaler operator deployment should", func() {
	defer g.GinkgoRecover()

	g.It("be available", func() {
		var err error
		client, err := e2e.LoadClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(e2e.IsDeploymentAvailable(client, "cluster-autoscaler-operator")).To(o.BeTrue())
	})
})

var _ = g.Describe("[Feature:Operators] Cluster autoscaler cluster operator status should", func() {
	defer g.GinkgoRecover()

	g.It("be available", func() {
		var err error
		client, err := e2e.LoadClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(isStatusAvailable(client, "cluster-autoscaler")).To(o.BeTrue())
	})
})

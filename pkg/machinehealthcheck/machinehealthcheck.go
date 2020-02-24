package machinehealthcheck

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	e2e "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	mhcv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/healthchecking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:MachineHealthCheck] MachineHealthCheck", func() {
	var client client.Client

	var machineSet *mapiv1beta1.MachineSet
	var machineSetParams e2e.MachineSetParams

	const E2EConditionType = "MachineHealthCheckE2E"

	nodeCondition := corev1.NodeCondition{
		Type:               E2EConditionType,
		Status:             corev1.ConditionTrue,
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             "E2E",
		Message:            "MachineHealthCheck E2E tests",
	}

	BeforeEach(func() {
		var err error

		client, err = e2e.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		// Get the current workers MachineSets so we can copy a ProviderSpec
		// from one to use with our new dedicated MachineSet.
		workers, err := e2e.GetWorkerMachineSets(client)
		Expect(err).ToNot(HaveOccurred())

		providerSpec := workers[0].Spec.Template.Spec.ProviderSpec.DeepCopy()
		clusterName := workers[0].Spec.Template.Labels[e2e.ClusterKey]

		// TODO(bison): This should probably be appended with
		// something other than a timestamp, e.g. a random string.
		msName := fmt.Sprintf("e2e-mhc-%d", time.Now().Unix())

		machineSetParams = e2e.MachineSetParams{
			Name:         msName,
			Replicas:     3,
			ProviderSpec: providerSpec,
			Labels: map[string]string{
				"mhc.e2e.openshift.io": msName,
				e2e.ClusterKey:         clusterName,
			},
		}

		By("Creating a new MachineSet")
		machineSet, err = e2e.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred())

		e2e.WaitForMachineSet(client, machineSet.GetName())
	})

	AfterEach(func() {
		By("Deleting the new MachineSet")
		err := client.Delete(context.Background(), machineSet)
		Expect(err).ToNot(HaveOccurred())

		e2e.WaitForMachineSetDelete(client, machineSet)
	})

	It("should remediate unhealthy nodes", func() {
		By("Setting conditions on a Node")

		selector := machineSet.Spec.Selector
		machines, err := e2e.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(machines).ToNot(BeEmpty())

		machine := machines[0]
		node, err := e2e.GetNodeForMachine(client, machine)
		Expect(err).ToNot(HaveOccurred())

		err = e2e.AddNodeCondition(client, node, nodeCondition)
		Expect(err).ToNot(HaveOccurred())

		By("Creating a MachineHealthCheck resource")
		mhcParams := e2e.MachineHealthCheckParams{
			Name:   machineSetParams.Name,
			Labels: machineSetParams.Labels,
			Conditions: []mhcv1beta1.UnhealthyCondition{
				{
					Type:    E2EConditionType,
					Status:  corev1.ConditionTrue,
					Timeout: "1s",
				},
			},
		}

		mhc, err := e2e.CreateMHC(client, mhcParams)
		Expect(err).ToNot(HaveOccurred())
		Expect(mhc).ToNot(BeNil())

		By("Verifying the matching Machine is deleted")
		e2e.WaitForMachineDelete(client, machine)

		By("Verifying the MachineSet recovers")
		e2e.WaitForMachineSet(client, machineSet.GetName())

		By("Deleting the MachineHealthCheck resource")
		err = client.Delete(context.Background(), mhc)
		Expect(err).ToNot(HaveOccurred())
	})

	// It("should respect maxUnhealthy", func() {
	// 	// TODO(bison): This.
	// })
})

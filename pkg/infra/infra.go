package infra

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var nodeDrainLabels = map[string]string{
	framework.WorkerNodeRoleLabel: "",
	"node-draining-test":          string(uuid.NewUUID()),
}

func replicationControllerWorkload(namespace string) *corev1.ReplicationController {
	var replicas int32 = 20
	return &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pdb-workload",
			Namespace: namespace,
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &replicas,
			Selector: map[string]string{
				"app": "nginx",
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "work",
							Image:   "registry.ci.openshift.org/openshift/origin-v4.0:base",
							Command: []string{"sleep", "10h"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("40m"),
									"memory": resource.MustParse("50Mi"),
								},
							},
						},
					},
					NodeSelector: nodeDrainLabels,
					Tolerations: []corev1.Toleration{
						{
							Key:      "kubemark",
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}
}

func podDisruptionBudget(namespace string) *policyv1.PodDisruptionBudget {
	maxUnavailable := intstr.FromInt(1)
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pdb",
			Namespace: namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			MaxUnavailable: &maxUnavailable,
		},
	}
}

func invalidMachinesetWithEmptyProviderConfig() *machinev1.MachineSet {
	var oneReplicas int32 = 1
	return &machinev1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-machineset",
			Namespace: framework.MachineAPINamespace,
		},
		Spec: machinev1.MachineSetSpec{
			Replicas: &oneReplicas,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"little-kitty": "i-am-little-kitty",
				},
			},
			Template: machinev1.MachineTemplateSpec{
				ObjectMeta: machinev1.ObjectMeta{
					Labels: map[string]string{
						"big-kitty": "i-am-bit-kitty",
					},
				},
				Spec: machinev1.MachineSpec{
					// Empty providerSpec!!! we don't want to provision real instances.
					// Just to observe how many machine replicas get created.
					ProviderSpec: machinev1.ProviderSpec{},
				},
			},
		},
	}
}

func deleteObject(client runtimeclient.Client, obj runtimeclient.Object) error {
	cascadeDelete := metav1.DeletePropagationForeground
	return client.Delete(context.TODO(), obj, &runtimeclient.DeleteOptions{
		PropagationPolicy: &cascadeDelete,
	})
}

func deleteObjects(client runtimeclient.Client, delObjects map[string]runtimeclient.Object) error {
	// Remove resources
	for _, obj := range delObjects {
		if err := deleteObject(client, obj); err != nil {
			klog.Errorf("[cleanup] error deleting object: %v", err)
			return err
		}
	}
	return nil
}

var _ = Describe("Managed cluster should", framework.LabelMachines, func() {
	var client runtimeclient.Client
	var machineSet *machinev1.MachineSet
	var machineSetParams framework.MachineSetParams

	var gatherer *gatherer.StateGatherer

	BeforeEach(func() {
		var err error

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred())

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred())

	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() == true {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
		}

		if machineSet != nil {
			By("Deleting the new MachineSet")
			err := client.Delete(context.Background(), machineSet)
			Expect(err).ToNot(HaveOccurred())
			framework.WaitForMachineSetDelete(client, machineSet)
		}

	})

	When("machineset has one replica", func() {
		BeforeEach(func() {
			var err error
			machineSetParams = framework.BuildMachineSetParams(client, 1)

			By("Creating a new MachineSet")
			machineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())

			framework.WaitForMachineSet(client, machineSet.GetName())
		})

		// Machines required for test: 1
		// Reason: This test works on a single machine and its node.
		It("have ability to additively reconcile taints from machine to nodes", func() {
			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(client, &selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(machines).ToNot(BeEmpty())

			machine := machines[0]
			By(fmt.Sprintf("getting machine %q", machine.Name))

			node, err := framework.GetNodeForMachine(client, machine)
			Expect(err).NotTo(HaveOccurred())
			By(fmt.Sprintf("getting the backed node %q", node.Name))

			nodeTaint := corev1.Taint{
				Key:    "not-from-machine",
				Value:  "true",
				Effect: corev1.TaintEffectNoSchedule,
			}
			By(fmt.Sprintf("updating node %q with taint: %v", node.Name, nodeTaint))
			for {
				node.Spec.Taints = append(node.Spec.Taints, nodeTaint)
				err = client.Update(context.TODO(), node)
				if !apierrors.IsConflict(err) {
					break
				}
			}
			Expect(err).NotTo(HaveOccurred())

			machineTaint := corev1.Taint{
				Key:    fmt.Sprintf("from-machine-%v", string(uuid.NewUUID())),
				Value:  "true",
				Effect: corev1.TaintEffectNoSchedule,
			}
			By(fmt.Sprintf("updating machine %q with taint: %v", machine.Name, machineTaint))
			for {
				machine.Spec.Taints = append(machine.Spec.Taints, machineTaint)
				err = client.Update(context.TODO(), machine)
				if !apierrors.IsConflict(err) {
					break
				}
			}
			Expect(err).NotTo(HaveOccurred())

			var expectedTaints = sets.NewString("not-from-machine", machineTaint.Key)
			Eventually(func() bool {
				klog.Info("Getting node from machine again for verification of taints")
				node, err := framework.GetNodeForMachine(client, machine)
				if err != nil {
					return false
				}
				var observedTaints = sets.NewString()
				for _, taint := range node.Spec.Taints {
					observedTaints.Insert(taint.Key)
				}
				if expectedTaints.Difference(observedTaints).HasAny("not-from-machine", machineTaint.Key) == false {
					klog.Infof("Expected : %v, observed %v , difference %v, ", expectedTaints, observedTaints, expectedTaints.Difference(observedTaints))
					return true
				}
				klog.Infof("Did not find all expected taints on the node. Missing: %v", expectedTaints.Difference(observedTaints))
				return false
			}, framework.WaitMedium, 5*time.Second).Should(BeTrue())
		})

	})

	When("machineset has 2 replicas", func() {
		BeforeEach(func() {
			var err error
			machineSetParams = framework.BuildMachineSetParams(client, 2)

			By("Creating a new MachineSet")
			machineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())

			framework.WaitForMachineSet(client, machineSet.GetName())
		})

		// Machines required for test: 2
		// Reason: We want to test that all machines get replaced when we delete them.
		It("recover from deleted worker machines", func() {
			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(client, &selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(machines).ToNot(BeEmpty())

			By(fmt.Sprint("deleting all machines"))
			err = framework.DeleteMachines(client, machines...)
			Expect(err).NotTo(HaveOccurred())
			framework.WaitForMachinesDeleted(client, machines...)

			framework.WaitForMachineSet(client, machineSet.GetName())
		})

		// Machines required for test: 4
		// Reason: MachineSet scales 2->0 and MachineSet2 scales 0->2. Changing to scaling 1->0 and 0->1 might not test this thoroughly.
		It("grow and decrease when scaling different machineSets simultaneously", framework.LabelPeriodic, func() {
			By("Creating a second MachineSet") // Machineset 1 can start with 1 replica
			machineSetParams := framework.BuildMachineSetParams(client, 0)
			machineSet2, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())

			// Make sure second machineset gets deleted anyway
			defer func() {
				By("Deleting the second MachineSet")
				err := deleteObject(client, machineSet2)
				Expect(err).ToNot(HaveOccurred())
				framework.WaitForMachineSetDelete(client, machineSet2)
			}()

			framework.WaitForMachineSet(client, machineSet2.GetName())

			Expect(framework.ScaleMachineSet(machineSet.GetName(), 0)).To(Succeed())
			Expect(framework.ScaleMachineSet(machineSet2.GetName(), 1)).To(Succeed())

			framework.WaitForMachineSet(client, machineSet.GetName())
			framework.WaitForMachineSet(client, machineSet2.GetName())
		})

		// Machines required for test: 2 (3 but it gets deleted without waiting for it to be ready)
		// Reason: Pods are spread across both machines. After one is deleted, the pods are rescheduled onto the other machine.
		It("drain node before removing machine resource", func() {
			By("Create a machine for node about to be drained")

			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(client, &selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(machines)).To(BeNumerically(">=", 2))

			// Add node draining labels to params
			for k, v := range nodeDrainLabels {
				machineSetParams.Labels[k] = v
			}

			machines[0].Spec.ObjectMeta.Labels = machineSetParams.Labels
			machines[1].Spec.ObjectMeta.Labels = machineSetParams.Labels

			err = client.Update(context.TODO(), machines[0])
			Expect(err).ToNot(HaveOccurred())

			err = client.Update(context.TODO(), machines[1])
			Expect(err).ToNot(HaveOccurred())

			// Make sure RC and PDB get deleted anyway
			delObjects := make(map[string]runtimeclient.Object)

			defer func() {
				err := deleteObjects(client, delObjects)
				Expect(err).ToNot(HaveOccurred())
			}()

			By("Creating RC with workload")

			// Use the openshift-machine-api namespace as it is excluded from
			// Pod security admission checks.
			namespace := framework.MachineAPINamespace

			rc := replicationControllerWorkload(namespace)
			err = client.Create(context.TODO(), rc)
			Expect(err).NotTo(HaveOccurred())
			delObjects["rc"] = rc

			By("Creating PDB for RC")
			pdb := podDisruptionBudget(namespace)
			err = client.Create(context.TODO(), pdb)
			Expect(err).NotTo(HaveOccurred())
			delObjects["pdb"] = pdb

			By("Wait until all replicas are ready")
			err = framework.WaitUntilAllRCPodsAreReady(client, rc)
			Expect(err).NotTo(HaveOccurred())

			// TODO(jchaloup): delete machine that has at least half of the RC pods

			// All pods are distributed evenly among all nodes so it's fine to drain
			// random node and observe reconciliation of pods on the other one.
			By("Delete machine to trigger node draining")
			err = client.Delete(context.TODO(), machines[0])
			Expect(err).NotTo(HaveOccurred())

			// We still should be able to list the machine as until rc.replicas-1 are running on the other node
			By("Observing and verifying node draining")
			drainedNodeName, err := framework.VerifyNodeDraining(client, machines[0], rc)
			Expect(err).NotTo(HaveOccurred())

			By("Validating the machine is deleted")
			framework.WaitForMachinesDeleted(client, machines[0])

			By("Validate underlying node corresponding to machine1 is removed as well")
			err = framework.WaitUntilNodeDoesNotExists(client, drainedNodeName)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// Machines required for test: 0
	// Reason: The machineSet creation is rejected by the webhook.
	It("reject invalid machinesets", func() {
		By("Creating invalid machineset")
		invalidMachineSet := invalidMachinesetWithEmptyProviderConfig()
		expectedAdmissionWebhookErr := "admission webhook \"default.machineset.machine.openshift.io\" denied the request: providerSpec.value: Required value: a value must be provided"

		err := client.Create(context.TODO(), invalidMachineSet)
		Expect(err).To(MatchError(expectedAdmissionWebhookErr))
	})
})

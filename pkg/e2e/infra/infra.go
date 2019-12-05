package infra

import (
	"context"
	"fmt"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"

	"github.com/golang/glog"
	e2e "github.com/openshift/cluster-api-actuator-pkg/pkg/e2e/framework"
	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kpolicyapi "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var nodeDrainLabels = map[string]string{
	e2e.WorkerNodeRoleLabel: "",
	"node-draining-test":    string(uuid.NewUUID()),
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
							Image:   "busybox",
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

func podDisruptionBudget(namespace string) *kpolicyapi.PodDisruptionBudget {
	maxUnavailable := intstr.FromInt(1)
	return &kpolicyapi.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pdb",
			Namespace: namespace,
		},
		Spec: kpolicyapi.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			MaxUnavailable: &maxUnavailable,
		},
	}
}

func invalidMachinesetWithEmptyProviderConfig() *mapiv1beta1.MachineSet {
	var oneReplicas int32 = 1
	return &mapiv1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-machineset",
			Namespace: e2e.MachineAPINamespace,
		},
		Spec: mapiv1beta1.MachineSetSpec{
			Replicas: &oneReplicas,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"little-kitty": "i-am-little-kitty",
				},
			},
			Template: mapiv1beta1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"big-kitty": "i-am-bit-kitty",
					},
				},
				Spec: mapiv1beta1.MachineSpec{
					// Empty providerSpec!!! we don't want to provision real instances.
					// Just to observe how many machine replicas get created.
					ProviderSpec: mapiv1beta1.ProviderSpec{},
				},
			},
		},
	}
}

func buildMachineSetParams(client runtimeclient.Client, replicas int) e2e.MachineSetParams {
	// Get the current workers MachineSets so we can copy a ProviderSpec
	// from one to use with our new dedicated MachineSet.
	workers, err := e2e.GetWorkerMachineSets(client)
	o.Expect(err).ToNot(o.HaveOccurred())

	providerSpec := workers[0].Spec.Template.Spec.ProviderSpec.DeepCopy()
	clusterName := workers[0].Spec.Template.Labels[e2e.ClusterKey]

	clusterInfra, err := e2e.GetInfrastructure(client)
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(clusterInfra.Status.InfrastructureName).ShouldNot(o.BeEmpty())

	msName := e2e.RandomString(clusterInfra.Status.InfrastructureName)

	return e2e.MachineSetParams{
		Name:         msName,
		Replicas:     int32(replicas),
		ProviderSpec: providerSpec,
		Labels: map[string]string{
			"mhc.e2e.openshift.io": msName,
			e2e.ClusterKey:         clusterName,
		},
	}
}

func deleteObject(client runtimeclient.Client, obj runtime.Object) error {
	cascadeDelete := metav1.DeletePropagationForeground
	return client.Delete(context.TODO(), obj, &runtimeclient.DeleteOptions{
		PropagationPolicy: &cascadeDelete,
	})
}

func deleteObjects(client runtimeclient.Client, delObjects map[string]runtime.Object) error {
	// Remove resources
	for _, obj := range delObjects {
		if err := deleteObject(client, obj); err != nil {
			glog.Errorf("[cleanup] error deleting object: %v", err)
			return err
		}
	}
	return nil
}

var _ = g.Describe("[Feature:Machines] Managed cluster should", func() {
	defer g.GinkgoRecover()

	var client runtimeclient.Client
	var machineSet *mapiv1beta1.MachineSet
	var machineSetParams e2e.MachineSetParams

	g.BeforeEach(func() {
		var err error

		client, err = e2e.LoadClient()
		o.Expect(err).ToNot(o.HaveOccurred())

		machineSetParams = buildMachineSetParams(client, 3)

		g.By("Creating a new MachineSet")
		machineSet, err = e2e.CreateMachineSet(client, machineSetParams)
		o.Expect(err).ToNot(o.HaveOccurred())

		e2e.WaitForMachineSet(client, machineSet.GetName())
	})

	g.AfterEach(func() {
		g.By("Deleting the new MachineSet")
		err := client.Delete(context.Background(), machineSet)
		o.Expect(err).ToNot(o.HaveOccurred())

		e2e.WaitForMachineSetDelete(client, machineSet)
	})

	g.It("have ability to additively reconcile taints from machine to nodes", func() {
		selector := machineSet.Spec.Selector
		machines, err := e2e.GetMachines(client, &selector)
		o.Expect(err).ToNot(o.HaveOccurred())

		machine := machines[0]
		g.By(fmt.Sprintf("getting machine %q", machine.Name))

		node, err := getNodeFromMachine(client, machine)
		o.Expect(err).NotTo(o.HaveOccurred())
		g.By(fmt.Sprintf("getting the backed node %q", node.Name))

		nodeTaint := corev1.Taint{
			Key:    "not-from-machine",
			Value:  "true",
			Effect: corev1.TaintEffectNoSchedule,
		}
		g.By(fmt.Sprintf("updating node %q with taint: %v", node.Name, nodeTaint))
		node.Spec.Taints = append(node.Spec.Taints, nodeTaint)
		err = client.Update(context.TODO(), node)
		o.Expect(err).NotTo(o.HaveOccurred())

		machineTaint := corev1.Taint{
			Key:    fmt.Sprintf("from-machine-%v", string(uuid.NewUUID())),
			Value:  "true",
			Effect: corev1.TaintEffectNoSchedule,
		}
		g.By(fmt.Sprintf("updating machine %q with taint: %v", machine.Name, machineTaint))
		machine.Spec.Taints = append(machine.Spec.Taints, machineTaint)
		err = client.Update(context.TODO(), machine)
		o.Expect(err).NotTo(o.HaveOccurred())

		var expectedTaints = sets.NewString("not-from-machine", machineTaint.Key)
		o.Eventually(func() bool {
			glog.Info("Getting node from machine again for verification of taints")
			node, err := getNodeFromMachine(client, machine)
			if err != nil {
				return false
			}
			var observedTaints = sets.NewString()
			for _, taint := range node.Spec.Taints {
				observedTaints.Insert(taint.Key)
			}
			if expectedTaints.Difference(observedTaints).HasAny("not-from-machine", machineTaint.Key) == false {
				glog.Infof("Expected : %v, observed %v , difference %v, ", expectedTaints, observedTaints, expectedTaints.Difference(observedTaints))
				return true
			}
			glog.Infof("Did not find all expected taints on the node. Missing: %v", expectedTaints.Difference(observedTaints))
			return false
		}, e2e.WaitMedium, 5*time.Second).Should(o.BeTrue())
	})

	g.It("recover from deleted worker machines", func() {
		selector := machineSet.Spec.Selector
		machines, err := e2e.GetMachines(client, &selector)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(machines).ToNot(o.BeEmpty())

		machine := machines[0]

		g.By(fmt.Sprintf("deleting machine object %q", machine.Name))
		err = deleteMachine(client, machine)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.WaitForMachineDelete(client, machine)

		e2e.WaitForMachineSet(client, machineSet.GetName())
	})

	g.It("grow and decrease when scaling different machineSets simultaneously", func() {
		g.By("Creating a second MachineSet")
		machineSetParams := buildMachineSetParams(client, 0)
		machineSet2, err := e2e.CreateMachineSet(client, machineSetParams)
		o.Expect(err).ToNot(o.HaveOccurred())

		// Make sure second machineset gets deleted anyway
		defer func() {
			g.By("Deleting the second MachineSet")
			err := deleteObject(client, machineSet2)
			o.Expect(err).ToNot(o.HaveOccurred())
			e2e.WaitForMachineSetDelete(client, machineSet2)
		}()

		e2e.WaitForMachineSet(client, machineSet2.GetName())
		scaleMachineSet(machineSet.GetName(), 0)
		scaleMachineSet(machineSet2.GetName(), 3)
		e2e.WaitForMachineSet(client, machineSet.GetName())
		e2e.WaitForMachineSet(client, machineSet2.GetName())
	})

	g.It("drain node before removing machine resource", func() {
		g.By("Create a machine for node about to be drained")

		selector := machineSet.Spec.Selector
		machines, err := e2e.GetMachines(client, &selector)
		o.Expect(err).ToNot(o.HaveOccurred())

		// Add node draining labels to params
		for k, v := range nodeDrainLabels {
			machineSetParams.Labels[k] = v
		}

		machines[0].Spec.ObjectMeta.Labels = machineSetParams.Labels
		machines[1].Spec.ObjectMeta.Labels = machineSetParams.Labels

		err = client.Update(context.TODO(), machines[0])
		o.Expect(err).ToNot(o.HaveOccurred())

		err = client.Update(context.TODO(), machines[1])
		o.Expect(err).ToNot(o.HaveOccurred())

		// Make sure RC and PDB get deleted anyway
		delObjects := make(map[string]runtime.Object)

		defer func() {
			err := deleteObjects(client, delObjects)
			o.Expect(err).ToNot(o.HaveOccurred())
		}()

		g.By("Creating RC with workload")
		rc := replicationControllerWorkload("default")
		err = client.Create(context.TODO(), rc)
		o.Expect(err).NotTo(o.HaveOccurred())
		delObjects["rc"] = rc

		g.By("Creating PDB for RC")
		pdb := podDisruptionBudget("default")
		err = client.Create(context.TODO(), pdb)
		o.Expect(err).NotTo(o.HaveOccurred())
		delObjects["pdb"] = pdb

		g.By("Wait until all replicas are ready")
		err = waitUntilAllRCPodsAreReady(client, rc)
		o.Expect(err).NotTo(o.HaveOccurred())

		// TODO(jchaloup): delete machine that has at least half of the RC pods

		// All pods are distributed evenly among all nodes so it's fine to drain
		// random node and observe reconciliation of pods on the other one.
		g.By("Delete machine to trigger node draining")
		err = client.Delete(context.TODO(), machines[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		// We still should be able to list the machine as until rc.replicas-1 are running on the other node
		g.By("Observing and verifying node draining")
		drainedNodeName, err := verifyNodeDraining(client, machines[0], rc)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Validating the machine is deleted")
		e2e.WaitForMachineDelete(client, machines[0])

		g.By("Validate underlying node corresponding to machine1 is removed as well")
		err = waitUntilNodeDoesNotExists(client, drainedNodeName)
		o.Expect(err).NotTo(o.HaveOccurred())
	})

	g.It("reject invalid machinesets", func() {
		var err error
		g.By("Creating invalid machineset")
		invalidMachineSet := invalidMachinesetWithEmptyProviderConfig()

		err = client.Create(context.TODO(), invalidMachineSet)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Waiting for ReconcileError MachineSet event")
		err = wait.PollImmediate(e2e.RetryMedium, e2e.WaitShort, func() (bool, error) {
			eventList := corev1.EventList{}
			if err := client.List(context.TODO(), &eventList); err != nil {
				glog.Errorf("error querying api for eventList object: %v, retrying...", err)
				return false, nil
			}

			glog.Infof("Fetching ReconcileError MachineSet invalid-machineset event")
			for _, event := range eventList.Items {
				if event.Reason != "ReconcileError" || event.InvolvedObject.Kind != "MachineSet" || event.InvolvedObject.Name != invalidMachineSet.Name {
					continue
				}

				glog.Infof("Found ReconcileError event for %q machine set with the following message: %v", event.InvolvedObject.Name, event.Message)
				return true, nil
			}

			return false, nil
		})
		o.Expect(err).NotTo(o.HaveOccurred())

		// Verify the number of machines does not grow over time.
		// The assumption is once the ReconcileError event is recorded and caught,
		// the machineset is not reconciled again until it's updated.
		machineList := &mapiv1beta1.MachineList{}
		err = client.List(context.TODO(), machineList, runtimeclient.MatchingLabels(invalidMachineSet.Spec.Template.Labels))
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By(fmt.Sprintf("Verify no machine from %q machineset were created", invalidMachineSet.Name))
		glog.Infof("Have %v machines generated from %q machineset", len(machineList.Items), invalidMachineSet.Name)
		o.Expect(len(machineList.Items)).To(o.BeNumerically("==", 0))

		g.By("Deleting invalid machineset")
		err = client.Delete(context.TODO(), invalidMachineSet)
		o.Expect(err).NotTo(o.HaveOccurred())
	})
})

package framework

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CAPIMachineSetParams struct {
	msName            string
	clusterName       string
	failureDomain     string
	replicas          int32
	infrastructureRef corev1.ObjectReference
}

// NewCAPIMachineSetParams returns a new CAPIMachineSetParams object.
func NewCAPIMachineSetParams(msName, clusterName, failureDomain string, replicas int32, infrastructureRef corev1.ObjectReference) CAPIMachineSetParams {
	Expect(msName).ToNot(BeEmpty(), "expected the capi msName to not be empty")
	Expect(clusterName).ToNot(BeEmpty(), "expected the capi clusterName to not be empty")
	Expect(infrastructureRef.APIVersion).ToNot(BeEmpty(), "expected the infrastructureRef APIVersion to not be empty")
	Expect(infrastructureRef.Kind).ToNot(BeEmpty(), "expected the infrastructureRef Kind to not be empty")
	Expect(infrastructureRef.Name).ToNot(BeEmpty(), "expected the infrastructureRef Name to not be empty")

	return CAPIMachineSetParams{
		msName:            msName,
		clusterName:       clusterName,
		replicas:          replicas,
		infrastructureRef: infrastructureRef,
		failureDomain:     failureDomain,
	}
}

// UpdateCAPIMachineSetName returns CAPIMachineSetParams object with the updated machineset name.
func UpdateCAPIMachineSetName(msName string, params CAPIMachineSetParams) CAPIMachineSetParams {
	Expect(msName).ToNot(BeEmpty(), "expected the capi msName to not be empty")

	return CAPIMachineSetParams{
		msName:            msName,
		clusterName:       params.clusterName,
		replicas:          params.replicas,
		infrastructureRef: params.infrastructureRef,
		failureDomain:     params.failureDomain,
	}
}

// CreateCAPIMachineSet creates a new MachineSet resource.
func CreateCAPIMachineSet(ctx context.Context, cl client.Client, params CAPIMachineSetParams) (*clusterv1.MachineSet, error) {
	By(fmt.Sprintf("Creating MachineSet %q", params.msName))
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{"cluster.x-k8s.io/cluster-name": params.clusterName, "cluster.x-k8s.io/set-name": params.msName},
	}
	userDataSecret := "worker-user-data"
	template := clusterv1.MachineTemplateSpec{
		ObjectMeta: clusterv1.ObjectMeta{
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name":  params.clusterName,
				"cluster.x-k8s.io/set-name":      params.msName,
				"node-role.kubernetes.io/worker": "",
			},
		},
		Spec: clusterv1.MachineSpec{
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: &userDataSecret,
			},
			ClusterName:       params.clusterName,
			InfrastructureRef: params.infrastructureRef,
		},
	}
	ms := capiv1resourcebuilder.MachineSet().WithName(params.msName).WithNamespace(ClusterAPINamespace).WithReplicas(params.replicas).WithClusterName(params.clusterName).WithSelector(selector).WithTemplate(template).WithLabels(map[string]string{"cluster.x-k8s.io/cluster-name": params.clusterName}).Build()

	if params.failureDomain != "" {
		ms.Spec.Template.Spec.FailureDomain = &params.failureDomain
	}

	Eventually(func() error {
		return cl.Create(ctx, ms)
	}, WaitLong, RetryShort).Should(Succeed(), "it should have been able to create a new CAPI MachineSet")

	return ms, nil
}

// WaitForCAPIMachineSetsDeleted polls until the given MachineSets are not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForCAPIMachineSetsDeleted(ctx context.Context, cl client.Client, machineSets ...*clusterv1.MachineSet) {
	for _, ms := range machineSets {
		By(fmt.Sprintf("Waiting for MachineSet %q to be deleted", ms.GetName()))
		Eventually(func() bool {
			selector := ms.Spec.Selector

			machines, err := GetCAPIMachines(ctx, cl, &selector)
			if err != nil || len(machines) != 0 {
				return false // Still have Machines, or other error.
			}

			err = cl.Get(ctx, client.ObjectKey{
				Name:      ms.GetName(),
				Namespace: ms.GetNamespace(),
			}, &clusterv1.MachineSet{})

			return apierrors.IsNotFound(err) // MachineSet and Machines were deleted.
		}, WaitLong, RetryMedium).Should(BeTrue(), "it should have been able to delete all the CAPI MachineSets")
	}
}

// DeleteCAPIMachineSets deletes the specified machinesets and returns an error on failure.
func DeleteCAPIMachineSets(ctx context.Context, cl client.Client, machineSets ...*clusterv1.MachineSet) {
	for _, ms := range machineSets {
		By(fmt.Sprintf("Deleting MachineSet %q", ms.GetName()))
		Eventually(func() error {
			return cl.Delete(ctx, ms)
		}, WaitLong, RetryShort).Should(Succeed(), "the CAPI MachineSets should have been deleted")
	}
}

// WaitForCAPIMachinesRunning waits for the all Machines belonging to the named
// MachineSet to enter the "Running" phase, and for all nodes belonging to those
// Machines to be ready.
func WaitForCAPIMachinesRunning(ctx context.Context, cl client.Client, name string) {
	By(fmt.Sprintf("Waiting for MachineSet machines %q to enter Running phase", name))

	machineSet, err := GetCAPIMachineSet(ctx, cl, name)
	Expect(err).ToNot(HaveOccurred(), "Failed to get capi machineset")

	Eventually(func() error {
		machines, err := GetCAPIMachinesFromMachineSet(ctx, cl, machineSet)
		if err != nil {
			return err
		}

		replicas := ptr.Deref(machineSet.Spec.Replicas, 0)

		if len(machines) != int(replicas) {
			return fmt.Errorf("%q: found %d Machines, but MachineSet has %d replicas",
				name, len(machines), int(replicas))
		}

		running := FilterCAPIMachinesInPhase(machines, "Running")

		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			return fmt.Errorf("%q: not all Machines are running: %d of %d",
				name, len(running), len(machines))
		}

		for _, m := range running {
			node, err := GetCAPINodeForMachine(ctx, cl, m)
			if err != nil {
				return err
			}

			if !IsNodeReady(node) {
				return fmt.Errorf("%s: node is not ready", node.Name)
			}
		}

		return nil
	}, WaitOverLong, RetryMedium).Should(Succeed(), "all machines belonging to the MachineSet should be in Running phase")
}

// GetCAPIMachineSet gets a machineset by its name from the default machine API namespace.
func GetCAPIMachineSet(ctx context.Context, cl client.Client, name string) (*clusterv1.MachineSet, error) {
	machineSet := &clusterv1.MachineSet{}
	key := client.ObjectKey{Namespace: ClusterAPINamespace, Name: name}

	Eventually(func() error {
		return cl.Get(ctx, key, machineSet)
	}, WaitShort, RetryShort).Should(Succeed(), "it should be able to get a machineset by its name")

	return machineSet, nil
}

// GetCAPIMachinesFromMachineSet returns an array of machines owned by a given machineSet.
func GetCAPIMachinesFromMachineSet(ctx context.Context, cl client.Client, machineSet *clusterv1.MachineSet) ([]*clusterv1.Machine, error) {
	machines, err := GetCAPIMachines(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("error getting machines: %w", err)
	}

	var machinesForSet []*clusterv1.Machine

	for key := range machines {
		if metav1.IsControlledBy(machines[key], machineSet) {
			machinesForSet = append(machinesForSet, machines[key])
		}
	}

	return machinesForSet, nil
}

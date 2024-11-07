package framework

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type machineSetParams struct {
	msName            string
	clusterName       string
	failureDomain     string
	replicas          int32
	infrastructureRef corev1.ObjectReference
}

// NewMachineSetParams returns a new machineSetParams object.
func NewMachineSetParams(msName, clusterName, failureDomain string, replicas int32, infrastructureRef corev1.ObjectReference) machineSetParams {
	Expect(msName).ToNot(BeEmpty())
	Expect(clusterName).ToNot(BeEmpty())
	Expect(infrastructureRef.APIVersion).ToNot(BeEmpty())
	Expect(infrastructureRef.Kind).ToNot(BeEmpty())
	Expect(infrastructureRef.Name).ToNot(BeEmpty())

	return machineSetParams{
		msName:            msName,
		clusterName:       clusterName,
		replicas:          replicas,
		infrastructureRef: infrastructureRef,
		failureDomain:     failureDomain,
	}
}

// CreateMachineSet creates a new MachineSet resource.
func CreateMachineSet(cl client.Client, params machineSetParams) *clusterv1.MachineSet {
	By(fmt.Sprintf("Creating MachineSet %q", params.msName))

	userDataSecret := "worker-user-data"
	ms := &clusterv1.MachineSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineSet",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.msName,
			Namespace: CAPINamespace,
		},
		Spec: clusterv1.MachineSetSpec{
			Replicas:    &params.replicas,
			ClusterName: params.clusterName,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"machine.openshift.io/cluster-api-cluster":    params.clusterName,
					"machine.openshift.io/cluster-api-machineset": params.msName,
				},
			},
			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{
						"machine.openshift.io/cluster-api-cluster":    params.clusterName,
						"machine.openshift.io/cluster-api-machineset": params.msName,
					},
				},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{
						DataSecretName: &userDataSecret,
					},
					ClusterName:       params.clusterName,
					InfrastructureRef: params.infrastructureRef,
				},
			},
		},
	}

	if params.failureDomain != "" {
		ms.Spec.Template.Spec.FailureDomain = &params.failureDomain
	}

	Expect(cl.Create(ctx, ms)).To(Succeed())
	return ms
}

// WaitForMachineSetsDeleted polls until the given MachineSets are not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForMachineSetsDeleted(cl client.Client, machineSets ...*clusterv1.MachineSet) {
	for _, ms := range machineSets {
		By(fmt.Sprintf("Waiting for MachineSet %q to be deleted", ms.GetName()))
		Eventually(func() bool {
			selector := ms.Spec.Selector

			machines, err := GetMachines(cl, &selector)
			if err != nil || len(machines) != 0 {
				return false // Still have Machines, or other error.
			}

			err = cl.Get(ctx, client.ObjectKey{
				Name:      ms.GetName(),
				Namespace: ms.GetNamespace(),
			}, &clusterv1.MachineSet{})

			return apierrors.IsNotFound(err) // MachineSet and Machines were deleted.
		}, WaitLong, RetryMedium).Should(BeTrue())
	}
}

// DeleteMachineSets deletes the specified machinesets and returns an error on failure.
func DeleteMachineSets(cl client.Client, machineSets ...*clusterv1.MachineSet) {
	for _, ms := range machineSets {
		By(fmt.Sprintf("Deleting MachineSet %q", ms.GetName()))
		Expect(cl.Delete(ctx, ms)).To(Succeed())
	}
}

// WaitForMachineSet waits for the all Machines belonging to the named
// MachineSet to enter the "Running" phase, and for all nodes belonging to those
// Machines to be ready.
func WaitForMachineSet(cl client.Client, name string) {
	By(fmt.Sprintf("Waiting for MachineSet machines %q to enter Running phase", name))

	machineSet, err := GetMachineSet(cl, name)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		machines, err := GetMachinesFromMachineSet(cl, machineSet)
		if err != nil {
			return err
		}

		replicas := pointer.Int32PtrDerefOr(machineSet.Spec.Replicas, 0)

		if len(machines) != int(replicas) {
			return fmt.Errorf("%q: found %d Machines, but MachineSet has %d replicas",
				name, len(machines), int(replicas))
		}

		running := FilterRunningMachines(machines)

		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			return fmt.Errorf("%q: not all Machines are running: %d of %d",
				name, len(running), len(machines))
		}

		for _, m := range running {
			node, err := GetNodeForMachine(cl, m)
			if err != nil {
				return err
			}

			if !isNodeReady(node) {
				return fmt.Errorf("%s: node is not ready", node.Name)
			}
		}

		return nil
	}, WaitOverLong, RetryMedium).Should(Succeed())
}

// GetMachineSet gets a machineset by its name from the default machine API namespace.
func GetMachineSet(cl client.Client, name string) (*clusterv1.MachineSet, error) {
	machineSet := &clusterv1.MachineSet{}
	key := client.ObjectKey{Namespace: CAPINamespace, Name: name}

	Expect(cl.Get(ctx, key, machineSet)).To(Succeed())

	return machineSet, nil
}

// GetMachinesFromMachineSet returns an array of machines owned by a given machineSet
func GetMachinesFromMachineSet(cl client.Client, machineSet *clusterv1.MachineSet) ([]*clusterv1.Machine, error) {
	machines, err := GetMachines(cl)
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

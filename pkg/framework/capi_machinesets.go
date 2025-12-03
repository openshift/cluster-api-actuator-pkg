package framework

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
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
func CreateCAPIMachineSet(ctx context.Context, cl client.Client, params CAPIMachineSetParams) (*clusterv1beta1.MachineSet, error) {
	By(fmt.Sprintf("Creating MachineSet %q", params.msName))
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{"cluster.x-k8s.io/cluster-name": params.clusterName, "cluster.x-k8s.io/set-name": params.msName},
	}
	userDataSecret := "worker-user-data"
	template := clusterv1beta1.MachineTemplateSpec{
		ObjectMeta: clusterv1beta1.ObjectMeta{
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name":  params.clusterName,
				"cluster.x-k8s.io/set-name":      params.msName,
				"node-role.kubernetes.io/worker": "",
			},
		},
		Spec: clusterv1beta1.MachineSpec{
			Bootstrap: clusterv1beta1.Bootstrap{
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
func WaitForCAPIMachineSetsDeleted(ctx context.Context, cl client.Client, machineSets ...*clusterv1beta1.MachineSet) {
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
			}, &clusterv1beta1.MachineSet{})

			return apierrors.IsNotFound(err) // MachineSet and Machines were deleted.
		}, WaitLong, RetryMedium).Should(BeTrue(), "it should have been able to delete all the CAPI MachineSets")
	}
}

// DeleteCAPIMachineSets deletes the specified machinesets and returns an error on failure.
func DeleteCAPIMachineSets(ctx context.Context, cl client.Client, machineSets ...*clusterv1beta1.MachineSet) {
	for _, ms := range machineSets {
		By(fmt.Sprintf("Deleting MachineSet %q", ms.GetName()))
		Eventually(func() error {
			if err := cl.Delete(ctx, ms); err != nil && !apierrors.IsNotFound(err) {
				return err
			}

			return nil
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
func GetCAPIMachineSet(ctx context.Context, cl client.Client, name string) (*clusterv1beta1.MachineSet, error) {
	machineSet := &clusterv1beta1.MachineSet{}
	key := client.ObjectKey{Namespace: ClusterAPINamespace, Name: name}

	Eventually(func() error {
		return cl.Get(ctx, key, machineSet)
	}, WaitShort, RetryShort).Should(Succeed(), "it should be able to get a machineset by its name")

	return machineSet, nil
}

// GetCAPIMachinesFromMachineSet returns an array of machines owned by a given machineSet.
func GetCAPIMachinesFromMachineSet(ctx context.Context, cl client.Client, machineSet *clusterv1beta1.MachineSet) ([]*clusterv1beta1.Machine, error) {
	machines, err := GetCAPIMachines(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("error getting machines: %w", err)
	}

	var machinesForSet []*clusterv1beta1.Machine

	for key := range machines {
		if metav1.IsControlledBy(machines[key], machineSet) {
			machinesForSet = append(machinesForSet, machines[key])
		}
	}

	return machinesForSet, nil
}

// WaitForCAPIMachinesRunningWithRetry waits for all Machines belonging to the machineSet to be running and their nodes to be ready.
// Unlike WaitForCAPIMachinesRunning, this function does not fail the test when machines cannot be provisioned due to insufficient capacity.
// It returns an error only when machines fail due to insufficient cloud provider capacity, allowing the caller to retry with different configurations.
func WaitForCAPIMachinesRunningWithRetry(ctx context.Context, cl client.Client, name string, capacityErrorKeys []string) error {
	machineSet, err := GetCAPIMachineSet(ctx, cl, name)
	Expect(err).ToNot(HaveOccurred(), "Failed to get CAPI machineset %s", name)

	// Retry until the MachineSet is ready.
	return wait.PollUntilContextTimeout(ctx, RetryMedium, WaitLong, true, func(ctx context.Context) (bool, error) {
		machines, err := GetCAPIMachinesFromMachineSet(ctx, cl, machineSet)
		if err != nil {
			return false, fmt.Errorf("error getting machines from CAPI machineSet %s: %w", machineSet.Name, err)
		}

		replicas := ptr.Deref(machineSet.Spec.Replicas, 0)
		if len(machines) != int(replicas) {
			klog.Infof("%q: found %d Machines, but MachineSet has %d replicas", name, len(machines), int(replicas))
			return false, nil
		}

		// Check for machines with actual failed state (not capacity issues)
		failed := FilterCAPIMachinesInPhase(machines, string(clusterv1beta1.MachinePhaseFailed))
		if len(failed) > 0 {
			return false, handleFailedCAPIMachines(failed)
		}

		// Check if any machine did not get provisioned because of insufficient capacity.
		// Check the InfraMachine status for capacity error messages
		for _, m := range machines {
			insufficientCapacityResult, insufficientCapacityMessage, err := HasCAPIInsufficientCapacity(ctx, cl, m, capacityErrorKeys)
			if err != nil {
				return false, fmt.Errorf("error checking if CAPI machine %s has insufficient capacity: %w", m.Name, err)
			}

			if insufficientCapacityResult {
				return false, fmt.Errorf("%w: %s", ErrMachineNotProvisionedInsufficientCloudCapacity, insufficientCapacityMessage)
			}
		}

		running := FilterCAPIMachinesInPhase(machines, string(clusterv1beta1.MachinePhaseRunning))
		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			klog.Infof("%q: not all CAPI Machines are running: %d of %d", name, len(running), len(machines))
			return false, nil
		}

		for _, m := range running {
			node, err := GetCAPINodeForMachine(ctx, cl, m)
			if err != nil {
				klog.Infof("Node for CAPI machine %s not found yet: %v", m.Name, err)
				return false, nil
			}

			if !IsNodeReady(node) {
				klog.Infof("%s: node is not ready", node.Name)
				return false, nil
			}
		}

		return true, nil
	})
}

// GetCAPIInfraMachine retrieves the InfraMachine object for the given CAPI machine.
//
// Returns *unstructured.Unstructured because InfraMachine types are platform-specific
// (e.g., AWSMachine, AzureMachine, GCPMachine) and we want to handle them generically
// without importing all platform-specific APIs.
//
// Usage examples:
//   - Get spec fields: unstructured.NestedString(infraMachine.Object, "spec", "instanceType")
//   - Get status conditions: unstructured.NestedSlice(infraMachine.Object, "status", "conditions")
//   - Get nested maps: unstructured.NestedMap(infraMachine.Object, "status")
//
// The returned object contains the full InfraMachine specification and status,
// which can be accessed using the unstructured helper functions.
func GetCAPIInfraMachine(ctx context.Context, cl client.Client, m *clusterv1beta1.Machine) (*unstructured.Unstructured, error) {
	// Get the InfraMachine reference
	if m.Spec.InfrastructureRef.Name == "" {
		return nil, fmt.Errorf("machine %s has no infrastructure reference", m.Name)
	}

	// Create unstructured object to get the InfraMachine
	infraMachine := &unstructured.Unstructured{}
	infraMachine.SetAPIVersion(m.Spec.InfrastructureRef.APIVersion)
	infraMachine.SetKind(m.Spec.InfrastructureRef.Kind)

	// Get the InfraMachine object
	infraMachineKey := client.ObjectKey{
		Namespace: m.Spec.InfrastructureRef.Namespace,
		Name:      m.Spec.InfrastructureRef.Name,
	}
	if infraMachineKey.Namespace == "" {
		infraMachineKey.Namespace = m.Namespace
	}

	if err := cl.Get(ctx, infraMachineKey, infraMachine); err != nil {
		return nil, fmt.Errorf("failed to get InfraMachine %s: %w", infraMachineKey.Name, err)
	}

	return infraMachine, nil
}

// HasCAPIInsufficientCapacity returns true if the CAPI machine cannot be provisioned due to insufficient capacity.
// It checks the InfraMachine object status for capacity error messages.
// Returns: (hasInsufficientCapacity bool, capacityErrorDetails string, err error).
func HasCAPIInsufficientCapacity(ctx context.Context, cl client.Client, m *clusterv1beta1.Machine, capacityErrorKeys []string) (bool, string, error) {
	infraMachine, err := GetCAPIInfraMachine(ctx, cl, m)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, "", nil // InfraMachine not found, not a capacity issue
		}

		return false, "", err
	}

	// Extract status conditions from the InfraMachine
	statusConditions, found, err := unstructured.NestedSlice(infraMachine.Object, "status", "conditions")
	if err != nil {
		return false, "", fmt.Errorf("failed to get status conditions from InfraMachine %s: %w", m.Spec.InfrastructureRef.Name, err)
	}

	if !found {
		return false, "", nil // No conditions found
	}

	// Check each condition for capacity issues
	for _, conditionInterface := range statusConditions {
		condition, ok := conditionInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Get condition type, status, and message
		conditionType, typeOk := condition["type"].(string)
		conditionStatus, statusOk := condition["status"].(string)
		conditionMessage, msgOk := condition["message"].(string)

		if !typeOk || !statusOk || !msgOk {
			continue
		}

		// Check if this is a Ready condition with status False
		if conditionType == "InstanceReady" && conditionStatus == "False" {
			// Check for capacity error messages
			for _, errorKey := range capacityErrorKeys {
				if strings.Contains(conditionMessage, errorKey) {
					return true, conditionMessage, nil
				}
			}
		}
	}

	return false, "", nil
}

// handleFailedCAPIMachines handles the logging and error reporting for failed CAPI machines.
func handleFailedCAPIMachines(failed []*clusterv1beta1.Machine) error {
	// if there are failed machines, print them out before we exit
	klog.Errorf("found %d CAPI Machines in failed phase: ", len(failed))

	for _, m := range failed {
		reason := "reason not present in Ready condition"
		message := "message not present in Ready condition"

		// Check Ready condition for reason and message
		for _, condition := range m.Status.Conditions {
			if condition.Type != clusterv1beta1.ReadyCondition {
				continue
			}

			if condition.Reason != "" {
				reason = condition.Reason
			}

			if condition.Message != "" {
				message = condition.Message
			}

			break
		}

		klog.Errorf("Failed CAPI machine: %s, Reason: %s, Message: %s", m.Name, reason, message)
	}

	return fmt.Errorf("CAPI machine in the machineset is in a failed phase")
}

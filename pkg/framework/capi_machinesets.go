package framework

import (
	"context"
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/core/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/controllers/external"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These v1beta2 condition reasons identify non-transient provisioning failures.
const (
	invalidConfigurationReason = "InvalidConfiguration"
	unsupportedChangeReason    = "UnsupportedChange"
	joinClusterTimeoutReason   = "JoinClusterTimeoutError"
)

type CAPIMachineSetParams struct {
	msName            string
	clusterName       string
	failureDomain     string
	replicas          int32
	infrastructureRef clusterv1.ContractVersionedObjectReference
}

// NewCAPIMachineSetParams returns a new CAPIMachineSetParams object.
func NewCAPIMachineSetParams(msName, clusterName, failureDomain string, replicas int32, infrastructureRef clusterv1.ContractVersionedObjectReference) CAPIMachineSetParams {
	Expect(msName).ToNot(BeEmpty(), "expected the capi msName to not be empty")
	Expect(clusterName).ToNot(BeEmpty(), "expected the capi clusterName to not be empty")
	Expect(infrastructureRef.APIGroup).ToNot(BeEmpty(), "expected the infrastructureRef APIGroup to not be empty")
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
		ms.Spec.Template.Spec.FailureDomain = params.failureDomain
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

// GetCAPIWorkerMachineSets returns the CAPI MachineSets in the ClusterAPINamespace that label
// their Machine template with the "worker" node role.
//
// A nil, nil return means the list succeeded but no CAPI worker MachineSets were found: this is
// deliberately not treated as an error, so that callers can distinguish "no CAPI worker
// MachineSets exist" from "listing CAPI MachineSets failed" and fall through accordingly.
func GetCAPIWorkerMachineSets(ctx context.Context, cl client.Client) ([]*clusterv1.MachineSet, error) {
	machineSetList := &clusterv1.MachineSetList{}

	if err := cl.List(ctx, machineSetList, client.InNamespace(ClusterAPINamespace)); err != nil {
		return nil, fmt.Errorf("error listing CAPI MachineSets: %w", err)
	}

	var result []*clusterv1.MachineSet

	// CAPI does not label MachineSets with a role, but the installer labels the Machine
	// template with the node-role.kubernetes.io/worker label, so we check there instead.
	for i, ms := range machineSetList.Items {
		labels := ms.Spec.Template.Labels

		if labels == nil {
			continue
		}

		if _, ok := labels[clusterv1.NodeRoleLabelPrefix+"/worker"]; ok {
			result = append(result, &machineSetList.Items[i])
		}
	}

	return result, nil
}

// GetArchitectureFromCAPIMachineSetNodes returns the architecture of the nodes controlled by
// the given CAPI machineSet's machines. It mirrors GetArchitectureFromMachineSetNodes, but
// operates on CAPI types.
func GetArchitectureFromCAPIMachineSetNodes(ctx context.Context, cl client.Client, machineSet *clusterv1.MachineSet) (string, error) {
	machines, err := GetCAPIMachinesFromMachineSet(ctx, cl, machineSet)
	if err != nil {
		klog.Warningf("error getting machines for CAPI machineSet %s: %v", machineSet.Name, err)
	}

	for _, m := range machines {
		node, nodeErr := GetCAPINodeForMachine(ctx, cl, m)
		if nodeErr != nil || node == nil {
			continue
		}

		return node.Status.NodeInfo.Architecture, nil
	}

	klog.Warningf("error getting the CAPI machineSet's nodes or no nodes associated with %s. Falling back to the infrastructure MachineTemplate's NodeInfo status", machineSet.Name)

	arch, err := architectureFromInfraMachineTemplate(ctx, cl, machineSet)
	if err != nil {
		return "", fmt.Errorf("error getting the CAPI machineSet's nodes and unable to infer the architecture from its infrastructure MachineTemplate: %w", err)
	}

	return arch, nil
}

// GetWorkerMachineSetArchitecture returns the architecture of one of the cluster's worker
// MachineSets, either MAPI or CAPI.
func GetWorkerMachineSetArchitecture(ctx context.Context, cl client.Client) (string, error) {
	var errs []error

	if workers, err := GetWorkerMachineSets(ctx, cl); err != nil {
		return "", fmt.Errorf("error listing MAPI worker MachineSets: %w", err)
	} else {
		for _, ms := range workers {
			arch, archErr := GetArchitectureFromMachineSetNodes(ctx, cl, ms)
			if archErr == nil {
				return arch, nil
			}

			errs = append(errs, fmt.Errorf("error getting the architecture from MAPI worker MachineSet %s: %w", ms.Name, archErr))
		}
	}

	if workers, err := GetCAPIWorkerMachineSets(ctx, cl); err != nil {
		return "", fmt.Errorf("error listing CAPI worker MachineSets: %w", err)
	} else {
		for _, ms := range workers {
			arch, archErr := GetArchitectureFromCAPIMachineSetNodes(ctx, cl, ms)
			if archErr == nil {
				return arch, nil
			}

			errs = append(errs, fmt.Errorf("error getting the architecture from CAPI worker MachineSet %s: %w", ms.Name, archErr))
		}
	}

	if len(errs) > 0 {
		return "", fmt.Errorf("error getting the architecture from any worker MachineSet: %w", errors.Join(errs...))
	}

	return "", errNoWorkerMachineSetsFound
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

		for _, machine := range machines {
			if condition := terminalProvisioningCondition(machine); condition != nil {
				return false, fmt.Errorf("CAPI Machine %s has terminal provisioning condition %s (%s): %s",
					machine.Name, condition.Type, condition.Reason, condition.Message)
			}
		}

		replicas := ptr.Deref(machineSet.Spec.Replicas, 0)
		if len(machines) != int(replicas) {
			klog.Infof("%q: found %d Machines, but MachineSet has %d replicas", name, len(machines), int(replicas))
			return false, nil
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

		running := FilterCAPIMachinesInPhase(machines, string(clusterv1.MachinePhaseRunning))
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

// terminalProvisioningCondition returns conditions that represent known terminal errors.
// Generic NotReady and potentially transient error reasons must continue through the retry path.
func terminalProvisioningCondition(machine *clusterv1.Machine) *metav1.Condition {
	for i := range machine.Status.Conditions {
		condition := &machine.Status.Conditions[i]
		if condition.Status != metav1.ConditionFalse {
			continue
		}

		switch condition.Reason {
		case invalidConfigurationReason, unsupportedChangeReason, joinClusterTimeoutReason:
			return condition
		}
	}

	return nil
}

// HasCAPIInsufficientCapacity returns true if the CAPI machine cannot be provisioned due to insufficient capacity.
// It checks the InfraMachine object status for capacity error messages.
// Returns: (hasInsufficientCapacity bool, capacityErrorDetails string, err error).
func HasCAPIInsufficientCapacity(ctx context.Context, cl client.Client, m *clusterv1.Machine, capacityErrorKeys []string) (bool, string, error) {
	infraMachine, err := external.GetObjectFromContractVersionedRef(ctx, cl, m.Spec.InfrastructureRef, m.Namespace)
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

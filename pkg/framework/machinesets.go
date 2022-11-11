package framework

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// MachineSetParams represents the parameters for creating a new MachineSet
// resource for use in tests.
type MachineSetParams struct {
	Name         string
	Replicas     int32
	Labels       map[string]string
	Taints       []corev1.Taint
	ProviderSpec *machinev1.ProviderSpec
}

const (
	machineAPIGroup = "machine.openshift.io"
)

var (
	// ErrMachineNotProvisionedInsufficientCloudCapacity is used when we detect that the machine is not being provisioned due to insufficient provider capacity.
	ErrMachineNotProvisionedInsufficientCloudCapacity = errors.New("machine creation failed due to insufficient cloud provider capacity")

	// errTestForPlatformNotImplemented is used when platform specific test is run on a platform that does not have it implemented.
	errTestForPlatformNotImplemented = errors.New("test for current platform not implemented")

	// errMachineInMachineSetFailed is used when one of the machines in the machine set is in a failed state.
	errMachineInMachineSetFailed = errors.New("machine in the machineset is in a failed phase")
)

func BuildMachineSetParams(client runtimeclient.Client, replicas int) MachineSetParams {
	// Get the current workers MachineSets so we can copy a ProviderSpec
	// from one to use with our new dedicated MachineSet.
	workers, err := GetWorkerMachineSets(client)
	Expect(err).ToNot(HaveOccurred())

	providerSpec := workers[0].Spec.Template.Spec.ProviderSpec.DeepCopy()
	clusterName := workers[0].Spec.Template.Labels[ClusterKey]

	clusterInfra, err := GetInfrastructure(client)
	Expect(err).NotTo(HaveOccurred())
	Expect(clusterInfra.Status.InfrastructureName).ShouldNot(BeEmpty())

	uid, err := uuid.NewUUID()
	Expect(err).NotTo(HaveOccurred())

	return MachineSetParams{
		Name:         clusterInfra.Status.InfrastructureName,
		Replicas:     int32(replicas),
		ProviderSpec: providerSpec,
		Labels: map[string]string{
			"e2e.openshift.io": uid.String(),
			ClusterKey:         clusterName,
		},
		Taints: []corev1.Taint{
			{
				Key:    ClusterAPIActuatorPkgTaint,
				Effect: corev1.TaintEffectPreferNoSchedule,
			},
		},
	}
}

// CreateMachineSet creates a new MachineSet resource.
func CreateMachineSet(c client.Client, params MachineSetParams) (*machinev1.MachineSet, error) {
	ms := &machinev1.MachineSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineSet",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: params.Name,
			Namespace:    MachineAPINamespace,
			Labels:       params.Labels,
		},
		Spec: machinev1.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: params.Labels,
			},
			Template: machinev1.MachineTemplateSpec{
				ObjectMeta: machinev1.ObjectMeta{
					Labels: params.Labels,
				},
				Spec: machinev1.MachineSpec{
					ObjectMeta: machinev1.ObjectMeta{
						Labels: params.Labels,
					},
					ProviderSpec: *params.ProviderSpec,
					Taints:       params.Taints,
				},
			},
			Replicas: pointer.Int32Ptr(params.Replicas),
		},
	}

	if err := c.Create(context.Background(), ms); err != nil {
		return nil, err
	}

	return ms, nil
}

// BuildMachineSetParamsList creates a list of MachineSetParams based on the given machineSetParams with modified instance type.
func BuildAlternativeMachineSetParams(machineSetParams MachineSetParams, platform configv1.PlatformType) ([]MachineSetParams, error) {
	baseMachineSetParams := machineSetParams
	baseProviderSpec := baseMachineSetParams.ProviderSpec.DeepCopy()

	output := []MachineSetParams{}
	switch platform {
	case configv1.AWSPlatformType:
		// Using cheapest compute optimized instances that meet openshift minimum requirements (4 vCPU, 8GiB RAM)
		alternativeInstanceTypes := []string{"c5.xlarge", "c5a.xlarge", "m5.xlarge"}
		for _, instanceType := range alternativeInstanceTypes {
			updatedProviderSpec, err := updateProviderSpecAWSInstanceType(baseProviderSpec, instanceType)
			if err != nil {
				return nil, fmt.Errorf("failed to update provider spec with instance type %s: %v", instanceType, err)
			}
			baseMachineSetParams.ProviderSpec = &updatedProviderSpec
			output = append(output, baseMachineSetParams)
		}
	case configv1.AzurePlatformType:
		alternativeVMSizes := []string{"Standard_F4s_v2", "Standard_D4as_v5", "Standard_D4as_v4"}
		for _, VMSize := range alternativeVMSizes {
			updatedProviderSpec, err := updateProviderSpecAzureVMSize(baseProviderSpec, VMSize)
			if err != nil {
				return nil, fmt.Errorf("failed to update provider spec with VM size %s: %v", VMSize, err)
			}
			baseMachineSetParams.ProviderSpec = &updatedProviderSpec
			output = append(output, baseMachineSetParams)
		}
	default:
		return nil, fmt.Errorf("alternative instance types for platform %s not set", platform)
	}

	return output, nil
}

// updateProviderSpecAWSInstanceType creates a new ProviderSpec with the given instance type.
func updateProviderSpecAWSInstanceType(providerSpec *machinev1.ProviderSpec, instanceType string) (machinev1.ProviderSpec, error) {
	var awsProviderConfig machinev1.AWSMachineProviderConfig
	if err := json.Unmarshal(providerSpec.Value.Raw, &awsProviderConfig); err != nil {
		return machinev1.ProviderSpec{}, err
	}

	awsProviderConfig.InstanceType = instanceType

	updatedProviderSpec, err := json.Marshal(awsProviderConfig)
	if err != nil {
		return machinev1.ProviderSpec{}, err
	}
	newProviderSpec := machinev1.ProviderSpec{
		Value: &runtime.RawExtension{Raw: updatedProviderSpec},
	}
	return newProviderSpec, nil
}

// updateProviderSpecAzureVMSize creates a new ProviderSpec with the given VMSize.
func updateProviderSpecAzureVMSize(providerSpec *machinev1.ProviderSpec, VMSize string) (machinev1.ProviderSpec, error) {
	var azureProviderConfig machinev1.AzureMachineProviderSpec
	if err := json.Unmarshal(providerSpec.Value.Raw, &azureProviderConfig); err != nil {
		return machinev1.ProviderSpec{}, err
	}

	azureProviderConfig.VMSize = VMSize

	updatedProviderSpec, err := json.Marshal(azureProviderConfig)
	if err != nil {
		return machinev1.ProviderSpec{}, err
	}
	newProviderSpec := machinev1.ProviderSpec{
		Value: &runtime.RawExtension{Raw: updatedProviderSpec},
	}
	return newProviderSpec, nil
}

// GetMachineSets gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachineSets(client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]*machinev1.MachineSet, error) {
	machineSetList := &machinev1.MachineSetList{}

	listOpts := append([]runtimeclient.ListOption{},
		runtimeclient.InNamespace(MachineAPINamespace),
	)

	for _, selector := range selectors {
		s, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}

		listOpts = append(listOpts,
			runtimeclient.MatchingLabelsSelector{Selector: s},
		)
	}

	if err := client.List(context.Background(), machineSetList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineSetList object: %w", err)
	}

	machineSets := []*machinev1.MachineSet{}
	for _, ms := range machineSetList.Items {
		machineSet := ms
		machineSets = append(machineSets, &machineSet)
	}

	return machineSets, nil
}

// GetMachineSet gets a machineset by its name from the default machine API namespace.
func GetMachineSet(client runtimeclient.Client, name string) (*machinev1.MachineSet, error) {
	machineSet := &machinev1.MachineSet{}
	key := runtimeclient.ObjectKey{Namespace: MachineAPINamespace, Name: name}

	if err := client.Get(context.Background(), key, machineSet); err != nil {
		return nil, fmt.Errorf("error querying api for machineSet object: %w", err)
	}

	return machineSet, nil
}

// GetWorkerMachineSets returns the MachineSets that label their Machines with
// the "worker" role.
func GetWorkerMachineSets(client runtimeclient.Client) ([]*machinev1.MachineSet, error) {
	machineSets := &machinev1.MachineSetList{}

	if err := client.List(context.Background(), machineSets); err != nil {
		return nil, err
	}

	var result []*machinev1.MachineSet

	// The OpenShift installer does not label MachinSets with a type or role,
	// but the Machines themselves are labled as such via the template., so we
	// can reach into the template and check the lables there.
	for i, ms := range machineSets.Items {
		labels := ms.Spec.Template.ObjectMeta.Labels

		if labels == nil {
			continue
		}

		if labels[MachineRoleLabel] == "worker" {
			result = append(result, &machineSets.Items[i])
		}
	}

	if len(result) < 1 {
		return nil, fmt.Errorf("no worker MachineSets found")
	}

	return result, nil
}

// GetMachinesFromMachineSet returns an array of machines owned by a given machineSet
func GetMachinesFromMachineSet(client runtimeclient.Client, machineSet *machinev1.MachineSet) ([]*machinev1.Machine, error) {
	machines, err := GetMachines(client)
	if err != nil {
		return nil, fmt.Errorf("error getting machines: %w", err)
	}
	var machinesForSet []*machinev1.Machine
	for key := range machines {
		if metav1.IsControlledBy(machines[key], machineSet) {
			machinesForSet = append(machinesForSet, machines[key])
		}
	}
	return machinesForSet, nil
}

// NewMachineSet returns a new MachineSet object.
func NewMachineSet(
	clusterName, namespace, name string,
	selectorLabels map[string]string,
	templateLabels map[string]string,
	providerSpec *machinev1.ProviderSpec,
	replicas int32,
) *machinev1.MachineSet {
	ms := machinev1.MachineSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineSet",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ClusterKey: clusterName,
			},
		},
		Spec: machinev1.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					ClusterKey:    clusterName,
					MachineSetKey: name,
				},
			},
			Template: machinev1.MachineTemplateSpec{
				ObjectMeta: machinev1.ObjectMeta{
					Labels: map[string]string{
						ClusterKey:    clusterName,
						MachineSetKey: name,
					},
				},
				Spec: machinev1.MachineSpec{
					ProviderSpec: *providerSpec.DeepCopy(),
				},
			},
			Replicas: pointer.Int32Ptr(replicas),
		},
	}

	// Copy additional labels but do not overwrite those that
	// already exist.
	for k, v := range selectorLabels {
		if _, exists := ms.Spec.Selector.MatchLabels[k]; !exists {
			ms.Spec.Selector.MatchLabels[k] = v
		}
	}
	for k, v := range templateLabels {
		if _, exists := ms.Spec.Template.ObjectMeta.Labels[k]; !exists {
			ms.Spec.Template.ObjectMeta.Labels[k] = v
		}
	}

	return &ms
}

// ScaleMachineSet scales a machineSet with a given name to the given number of replicas
func ScaleMachineSet(name string, replicas int) error {
	scaleClient, err := getScaleClient()
	if err != nil {
		return fmt.Errorf("error calling getScaleClient %w", err)
	}

	scale, err := scaleClient.Scales(MachineAPINamespace).Get(context.Background(), schema.GroupResource{Group: machineAPIGroup, Resource: "MachineSet"}, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error calling scaleClient.Scales get: %w", err)
	}

	scaleUpdate := scale.DeepCopy()
	scaleUpdate.Spec.Replicas = int32(replicas)
	_, err = scaleClient.Scales(MachineAPINamespace).Update(context.Background(), schema.GroupResource{Group: machineAPIGroup, Resource: "MachineSet"}, scaleUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error calling scaleClient.Scales update: %w", err)
	}
	return nil
}

// getScaleClient returns a ScalesGetter object to manipulate scale subresources
func getScaleClient() (scale.ScalesGetter, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting config %w", err)
	}
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, fmt.Errorf("error calling NewDiscoveryRESTMapper %w", err)
	}

	discovery := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discovery)
	scaleClient, err := scale.NewForConfig(cfg, mapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)
	if err != nil {
		return nil, fmt.Errorf("error calling building scale client %w", err)
	}
	return scaleClient, nil
}

// WaitForMachineSet waits for the all Machines belonging to the named
// MachineSet to enter the "Running" phase, and for all nodes belonging to those
// Machines to be ready. If a Machine is detected in "Failed" phase, the test
// will exit early.
func WaitForMachineSet(c client.Client, name string) {
	machineSet, err := GetMachineSet(c, name)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		machines, err := GetMachinesFromMachineSet(c, machineSet)
		if err != nil {
			return err
		}

		replicas := pointer.Int32PtrDerefOr(machineSet.Spec.Replicas, 0)

		if len(machines) != int(replicas) {
			return fmt.Errorf("%q: found %d Machines, but MachineSet has %d replicas",
				name, len(machines), int(replicas))
		}

		failed := FilterMachines(machines, MachinePhaseFailed)
		if len(failed) > 0 {
			// if there are failed machines, print them out before we exit
			klog.Errorf("found %d Machines in failed phase: ", len(failed))
			for _, m := range failed {
				reason := "failureReason not present in Machine.status"
				if m.Status.ErrorReason != nil {
					reason = string(*m.Status.ErrorReason)
				}
				message := "failureMessage not present in Machine.status"
				if m.Status.ErrorMessage != nil {
					message = string(*m.Status.ErrorMessage)
				}
				klog.Errorf("Failed machine: %s, Reason: %s, Message: %s", m.Name, reason, message)
			}
		}
		Expect(len(failed)).To(Equal(0))

		running := FilterRunningMachines(machines)

		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			return fmt.Errorf("%q: not all Machines are running: %d of %d",
				name, len(running), len(machines))
		}

		for _, m := range running {
			node, err := GetNodeForMachine(c, m)
			if err != nil {
				return err
			}

			if !IsNodeReady(node) {
				return fmt.Errorf("%s: node is not ready", node.Name)
			}
		}

		return nil
	}, WaitOverLong, RetryMedium).ShouldNot(HaveOccurred())
}

// WaitForMachineSetDelete polls until the given MachineSet is not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForMachineSetDelete(c runtimeclient.Client, machineSet *machinev1.MachineSet) {
	WaitForMachineSetsDeleted(c, machineSet)
}

// WaitForSpotMachineSet waits for all Machines belonging to the machineSet to be running and their nodes to be ready.
// Unlike WaitForMachineSet, this function does not fail the test when machine cannoct be provisioned due to insufficient spot capacity.
func WaitForSpotMachineSet(c client.Client, name string) error {
	machineSet, err := GetMachineSet(c, name)
	if err != nil {
		return fmt.Errorf("could not get machineset %s: %v", name, err)
	}

	// Retry until the MachineSet is ready.
	err = wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		machines, err := GetMachinesFromMachineSet(c, machineSet)
		if err != nil {
			return false, fmt.Errorf("error getting machines from machineSet %s: %v", machineSet.Name, err)
		}

		replicas := pointer.Int32PtrDerefOr(machineSet.Spec.Replicas, 0)
		if len(machines) != int(replicas) {
			klog.Infof("%q: found %d Machines, but MachineSet has %d replicas", name, len(machines), int(replicas))
			return false, nil
		}

		failed := FilterMachines(machines, MachinePhaseFailed)
		if len(failed) > 0 {
			// if there are failed machines, print them out before we exit
			klog.Errorf("found %d Machines in failed phase: ", len(failed))
			for _, m := range failed {
				reason := "failureReason not present in Machine.status"
				if m.Status.ErrorReason != nil {
					reason = string(*m.Status.ErrorReason)
				}
				message := "failureMessage not present in Machine.status"
				if m.Status.ErrorMessage != nil {
					message = string(*m.Status.ErrorMessage)
				}
				klog.Errorf("Failed machine: %s, Reason: %s, Message: %s", m.Name, reason, message)
			}
			return false, errMachineInMachineSetFailed
		}

		// Check if any machine did not get provisioned because of insufficient spot capacity.
		for _, m := range machines {
			insufficientCapacityResult, err := hasInsufficientCapacity(m, platform)
			if err != nil {
				return false, fmt.Errorf("error checking if machine %s has insufficient capacity: %v", m.Name, err)
			}
			if insufficientCapacityResult {
				return false, ErrMachineNotProvisionedInsufficientCloudCapacity
			}
		}

		running := FilterRunningMachines(machines)
		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			klog.Infof("%q: not all Machines are running: %d of %d", name, len(running), len(machines))
			return false, nil
		}

		for _, m := range running {
			node, err := GetNodeForMachine(c, m)
			if err != nil {
				klog.Infof("Node for machine %s not found yet: %v", m.Name, err)
				return false, nil
			}

			if !IsNodeReady(node) {
				klog.Infof("%s: node is not ready", node.Name)
				return false, nil
			}
		}

		return true, nil
	})
	return err
}

// hasInsufficientCapacity return true if the machine cannot be provisioned due to insufficient spot capacity.
func hasInsufficientCapacity(m *machinev1.Machine, platform configv1.PlatformType) (bool, error) {
	switch platform {
	case configv1.AWSPlatformType:
		awsProviderStatus := machinev1.AWSMachineProviderStatus{}
		if m.Status.ProviderStatus != nil {
			err := json.Unmarshal(m.Status.ProviderStatus.Raw, &awsProviderStatus)
			if err != nil {
				return false, fmt.Errorf("error unmarshalling provider status: %v", err)
			}
			return hasInsufficientCapacityCondition(awsProviderStatus.Conditions, configv1.AWSPlatformType)
		}
	case configv1.AzurePlatformType:
		azureProviderStatus := machinev1.AzureMachineProviderStatus{}
		if m.Status.ProviderStatus != nil {
			err := json.Unmarshal(m.Status.ProviderStatus.Raw, &azureProviderStatus)
			if err != nil {
				return false, fmt.Errorf("error unmarshalling provider status: %v", err)
			}
			return hasInsufficientCapacityCondition(azureProviderStatus.Conditions, configv1.AzurePlatformType)
		}
	default:
		return false, errTestForPlatformNotImplemented
	}
	return false, nil
}

// hasInsufficientCapacity return true if there is an insufficient spot capacity condition.
func hasInsufficientCapacityCondition(conditions []metav1.Condition, platform configv1.PlatformType) (bool, error) {
	for _, condition := range conditions {
		if (condition.Type == string(machinev1.MachineCreation) || condition.Type == string(machinev1.MachineCreated)) &&
			condition.Status == metav1.ConditionFalse {
			switch platform {
			case configv1.AWSPlatformType:
				return strings.Contains(condition.Message, "InsufficientInstanceCapacity"), nil
			case configv1.AzurePlatformType:
				return strings.Contains(condition.Message, "SkuNotAvailable"), nil
			default:
				return false, errTestForPlatformNotImplemented
			}
		}
	}
	return false, nil
}

// WaitForMachineSetsDeleted polls until the given MachineSets are not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForMachineSetsDeleted(c runtimeclient.Client, machineSets ...*machinev1.MachineSet) {
	for _, ms := range machineSets {
		// Run a short check to wait for the deletion timestamp to show up.
		// If it doesn't show there's no reason to run the longer check.
		Eventually(func() error {
			machineSet := &machinev1.MachineSet{}
			err := c.Get(context.Background(), runtimeclient.ObjectKey{
				Name:      ms.GetName(),
				Namespace: ms.GetNamespace(),
			}, machineSet)
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("could not fetch MachineSet %s: %v", ms.GetName(), err)
			} else if apierrors.IsNotFound(err) {
				return nil
			}

			if machineSet.DeletionTimestamp.IsZero() {
				return fmt.Errorf("MachineSet %s still exists and does not have a deletion timestamp", ms.GetName())
			}

			// Deletion timestamp is set, so we can move on to the longer check.
			return nil
		}, WaitShort).Should(Succeed())

		Eventually(func() error {
			selector := ms.Spec.Selector

			machines, err := GetMachines(c, &selector)
			if err != nil {
				return fmt.Errorf("could not fetch Machines for MachineSet %s: %v", ms.GetName(), err)
			}

			if len(machines) != 0 {
				return fmt.Errorf("%d Machines still present for MachineSet %s", len(machines), ms.GetName())
			}

			machineSetErr := c.Get(context.Background(), runtimeclient.ObjectKey{
				Name:      ms.GetName(),
				Namespace: ms.GetNamespace(),
			}, &machinev1.MachineSet{})
			if machineSetErr != nil && !apierrors.IsNotFound(machineSetErr) {
				return fmt.Errorf("could not fetch MachineSet %s: %v", ms.GetName(), err)
			}

			// No error means the MachineSet still exists.
			if machineSetErr == nil {
				return fmt.Errorf("MachineSet %s still present, but has no Machines", ms.GetName())
			}

			return nil // MachineSet and Machines were deleted.
		}, WaitLong, RetryMedium).ShouldNot(HaveOccurred())
	}
}

// DeleteMachineSets deletes the specified machinesets and returns an error on failure.
func DeleteMachineSets(client runtimeclient.Client, machineSets ...*machinev1.MachineSet) error {
	for _, ms := range machineSets {
		if err := client.Delete(context.TODO(), ms); err != nil {
			klog.Errorf("Error querying api for machine object %q: %v, retrying...", ms.Name, err)
			return err
		}
	}
	return nil
}

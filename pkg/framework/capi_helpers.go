package framework

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-capi-operator/pkg/conversion/capi2mapi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// errUnsupportedCAPIToMAPIPlatform is returned when a CAPI to MAPI MachineSet
// conversion is attempted on a platform that is not yet supported by this
// helper.
var errUnsupportedCAPIToMAPIPlatform = errors.New("converting a CAPI MachineSet to a MAPI MachineSet is not supported for this platform")

// errUnsupportedArchitectureLookupPlatform is returned when looking up an infrastructure
// MachineTemplate's NodeInfo.Architecture is attempted on a platform that is not yet
// supported by this helper.
var errUnsupportedArchitectureLookupPlatform = errors.New("looking up the architecture from an infrastructure MachineTemplate is not supported for this platform")

// errNodeInfoArchitectureUnavailable is returned when the referenced infrastructure
// MachineTemplate does not have a populated NodeInfo.Architecture status field.
var errNodeInfoArchitectureUnavailable = errors.New("infrastructure MachineTemplate has no NodeInfo.Architecture status populated")

// ptrToClientObject is a type constraint that asserts a pointer to T is a client.Object.
type ptrToClientObject[T any] interface {
	*T
	client.Object
}

// getInfraMachineTemplateAndCluster fetches the infrastructure-specific MachineTemplate
// referenced by the given CAPI MachineSet's InfrastructureRef, as well as the
// infrastructure-specific Cluster object referenced by the MachineSet's ClusterName.
//
// The infra-cluster's Name is always equal to the CAPI Cluster's Name, which in
// turn equals ms.Spec.ClusterName.
func getInfraMachineTemplateAndCluster[
	T, C any,
	PT ptrToClientObject[T], PC ptrToClientObject[C],
](ctx context.Context, cl client.Client, ms *clusterv1beta1.MachineSet) (PT, PC, error) {
	template := PT(new(T))
	templateKey := client.ObjectKey{
		Namespace: ms.Namespace,
		Name:      ms.Spec.Template.Spec.InfrastructureRef.Name,
	}

	if err := cl.Get(ctx, templateKey, template); err != nil {
		return nil, nil, fmt.Errorf("failed to get infrastructure MachineTemplate %q: %w", templateKey.Name, err)
	}

	cluster := PC(new(C))
	clusterKey := client.ObjectKey{
		Namespace: ms.Namespace,
		Name:      ms.Spec.ClusterName,
	}

	if err := cl.Get(ctx, clusterKey, cluster); err != nil {
		return nil, nil, fmt.Errorf("failed to get infrastructure Cluster %q: %w", clusterKey.Name, err)
	}

	return template, cluster, nil
}

// architectureFromInfraMachineTemplate looks up the architecture reported by the
// platform-specific infrastructure MachineTemplate referenced by the given CAPI
// MachineSet, via the upstream "autoscaling from zero" NodeInfo status convention (see
// https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20210310-opt-in-autoscaling-from-zero.md).
func architectureFromInfraMachineTemplate(ctx context.Context, cl client.Client, ms *clusterv1beta1.MachineSet) (string, error) {
	platform, err := GetPlatform(ctx, cl)
	if err != nil {
		return "", fmt.Errorf("failed to get platform: %w", err)
	}

	switch platform {
	case configv1.AWSPlatformType:
		return architectureFromAWSMachineTemplate(ctx, cl, ms)
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedArchitectureLookupPlatform, platform)
	}
}

// architectureFromAWSMachineTemplate fetches the AWSMachineTemplate referenced by the given
// CAPI MachineSet's InfrastructureRef, and returns its Status.NodeInfo.Architecture.
func architectureFromAWSMachineTemplate(ctx context.Context, cl client.Client, ms *clusterv1beta1.MachineSet) (string, error) {
	template := &awsv1.AWSMachineTemplate{}
	key := client.ObjectKey{
		Namespace: ClusterAPINamespace,
		Name:      ms.Spec.Template.Spec.InfrastructureRef.Name,
	}

	if err := cl.Get(ctx, key, template); err != nil {
		return "", fmt.Errorf("failed to get AWSMachineTemplate %q: %w", key.Name, err)
	}

	if template.Status.NodeInfo == nil || template.Status.NodeInfo.Architecture == "" {
		return "", fmt.Errorf("%w: AWSMachineTemplate %q", errNodeInfoArchitectureUnavailable, key.Name)
	}

	return string(template.Status.NodeInfo.Architecture), nil
}

// convertCAPIWorkerMachineSetToMAPI converts a real, live CAPI worker MachineSet into an
// in-memory MAPI MachineSet, using cluster-capi-operator's capi2mapi conversion library.
//
// The returned MachineSet is never created against the API server: it is intended to be used
// only as a read-only template (e.g. to copy a ProviderSpec) by callers that need a MAPI
// MachineSet to exist, on clusters where CAPI provisions workers and no real MAPI MachineSet
// exists.
func convertCAPIWorkerMachineSetToMAPI(ctx context.Context, cl client.Client, ms *clusterv1beta1.MachineSet) (*machinev1.MachineSet, error) {
	platform, err := GetPlatform(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform: %w", err)
	}

	switch platform {
	case configv1.AWSPlatformType:
		return convertCAPIWorkerMachineSetToMAPIAWS(ctx, cl, ms)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedCAPIToMAPIPlatform, platform)
	}
}

// convertCAPIWorkerMachineSetToMAPIAWS converts an AWS CAPI worker MachineSet to a MAPI
// MachineSet, via cluster-capi-operator's capi2mapi.FromMachineSetAndAWSMachineTemplateAndAWSCluster.
func convertCAPIWorkerMachineSetToMAPIAWS(ctx context.Context, cl client.Client, ms *clusterv1beta1.MachineSet) (*machinev1.MachineSet, error) {
	awsMachineTemplate, awsCluster, err := getInfraMachineTemplateAndCluster[awsv1.AWSMachineTemplate, awsv1.AWSCluster](ctx, cl, ms)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS infrastructure objects for CAPI MachineSet %q: %w", ms.Name, err)
	}

	// This is ugly. We do it because all other users of core CAPI types in
	// cluster-api-actuator-pkg still use v1beta1, but the capi-operator
	// conversion framework uses v1beta2. We will no longer require this when we
	// update cluster-api-actuator-pkg to use v1beta2.
	v1beta2MachineSet := &clusterv1.MachineSet{}
	if err := ms.ConvertTo(v1beta2MachineSet); err != nil {
		return nil, fmt.Errorf("failed to convert CAPI MachineSet %q to v1beta2: %w", ms.Name, err)
	}

	mapiMachineSet, warnings, err := capi2mapi.FromMachineSetAndAWSMachineTemplateAndAWSCluster(v1beta2MachineSet, awsMachineTemplate, awsCluster).ToMachineSet()
	if err != nil {
		return nil, fmt.Errorf("failed to convert CAPI MachineSet %q to a MAPI MachineSet: %w", ms.Name, err)
	}

	if len(warnings) > 0 {
		klog.Warningf("warnings converting CAPI MachineSet %q to a MAPI MachineSet: %v", ms.Name, warnings)
	}

	if err := sanitizeAWSKeyName(mapiMachineSet); err != nil {
		return nil, fmt.Errorf("failed to sanitize AWS providerSpec for converted MachineSet %q: %w", ms.Name, err)
	}

	return mapiMachineSet, nil
}

// sanitizeAWSKeyName clears a MAPI AWSMachineProviderConfig's keyName field when it is a
// non-nil pointer to an empty string.
//
// CAPI's AWSMachine.Spec.SSHKeyName is tri-state: nil means "use the default SSH key",
// a pointer to "" explicitly means "do not use any SSH key", and a pointer to a name means
// "use this SSH key". capi2mapi copies that pointer as-is into MAPI's keyName field, but MAPI's
// AWS actuator has no such tri-state handling: any non-nil keyName - including an explicit empty
// string - is passed straight through to AWS's RunInstances call as the key pair name, which AWS
// rejects with "Invalid value ” for keyPairNames. It should not be blank". Real MAPI worker
// MachineSets simply omit the field (nil) to mean "no SSH key", so we normalize to that here.
func sanitizeAWSKeyName(mapiMachineSet *machinev1.MachineSet) error {
	providerSpec := mapiMachineSet.Spec.Template.Spec.ProviderSpec.Value
	if providerSpec == nil || len(providerSpec.Raw) == 0 {
		return nil
	}

	awsProviderSpec := &machinev1.AWSMachineProviderConfig{}
	if err := json.Unmarshal(providerSpec.Raw, awsProviderSpec); err != nil {
		return fmt.Errorf("failed to unmarshal AWS providerSpec: %w", err)
	}

	if awsProviderSpec.KeyName == nil || *awsProviderSpec.KeyName != "" {
		return nil
	}

	awsProviderSpec.KeyName = nil

	rawProviderSpec, err := json.Marshal(awsProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal AWS providerSpec: %w", err)
	}

	mapiMachineSet.Spec.Template.Spec.ProviderSpec.Value = &runtime.RawExtension{Raw: rawProviderSpec}

	return nil
}

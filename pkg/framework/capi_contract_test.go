package framework

import (
	"context"
	"fmt"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	capiV1beta2ContractLabel     = "cluster.x-k8s.io/v1beta2"
	awsMachineCRDName            = "awsmachines.infrastructure.cluster.x-k8s.io"
	infrastructureAPIGroup       = "infrastructure.cluster.x-k8s.io"
	insufficientInstanceCapacity = "InsufficientInstanceCapacity"
)

func TestHasCAPIInsufficientCapacityUsesContractVersionedReference(t *testing.T) {
	ctx := context.Background()
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: awsMachineCRDName,
			Labels: map[string]string{
				capiV1beta2ContractLabel: "v1beta2_v1beta1",
			},
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: infrastructureAPIGroup,
		},
	}
	infraMachine := &unstructured.Unstructured{}
	infraMachine.SetAPIVersion(infrastructureAPIGroup + "/v1beta2")
	infraMachine.SetKind("AWSMachine")
	infraMachine.SetNamespace("test")
	infraMachine.SetName("aws-machine")
	infraMachine.Object["status"] = map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"type":    "InstanceReady",
				"status":  "False",
				"message": insufficientInstanceCapacity,
			},
		},
	}

	cl := &contractTestClient{crd: crd, infraMachine: infraMachine}
	machine := contractTestMachine()

	hasCapacityIssue, message, err := HasCAPIInsufficientCapacity(ctx, cl, machine, []string{insufficientInstanceCapacity})
	if err != nil {
		t.Fatalf("HasCAPIInsufficientCapacity() error = %v", err)
	}

	if !hasCapacityIssue {
		t.Fatal("HasCAPIInsufficientCapacity() = false, want true")
	}

	if message != insufficientInstanceCapacity {
		t.Errorf("HasCAPIInsufficientCapacity() message = %q", message)
	}
}

func TestHasCAPIInsufficientCapacityReturnsCRDLookupNotFound(t *testing.T) {
	cl := &contractTestClient{
		crdErr: apierrors.NewNotFound(schema.GroupResource{Group: apiextensionsv1.GroupName, Resource: "customresourcedefinitions"},
			awsMachineCRDName),
	}

	hasCapacityIssue, message, err := HasCAPIInsufficientCapacity(context.Background(), cl, contractTestMachine(), []string{insufficientInstanceCapacity})
	if err == nil {
		t.Fatal("HasCAPIInsufficientCapacity() error = nil, want CRD lookup error")
	}

	if apierrors.IsNotFound(err) {
		t.Errorf("HasCAPIInsufficientCapacity() error = %v, CRD lookup error must not be classified as object NotFound", err)
	}

	if hasCapacityIssue || message != "" {
		t.Errorf("HasCAPIInsufficientCapacity() = (%t, %q), want (false, empty)", hasCapacityIssue, message)
	}
}

func TestHasCAPIInsufficientCapacityIgnoresMissingInfraMachine(t *testing.T) {
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: awsMachineCRDName,
			Labels: map[string]string{
				capiV1beta2ContractLabel: "v1beta2",
			},
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: infrastructureAPIGroup,
		},
	}
	cl := &contractTestClient{
		crd: crd,
		infraMachineErr: apierrors.NewNotFound(schema.GroupResource{
			Group: infrastructureAPIGroup, Resource: "awsmachines",
		}, "aws-machine"),
	}

	hasCapacityIssue, message, err := HasCAPIInsufficientCapacity(context.Background(), cl, contractTestMachine(), []string{insufficientInstanceCapacity})
	if err != nil {
		t.Fatalf("HasCAPIInsufficientCapacity() error = %v, want nil for missing InfraMachine", err)
	}

	if hasCapacityIssue || message != "" {
		t.Errorf("HasCAPIInsufficientCapacity() = (%t, %q), want (false, empty)", hasCapacityIssue, message)
	}
}

func contractTestMachine() *clusterv1.Machine {
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: "capi-machine", Namespace: "test"},
		Spec: clusterv1.MachineSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{
				APIGroup: infrastructureAPIGroup,
				Kind:     "AWSMachine",
				Name:     "aws-machine",
			},
		},
	}
}

type contractTestClient struct {
	client.Client
	crd             *apiextensionsv1.CustomResourceDefinition
	crdErr          error
	infraMachine    *unstructured.Unstructured
	infraMachineErr error
}

func (c *contractTestClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch target := obj.(type) {
	case *metav1.PartialObjectMetadata:
		if key.Name != awsMachineCRDName || key.Namespace != "" {
			return fmt.Errorf("unexpected CRD lookup key %v", key)
		}

		if c.crdErr != nil {
			return c.crdErr
		}

		target.ObjectMeta = *c.crd.ObjectMeta.DeepCopy()
	case *unstructured.Unstructured:
		if key.Name != "aws-machine" || key.Namespace != "test" || target.GetKind() != "AWSMachine" {
			return fmt.Errorf("unexpected infrastructure lookup: kind %q, key %v", target.GetKind(), key)
		}

		if c.infraMachineErr != nil {
			return c.infraMachineErr
		}

		if target.GetAPIVersion() != c.infraMachine.GetAPIVersion() {
			return fmt.Errorf("requested API version %q does not match fixture API version %q", target.GetAPIVersion(), c.infraMachine.GetAPIVersion())
		}

		*target = *c.infraMachine.DeepCopy()
	default:
		return fmt.Errorf("unexpected object type %T", obj)
	}

	return nil
}

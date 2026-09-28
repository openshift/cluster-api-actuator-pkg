package framework

import (
	"context"
	"errors"
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

func TestAPIVersionForCAPIContract(t *testing.T) {
	tests := []struct {
		name           string
		labels         map[string]string
		wantAPIVersion string
		wantErr        bool
	}{
		{
			name: "uses the last non-empty API version from the v1beta2 contract",
			labels: map[string]string{
				"cluster.x-k8s.io/v1beta1": "v1beta3",
				capiV1beta2ContractLabel:   "v1beta3__v1beta2_",
			},
			wantAPIVersion: infrastructureAPIGroup + "/v1beta2",
		},
		{
			name: "falls back to a v1beta1 provider contract",
			labels: map[string]string{
				"cluster.x-k8s.io/v1beta1": "v1beta1",
			},
			wantAPIVersion: infrastructureAPIGroup + "/v1beta1",
		},
		{
			name:    "errors when no compatible provider contract is advertised",
			labels:  map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:   awsMachineCRDName,
					Labels: tt.labels,
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: infrastructureAPIGroup,
				},
			}

			got, err := apiVersionForCAPIContract(crd)
			if (err != nil) != tt.wantErr {
				t.Fatalf("apiVersionForCAPIContract() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && got != tt.wantAPIVersion {
				t.Errorf("apiVersionForCAPIContract() = %q, want %q", got, tt.wantAPIVersion)
			}
		})
	}
}

func TestGetCAPIInfraMachineUsesContractVersionedReference(t *testing.T) {
	ctx := context.Background()
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: awsMachineCRDName,
			Labels: map[string]string{
				capiV1beta2ContractLabel: "v1beta3__v1beta2_",
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

	got, err := GetCAPIInfraMachine(ctx, cl, machine)
	if err != nil {
		t.Fatalf("GetCAPIInfraMachine() error = %v", err)
	}

	if got.GetAPIVersion() != infrastructureAPIGroup+"/v1beta2" {
		t.Errorf("GetCAPIInfraMachine() apiVersion = %q", got.GetAPIVersion())
	}

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

	if !errors.Is(err, errCAPIInfraMachineCRDNotFound) {
		t.Errorf("HasCAPIInsufficientCapacity() error = %v, want CRD lookup error", err)
	}

	if !apierrors.IsNotFound(err) {
		t.Errorf("HasCAPIInsufficientCapacity() error = %v, want wrapped NotFound cause", err)
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

func (c *contractTestClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch target := obj.(type) {
	case *apiextensionsv1.CustomResourceDefinition:
		if c.crdErr != nil {
			return c.crdErr
		}

		*target = *c.crd.DeepCopy()
	case *unstructured.Unstructured:
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

package framework

import (
	"context"
	"fmt"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAPIVersionForCAPIContract(t *testing.T) {
	tests := []struct {
		name          string
		labels        map[string]string
		wantAPIVersion string
		wantErr       bool
	}{
		{
			name: "uses the v1beta2 contract and newest compatible API version",
			labels: map[string]string{
				"cluster.x-k8s.io/v1beta1": "v1beta3",
				"cluster.x-k8s.io/v1beta2": "v1beta2_v1beta3",
			},
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta3",
		},
		{
			name: "falls back to a v1beta1 provider contract",
			labels: map[string]string{
				"cluster.x-k8s.io/v1beta1": "v1beta1",
			},
			wantAPIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
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
					Name:   "awsmachines.infrastructure.cluster.x-k8s.io",
					Labels: tt.labels,
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "infrastructure.cluster.x-k8s.io",
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
			Name: "awsmachines.infrastructure.cluster.x-k8s.io",
			Labels: map[string]string{
				"cluster.x-k8s.io/v1beta2": "v1beta2",
			},
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "infrastructure.cluster.x-k8s.io",
		},
	}
	infraMachine := &unstructured.Unstructured{}
	infraMachine.SetAPIVersion("infrastructure.cluster.x-k8s.io/v1beta2")
	infraMachine.SetKind("AWSMachine")
	infraMachine.SetNamespace("test")
	infraMachine.SetName("aws-machine")
	infraMachine.Object["status"] = map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"type":    "InstanceReady",
				"status":  "False",
				"message": "InsufficientInstanceCapacity",
			},
		},
	}

	cl := &contractTestClient{crd: crd, infraMachine: infraMachine}
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: "capi-machine", Namespace: "test"},
		Spec: clusterv1.MachineSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{
				APIGroup: "infrastructure.cluster.x-k8s.io",
				Kind:     "AWSMachine",
				Name:     "aws-machine",
			},
		},
	}

	got, err := GetCAPIInfraMachine(ctx, cl, machine)
	if err != nil {
		t.Fatalf("GetCAPIInfraMachine() error = %v", err)
	}
	if got.GetAPIVersion() != "infrastructure.cluster.x-k8s.io/v1beta2" {
		t.Errorf("GetCAPIInfraMachine() apiVersion = %q", got.GetAPIVersion())
	}

	hasCapacityIssue, message, err := HasCAPIInsufficientCapacity(ctx, cl, machine, []string{"InsufficientInstanceCapacity"})
	if err != nil {
		t.Fatalf("HasCAPIInsufficientCapacity() error = %v", err)
	}
	if !hasCapacityIssue {
		t.Fatal("HasCAPIInsufficientCapacity() = false, want true")
	}
	if message != "InsufficientInstanceCapacity" {
		t.Errorf("HasCAPIInsufficientCapacity() message = %q", message)
	}
}

type contractTestClient struct {
	client.Client
	crd          *apiextensionsv1.CustomResourceDefinition
	infraMachine *unstructured.Unstructured
}

func (c *contractTestClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch target := obj.(type) {
	case *apiextensionsv1.CustomResourceDefinition:
		*target = *c.crd.DeepCopy()
	case *unstructured.Unstructured:
		*target = *c.infraMachine.DeepCopy()
	default:
		return fmt.Errorf("unexpected object type %T", obj)
	}

	return nil
}

package framework

import (
	"context"

	mapiv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineHealthCheckParams represents the parameters for creating a
// new MachineHealthCheck resource for use in tests.
type MachineHealthCheckParams struct {
	Name         string
	Labels       map[string]string
	Conditions   []mapiv1beta1.UnhealthyCondition
	MaxUnhealthy *int
}

// CreateMHC creates a new MachineHealthCheck resource.
func CreateMHC(c client.Client, params MachineHealthCheckParams) (*mapiv1beta1.MachineHealthCheck, error) {
	mhc := &mapiv1beta1.MachineHealthCheck{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machine.openshift.io/v1beta1",
			Kind:       "MachineHealthCheck",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: MachineAPINamespace,
		},
		Spec: mapiv1beta1.MachineHealthCheckSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: params.Labels,
			},
			UnhealthyConditions: params.Conditions,
		},
	}

	if params.MaxUnhealthy != nil {
		maxUnhealthy := intstr.FromInt(*params.MaxUnhealthy)
		mhc.Spec.MaxUnhealthy = &maxUnhealthy
	}

	if err := c.Create(context.Background(), mhc); err != nil {
		return nil, err
	}

	return mhc, nil
}

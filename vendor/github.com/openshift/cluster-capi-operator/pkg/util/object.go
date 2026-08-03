/*
Copyright 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package util

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errObjectNotClientObject = errors.New("object does not implement client.Object")

// IsNilObject checks whether a client.Object is nil or not.
func IsNilObject(obj client.Object) bool {
	return obj == nil || reflect.ValueOf(obj).IsNil()
}

// GetReferencedObject retrieves a Kubernetes object dynamically based on an ObjectReference.
func GetReferencedObject(ctx context.Context, c client.Reader, scheme *runtime.Scheme, ref corev1.ObjectReference) (client.Object, error) {
	// Construct GVK from the reference.
	gvk := schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind)

	// Create an empty object dynamically.
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to create object for GVK %v: %w", gvk, err)
	}

	// Ensure it implements client.Object.
	clientObj, ok := obj.(client.Object)
	if !ok {
		return nil, errObjectNotClientObject
	}

	// Set GVK explicitly.
	clientObj.GetObjectKind().SetGroupVersionKind(gvk)

	// Fetch the object from the cluster.
	if err := c.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, clientObj); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("object %s/%s not found: %w", ref.Namespace, ref.Name, err)
		}

		return nil, fmt.Errorf("failed to get object %s/%s: %w", ref.Namespace, ref.Name, err)
	}

	return clientObj, nil
}

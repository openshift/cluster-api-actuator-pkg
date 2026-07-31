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

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	errUnableToAssertClientObject = errors.New("unable to assert client.Object after deepcopy")
	errObjectPassedIsNil          = errors.New("failed to ensure finalizer: object passed is nil")
)

// EnsureFinalizer ensures that the specified finalizer is added to the given object using a Patch operation.
func EnsureFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) (bool, error) {
	if IsNilObject(obj) {
		return false, errObjectPassedIsNil
	}
	// Create a deep copy of the original object.
	originalObj, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return false, errUnableToAssertClientObject
	}

	if updated := controllerutil.AddFinalizer(obj, finalizer); !updated {
		return false, nil
	}
	// Apply the patch operation.
	if err := c.Patch(ctx, obj, client.MergeFrom(originalObj)); err != nil {
		if kerrors.IsNotFound(err) {
			return false, fmt.Errorf("object %s/%s not found: %w", obj.GetNamespace(), obj.GetName(), err)
		}

		return false, fmt.Errorf("error patching resource %s/%s to add finalizer: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	return true, nil
}

// RemoveFinalizer ensures that the specified finalizer is removed from the given object using a Patch operation.
func RemoveFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) (bool, error) {
	if IsNilObject(obj) {
		return false, nil
	}

	// Create a deep copy of the original object.
	originalObj, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return false, errUnableToAssertClientObject
	}

	if updated := controllerutil.RemoveFinalizer(obj, finalizer); !updated {
		return false, nil
	}

	// Apply the patch operation.
	if err := c.Patch(ctx, obj, client.MergeFrom(originalObj)); err != nil {
		if kerrors.IsNotFound(err) {
			return false, fmt.Errorf("object %s/%s not found: %w", obj.GetNamespace(), obj.GetName(), err)
		}

		return false, fmt.Errorf("error patching resource %s/%s to remove finalizer: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	return true, nil
}

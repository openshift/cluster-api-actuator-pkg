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
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/go-test/deep"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errObjectsToCompareCannotBeNil = errors.New("objects to diff cannot be nil")
	errObjectsToCompareNotSameType = errors.New("objects to diff are not of the same type")
	errProviderSpecNotFound        = errors.New("providerSpec not found")
)

// DiffResult is the interface that represents the result of a diff operation.
type DiffResult interface {
	HasChanges() bool
	String() string
	HasMetadataChanges() bool
	HasSpecChanges() bool
	HasProviderSpecChanges() bool
	HasStatusChanges() bool
}

// diffResult is the implementation of the DiffResult interface.
type diffResult struct {
	diff             map[string][]string
	providerSpecPath string
}

// HasChanges returns true if the diff detected any changes.
func (d *diffResult) HasChanges() bool {
	return len(d.diff) > 0
}

// HasMetadataChanges returns true if the diff detected any changes to the metadata.
func (d *diffResult) HasMetadataChanges() bool {
	_, ok := d.diff["metadata"]
	return ok
}

// HasSpecChanges returns true if the diff detected any changes to the spec.
func (d *diffResult) HasSpecChanges() bool {
	_, ok := d.diff["spec"]
	return ok
}

// HasProviderSpecChanges returns true if the diff detected any changes to the providerSpec.
// Only ever returns true if d.providerSpecPath was set.
func (d *diffResult) HasProviderSpecChanges() bool {
	if d.providerSpecPath == "" {
		return false
	}

	_, ok := d.diff[d.providerSpecPath]

	return ok
}

// HasStatusChanges returns true if the diff detected any changes to the status.
func (d *diffResult) HasStatusChanges() bool {
	_, ok := d.diff["status"]
	return ok
}

// String returns the diff as a string.
func (d *diffResult) String() string {
	if !d.HasChanges() {
		return ""
	}

	diff := []string{}

	for k, v := range d.diff {
		for _, d := range v {
			if strings.Contains(d, "]: ") {
				diff = append(diff, fmt.Sprintf("[%s].%s", k, d))
			} else {
				diff = append(diff, fmt.Sprintf("[%s]: %s", k, d))
			}
		}
	}

	sort.Strings(diff)

	out := "." + strings.Join(diff, ", .")
	out = strings.ReplaceAll(out, ".slice[", "[")
	out = strings.ReplaceAll(out, "map[", "[")

	return out
}

type differ struct {
	modifyFuncs     map[string]func(obj map[string]interface{}) error
	lateModifyFuncs map[string]func(obj map[string]interface{}) error
	ignoredPath     [][]string

	// providerSpecPath is a custom path to the providerSpec field which differs between
	// Machine API Machines and MachineSets and is passed down to the result to enable HasProviderSpecChanges().
	// It is only used when set, so it does not make a difference for non Machine API objects.
	providerSpecPath string
}

// Diff compares the objects a and b, and returns a DiffResult.
func (d *differ) Diff(a, b client.Object) (DiffResult, error) {
	if a == nil || b == nil {
		return nil, errObjectsToCompareCannotBeNil
	}

	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return nil, fmt.Errorf("%w: %T != %T", errObjectsToCompareNotSameType, a, b)
	}

	// 1. Convert the objects to unstructured.
	unstructuredA, err := runtime.DefaultUnstructuredConverter.ToUnstructured(a)
	if err != nil {
		return nil, fmt.Errorf("failed to convert a to unstructured: %w", err)
	}

	unstructuredB, err := runtime.DefaultUnstructuredConverter.ToUnstructured(b)
	if err != nil {
		return nil, fmt.Errorf("failed to convert b to unstructured: %w", err)
	}

	// 2. Run the configured modify functions
	// This allows customizing the diffing process, e.g. remove conditions last transition time to ignore them during diffing
	// or separate handling for providerSpec.
	if err := d.applyModifyFuncs(unstructuredA, unstructuredB, d.modifyFuncs); err != nil {
		return nil, fmt.Errorf("failed to apply modify functions: %w", err)
	}

	// 3. Remove fields configured to be ignored.
	for _, ignorePath := range d.ignoredPath {
		unstructured.RemoveNestedField(unstructuredA, ignorePath...)
		unstructured.RemoveNestedField(unstructuredB, ignorePath...)
	}

	// 4. Run the late modify functions.
	// This allows customize the diffing process to make objects better comparable. E.g. compare conditions as maps with their type as key.
	if err := d.applyModifyFuncs(unstructuredA, unstructuredB, d.lateModifyFuncs); err != nil {
		return nil, fmt.Errorf("failed to apply modify functions: %w", err)
	}

	// 4. Diff both resulted unstructured objects.
	// Record the result for each top-level key in the maps, so it can be used later on for the `Has*Changes` functions.

	// Collect all top-level keys.
	allKeys := sets.Set[string]{}

	for k := range unstructuredA {
		allKeys.Insert(k)
	}

	for k := range unstructuredB {
		allKeys.Insert(k)
	}

	diffByKey := map[string][]string{}

	// Diff each top-level key separately and record the output to the diff map.
	for k := range allKeys {
		diff := deep.Equal(unstructuredA[k], unstructuredB[k])

		// Make the result deterministic.
		sort.Strings(diff)

		if len(diff) > 0 {
			diffByKey[k] = diff
		}
	}

	return &diffResult{
		diff:             diffByKey,
		providerSpecPath: d.providerSpecPath,
	}, nil
}

func (d *differ) applyModifyFuncs(a, b map[string]interface{}, modifyFuncs map[string]func(obj map[string]interface{}) error) error {
	for funcName, modifyFunc := range modifyFuncs {
		if err := modifyFunc(a); err != nil {
			return fmt.Errorf("modify function %s on a failed: %w", funcName, err)
		}

		if err := modifyFunc(b); err != nil {
			return fmt.Errorf("modify function %s on b failed: %w", funcName, err)
		}
	}

	return nil
}

// NewDefaultDiffer creates a new default differ with the default options.
func NewDefaultDiffer(opts ...DiffOption) *differ {
	return newDiffer(append(opts,
		// Always ignore kind and apiVersion as they may not be always set. Instead the differ checks if the input objects have the same type.
		WithIgnoreField("kind"),
		WithIgnoreField("apiVersion"),

		// Options for handling of metadata fields.

		// Special handling for Cluster API's conversion-data label.
		WithIgnoreField("metadata", "annotations", "cluster.x-k8s.io/conversion-data"),

		WithIgnoreField("metadata", "name"),
		WithIgnoreField("metadata", "generateName"),
		WithIgnoreField("metadata", "namespace"),
		WithIgnoreField("metadata", "selfLink"),
		WithIgnoreField("metadata", "uid"),
		WithIgnoreField("metadata", "resourceVersion"),
		WithIgnoreField("metadata", "generation"),
		WithIgnoreField("metadata", "creationTimestamp"),
		WithIgnoreField("metadata", "deletionTimestamp"),
		WithIgnoreField("metadata", "deletionGracePeriodSeconds"),
		WithIgnoreField("metadata", "finalizers"),
		WithIgnoreField("metadata", "managedFields"),

		// Options for handling of status fields.
		WithIgnoreConditionsLastTransitionTime(),
		WithConditionsAsMap(),
	)...)
}

// WithIgnoreField adds a path to the list of paths to ignore when executing Diff.
func WithIgnoreField(path ...string) DiffOption {
	return func(d *differ) {
		d.ignoredPath = append(d.ignoredPath, path)
	}
}

// WithIgnoreConditionsLastTransitionTime configures the differ to ignore LastTransitionTime for conditions when executing Diff.
func WithIgnoreConditionsLastTransitionTime() DiffOption {
	return func(d *differ) {
		d.modifyFuncs["RemoveConditionsLastTransitionTime"] = func(a map[string]interface{}) error {
			conditionPaths := [][]string{
				{"status", "conditions"},
				{"status", "v1beta2", "conditions"},
				{"status", "deprecated", "v1beta1", "conditions"},
			}

			for _, conditionPath := range conditionPaths {
				conditions, found, err := unstructured.NestedSlice(a, conditionPath...)
				if !found || err != nil {
					continue
				}

				for i, condition := range conditions {
					conditionMap, ok := condition.(map[string]interface{})
					if !ok {
						continue
					}

					conditionMap["lastTransitionTime"] = "ignored"
					conditions[i] = conditionMap
				}

				if err := unstructured.SetNestedField(a, conditions, conditionPath...); err != nil {
					return fmt.Errorf("failed to set nested field %s: %w", strings.Join(conditionPath, "."), err)
				}
			}

			return nil
		}
	}
}

// WithConditionsAsMap ensures the conditions are converted to maps for comparison.
func WithConditionsAsMap() DiffOption {
	return func(d *differ) {
		d.lateModifyFuncs["ConditionsAsMap"] = func(a map[string]interface{}) error {
			conditionPaths := [][]string{
				{"status", "conditions"},
				{"status", "v1beta2", "conditions"},
				{"status", "deprecated", "v1beta1", "conditions"},
			}

			for _, conditionPath := range conditionPaths {
				conditions, found, err := unstructured.NestedSlice(a, conditionPath...)
				if !found || err != nil {
					continue
				}

				newConditions := map[string]interface{}{}

				for _, condition := range conditions {
					conditionMap, ok := condition.(map[string]interface{})
					if !ok || conditionMap["type"] == nil {
						continue
					}

					conditionType, ok := conditionMap["type"].(string)
					if !ok {
						continue
					}

					newConditions[fmt.Sprintf("type=%s", conditionType)] = condition
				}

				if err := unstructured.SetNestedField(a, newConditions, conditionPath...); err != nil {
					return fmt.Errorf("failed to set nested field %s: %w", strings.Join(conditionPath, "."), err)
				}
			}

			return nil
		}
	}
}

// WithIgnoreConditionType conditionType configures the differ to ignore the condition of the given type when executing Diff.
func WithIgnoreConditionType(conditionType string) DiffOption {
	return func(d *differ) {
		d.modifyFuncs[fmt.Sprintf("RemoveCondition[%s]", conditionType)] = func(a map[string]interface{}) error {
			conditionPaths := [][]string{
				{"status", "conditions"},
				{"status", "v1beta2", "conditions"},
				{"status", "deprecated", "v1beta1", "conditions"},
			}

			for _, conditionPath := range conditionPaths {
				conditions, found, err := unstructured.NestedSlice(a, conditionPath...)
				if !found || err != nil {
					continue
				}

				newConditions := []interface{}{}

				for _, condition := range conditions {
					conditionMap, ok := condition.(map[string]interface{})
					if !ok {
						continue
					}

					// Skip condition of the given type.
					if conditionMap["type"] == conditionType {
						continue
					}

					newConditions = append(newConditions, condition)
				}

				if err := unstructured.SetNestedField(a, newConditions, conditionPath...); err != nil {
					return fmt.Errorf("failed to set nested field %s: %w", strings.Join(conditionPath, "."), err)
				}
			}

			return nil
		}
	}
}

// WithProviderSpec configures the differ to separately diff .spec.providerSpec.
func WithProviderSpec(platform configv1.PlatformType, path []string, marshalProviderSpec func(platform configv1.PlatformType, rawExtension *runtime.RawExtension) (any, error)) DiffOption {
	return func(d *differ) {
		d.providerSpecPath = strings.Join(path, ".")

		d.modifyFuncs["ProviderSpec"] = func(obj map[string]interface{}) error {
			rawExtensionMap, found, err := unstructured.NestedMap(obj, path...)
			if err != nil {
				return fmt.Errorf("failed to get providerSpec value: %w", err)
			} else if !found {
				return fmt.Errorf("%w at path %s", errProviderSpecNotFound, strings.Join(path, "."))
			}

			rawExtension := &runtime.RawExtension{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rawExtensionMap, rawExtension); err != nil {
				return fmt.Errorf("failed to convert providerSpec value map to raw extension: %w", err)
			}

			providerSpec, err := marshalProviderSpec(platform, rawExtension)
			if err != nil {
				return fmt.Errorf("failed to marshal providerSpec: %w", err)
			}

			// unset the nested field
			unstructured.RemoveNestedField(obj, path...)
			// add it as top-level field
			obj[d.providerSpecPath] = providerSpec

			return nil
		}
	}
}

// DiffOption is the type for options to configure the differ.
type DiffOption func(*differ)

func newDiffer(opts ...DiffOption) *differ {
	d := &differ{
		modifyFuncs:     map[string]func(obj map[string]interface{}) error{},
		lateModifyFuncs: map[string]func(obj map[string]interface{}) error{},
	}
	for _, opt := range opts {
		opt(d)
	}

	return d
}

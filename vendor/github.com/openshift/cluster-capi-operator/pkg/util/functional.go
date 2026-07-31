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
	"iter"
)

// SliceMap applies a map function to each element of a slice and returns a new slice.
func SliceMap[A, B any](source []A, fn func(A) B) []B {
	if len(source) == 0 {
		return nil
	}

	result := make([]B, len(source))
	for i, a := range source {
		result[i] = fn(a)
	}

	return result
}

// SliceFilter applies a filter function to each element of a slice and returns a new slice.
func SliceFilter[A any](source []A, fn func(A) bool) []A {
	result := []A{}

	for _, a := range source {
		if fn(a) {
			result = append(result, a)
		}
	}

	return result
}

// IterFilter returns a new iterator that applies a filter to the input iterator.
func IterFilter[T any](i iter.Seq[T], filter func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range i {
			if filter(item) {
				if !yield(item) {
					return
				}
			}
		}
	}
}

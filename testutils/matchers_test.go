/*
Copyright 2022 Red Hat, Inc.

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

package testutils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Matchers", func() {
	Context("MatchViaJSON", func() {
		type testStruct struct {
			Name   string            `json:"name"`
			Age    int               `json:"age"`
			Labels map[string]string `json:"labels"`
			List   []string          `json:"list"`
		}

		testStructBasic := testStruct{
			Name: "John",
			Age:  30,
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			List: []string{"a", "b", "c"},
		}

		testStructVaryingMapOrder := testStruct{
			Name: "John",
			Age:  30,
			Labels: map[string]string{
				"key2": "value2",
				"key1": "value1",
			},
			List: []string{"a", "b", "c"},
		}

		testStructMissingMapEntry := testStruct{
			Name: "John",
			Age:  30,
			Labels: map[string]string{
				"key1": "value1",
			},
			List: []string{"a", "b", "c"},
		}

		testStructMissingListEntry := testStruct{
			Name: "John",
			Age:  30,
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			List: []string{"a", "b"},
		}

		testStructExtraListEntry := testStruct{
			Name: "John",
			Age:  30,
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			List: []string{"a", "b", "c", "d"},
		}

		testStructDifferentListValue := testStruct{
			Name: "John",
			Age:  30,
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			List: []string{"a", "b", "d"},
		}

		testStructDifferentInt := testStruct{
			Name: "John",
			Age:  31,
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			List: []string{"a", "b", "c"},
		}

		testStructDifferentString := testStruct{
			Name: "Jane",
			Age:  30,
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			List: []string{"a", "b", "c"},
		}

		type testMatchViaJSONInput struct {
			actual      interface{}
			expected    interface{}
			shouldMatch bool
		}

		DescribeTable("MatchViaJSON should match objects via JSON", func(in testMatchViaJSONInput) {
			matcher := MatchViaJSON(in.expected)
			matched, err := matcher.Match(in.actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(Equal(in.shouldMatch))
		},
			Entry("should match identical objects", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructBasic,
				shouldMatch: true,
			}),
			Entry("should match objects with varying map ordering", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructVaryingMapOrder,
				shouldMatch: true,
			}),
			Entry("should not match objects with missing map entry", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructMissingMapEntry,
				shouldMatch: false,
			}),
			Entry("should not match objects with missing list entry", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructMissingListEntry,
				shouldMatch: false,
			}),
			Entry("should not match objects with extra list entry", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructExtraListEntry,
				shouldMatch: false,
			}),
			Entry("should not match objects with different list value", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructDifferentListValue,
				shouldMatch: false,
			}),
			Entry("should not match objects with different int value", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructDifferentInt,
				shouldMatch: false,
			}),
			Entry("should not match objects with different string value", testMatchViaJSONInput{
				actual:      testStructBasic,
				expected:    testStructDifferentString,
				shouldMatch: false,
			}),
		)
	})
})

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
	"errors"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testLogMessage = "some log message"
	testLogDetail  = "detail"
	testLogExtra   = "extra"
)

var _ = Describe("TestLogger", func() {
	var logger TestLogger

	BeforeEach(func() {
		logger = NewTestLogger()
	})

	It("adds info logs to the entries", func() {
		logger.Logger().V(1).Info(testLogMessage, testLogExtra, testLogDetail)
		logger.Logger().V(3).Info("another log message", "key", "value")

		Expect(logger.Entries()).To(ConsistOf(
			LogEntry{
				Level:         1,
				Message:       testLogMessage,
				KeysAndValues: []interface{}{testLogExtra, testLogDetail},
			},
			LogEntry{
				Level:         3,
				Message:       "another log message",
				KeysAndValues: []interface{}{"key", "value"},
			},
		))
	})

	It("adds error logs to the entries", func() {
		err := errors.New("error")
		logger.Logger().Error(err, testLogMessage, testLogExtra, testLogDetail)

		Expect(logger.Entries()).To(ConsistOf(
			LogEntry{
				Error:         err,
				Message:       testLogMessage,
				KeysAndValues: []interface{}{testLogExtra, testLogDetail},
			},
		))
	})

	Context("when adding extra keys and values to the logger", func() {
		var myKeyValueLogger logr.Logger

		BeforeEach(func() {
			myKeyValueLogger = logger.Logger().WithValues("myKey", "myValue")
		})

		It("adds the key and value to the logs", func() {
			myKeyValueLogger.V(2).Info(testLogMessage, testLogExtra, testLogDetail)

			Expect(logger.Entries()).To(ConsistOf(
				LogEntry{
					Level:         2,
					Message:       testLogMessage,
					KeysAndValues: []interface{}{"myKey", "myValue", testLogExtra, testLogDetail},
				},
			))
		})

		It("does not add the key and value to the logs from the parent logger", func() {
			logger.Logger().V(2).Info(testLogMessage, testLogExtra, testLogDetail)

			Expect(logger.Entries()).To(ConsistOf(
				LogEntry{
					Level:         2,
					Message:       testLogMessage,
					KeysAndValues: []interface{}{testLogExtra, testLogDetail},
				},
			))
		})
	})
})

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

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type terminalWithReasonError struct {
	error
	Reason string
}

var _ error = &terminalWithReasonError{}

// Unwrap returns the wrapped error.
func (e *terminalWithReasonError) Unwrap() error {
	return e.error
}

// Error returns the error message.
func (e *terminalWithReasonError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.error.Error())
}

// Is checks if the error is a terminal error with a reason.
func (e *terminalWithReasonError) Is(target error) bool {
	tp := &terminalWithReasonError{}
	return errors.As(target, &tp)
}

// TerminalWithReasonError returns a terminal error with a reason.
// This will prevent the reconcile from being retried.
func TerminalWithReasonError(err error, reason string) error {
	return reconcile.TerminalError(&terminalWithReasonError{err, reason}) //nolint:wrapcheck
}

// AsTerminalWithReasonError checks if the error is a terminal error with a reason.
// If it is, it returns the terminal error with a reason.
// If it is not, it returns nil.
func AsTerminalWithReasonError(err error) *terminalWithReasonError {
	terminalWithReasonError := &terminalWithReasonError{}
	if errors.As(err, &terminalWithReasonError) {
		return terminalWithReasonError
	}

	return nil
}

// IsTerminalWithReasonError checks if the error is a terminal error with a reason.
func IsTerminalWithReasonError(err error) bool {
	return errors.Is(err, &terminalWithReasonError{})
}

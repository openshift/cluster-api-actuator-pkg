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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
)

// GenerateInfraMachineTemplateNameWithSpecHash generates hash infra machine spec and combines it with infra machine name.
// Resulting name is "<name>-<hash>" where <hash> is a hex string of 8 characters.
// The hash function is FNV-1a 32-bit.
func GenerateInfraMachineTemplateNameWithSpecHash(name string, spec interface{}) (string, error) {
	jsonSpec, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal spec when creating infrastructure machine template: %w", err)
	}

	hasher := fnv.New32a()
	if _, err := hasher.Write(jsonSpec); err != nil {
		return "", fmt.Errorf("failed to write to hash: %w", err)
	}

	hashBytes := hasher.Sum(nil) // 32 bits make 8 hex digits

	return fmt.Sprintf("%s-%s", name, hex.EncodeToString(hashBytes)), nil
}

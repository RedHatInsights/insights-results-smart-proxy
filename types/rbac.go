// Copyright 2024 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

// RbacResponse represents structure of a response from the RBAC service.
type RbacResponse struct {
	Meta  RbacMetadata `json:"meta"`
	Links rbacLinks    `json:"links"`
	Data  []RbacData   `json:"data"`
}

// RbacMetadata holds pagination information provided by the RBAC service.
type RbacMetadata struct {
	Count  int `json:"count"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// RbacData represents a single permission record or Access Control Entry.
type RbacData struct {
	ResourceDefinitions []rbacResourceDefinitions `json:"resourceDefinitions,omitempty"`
	Permission          string                    `json:"permission,omitempty"`
}

type rbacResourceDefinitions struct {
	AttributeFilter attributeFilter `json:"attributeFilter,omitempty"`
}

type attributeFilter struct {
	Key       string
	Value     interface{}
	Operation string
}

type rbacLinks struct {
	First    string
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Last     string
}

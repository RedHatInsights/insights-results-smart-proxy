/*
Copyright © 2024 Red Hat, Inc.

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

package auth_test

import (
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/stretchr/testify/assert"
)

// TestAuthenticationError checks the method Error() for data structure
// AuthenticationError
func TestAuthenticationError(t *testing.T) {
	// expected error value
	const expected = "errorMessage"

	// construct an instance of error interface
	err := auth.AuthenticationError{
		ErrString: "errorMessage"}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestAuthorizationError checks the method Error() for data structure
// AuthorizationError
func TestAuthorizationError(t *testing.T) {
	// expected error value
	const expected = "errorMessage"

	// construct an instance of error interface
	err := auth.AuthorizationError{
		ErrString: "errorMessage"}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

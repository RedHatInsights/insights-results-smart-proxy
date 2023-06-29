// Copyright 2023 Red Hat, Inc
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

package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
)

// TestRouterMissingParamError checks the method Error() for data structure
// RouterMissingParamError
func TestRouterMissingParamError(t *testing.T) {
	// expected error value
	const expected = "Missing required param from request: paramName"

	// construct an instance of error interface
	err := server.RouterMissingParamError{
		ParamName: "paramName",
	}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestRouterParsingError checks the method Error() for data structure
// RouterParsingError
func TestRouterParsingError(t *testing.T) {
	// expected error value
	const expected = "Error during parsing param 'paramName' with value 'paramValue'. Error: 'errorMessage'"

	// construct an instance of error interface
	err := server.RouterParsingError{
		ParamName:  "paramName",
		ParamValue: "paramValue",
		ErrString:  "errorMessage"}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestAuthenticationError checks the method Error() for data structure
// AuthenticationError
func TestAuthenticationError(t *testing.T) {
	// expected error value
	const expected = "errorMessage"

	// construct an instance of error interface
	err := server.AuthenticationError{
		ErrString: "errorMessage"}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestNoBodyError checks the method Error() for data structure
// NoBodyError
func TestNoBodyError(t *testing.T) {
	// expected error value
	const expected = "client didn't provide request body"

	// construct an instance of error interface
	err := server.NoBodyError{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestBadBodyContent checks the method Error() for data structure
// BadBodyContent
func TestBadBodyContent(t *testing.T) {
	// expected error value
	const expected = "client didn't provide a valid request body"

	// construct an instance of error interface
	err := server.BadBodyContent{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestContentServiceUnavailableError checks the method Error() for data structure
// ContentServiceUnavailableError
func TestContentServiceUnavailableError(t *testing.T) {
	// expected error value
	const expected = "Content service is unreachable"

	// construct an instance of error interface
	err := server.ContentServiceUnavailableError{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestAggregatorServiceUnavailableError checks the method Error() for data structure
// AggregatorServiceUnavailableError
func TestAggregatorServiceUnavailableError(t *testing.T) {
	// expected error value
	const expected = "Aggregator service is unreachable"

	// construct an instance of error interface
	err := server.AggregatorServiceUnavailableError{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestUpgradesDataEngServiceUnavailableError checks the method Error() for data structure
// UpgradesDataEngServiceUnavailableError
func TestUpgradesDataEngServiceUnavailableError(t *testing.T) {
	// expected error value
	const expected = "Upgrade Failure Prediction service is unreachable"

	// construct an instance of error interface
	err := server.UpgradesDataEngServiceUnavailableError{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestAMSAPIUnavailableError checks the method Error() for data structure
// AMSAPIUnavailableError
func TestAMSAPIUnavailableError(t *testing.T) {
	// expected error value
	const expected = "AMS API is unreachable"

	// construct an instance of error interface
	err := server.AMSAPIUnavailableError{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

// TestParamsParsingError checks the method Error() for data structure
// ParamsParsingError
func TestParamsParsingError(t *testing.T) {
	// expected error value
	const expected = "the parameters contains invalid characters and cannot be used"

	// construct an instance of error interface
	err := server.ParamsParsingError{}

	// check if error value is correct
	assert.Equal(t, err.Error(), expected)
}

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
	"encoding/json"

	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
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

// TestHandleServer error check the function HandleServerError defined in errors.go
func TestHandleServerError(t *testing.T) {
	// check the behaviour with all error types defined in this package
	testResponse(t, &server.RouterMissingParamError{}, http.StatusBadRequest)
	testResponse(t, &server.RouterParsingError{}, http.StatusBadRequest)
	testResponse(t, &auth.AuthenticationError{}, http.StatusForbidden)
	testResponse(t, &auth.AuthorizationError{}, http.StatusForbidden)
	testResponse(t, &server.NoBodyError{}, http.StatusBadRequest)
	testResponse(t, &server.BadBodyContent{}, http.StatusBadRequest)
	testResponse(t, &server.ContentServiceUnavailableError{}, http.StatusServiceUnavailable)
	testResponse(t, &server.AggregatorServiceUnavailableError{}, http.StatusServiceUnavailable)
	testResponse(t, &server.UpgradesDataEngServiceUnavailableError{}, http.StatusServiceUnavailable)
	testResponse(t, &server.AMSAPIUnavailableError{}, http.StatusServiceUnavailable)
	testResponse(t, &server.ParamsParsingError{}, http.StatusBadRequest)

	// also some errors from types package are handled
	testResponse(t, &types.ItemNotFoundError{}, http.StatusNotFound)
	testResponse(t, &types.NoContentError{}, http.StatusNoContent)

	// error can be nil
	testResponse(t, nil, http.StatusInternalServerError)

	// we need to retrieve json.UnmarshalTypeError
	// so let's try to unmarshal "foo" string into an integer
	var x int
	err := json.Unmarshal([]byte("\"foo\""), &x)

	/// test with json.UnmarshalTypeError
	testResponse(t, err, http.StatusBadRequest)
}

// testResponse function uses HTTP server mock to check server response
// handlers
func testResponse(t *testing.T, e error, expectedCode int) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.HandleServerError(w, e)
	}))
	defer testServer.Close()

	res, err := http.Get(testServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != expectedCode {
		t.Errorf("Expected status code %v but got %v", expectedCode, res.StatusCode)
	}
}

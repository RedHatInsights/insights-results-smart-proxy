// Copyright 2020 Red Hat, Inc
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

func TestMakeURLToEndpointWithValidValue(t *testing.T) {
	apiPrefix := "api/v1/"
	endpoint := "some_valid_endpoint"

	retval := server.MakeURLToEndpoint(apiPrefix, endpoint)

	assert.Equal(t, retval, "api/v1/some_valid_endpoint")
}

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

package helpers

import "github.com/RedHatInsights/insights-operator-utils/tests/helpers"

// FailOnError fails on error
var FailOnError = helpers.FailOnError

// ToJSONString converts anything to json string
var ToJSONString = helpers.ToJSONString

// RunTestWithTimeout runs test with timeToRun timeout and fails if it wasn't in time
var RunTestWithTimeout = helpers.RunTestWithTimeout

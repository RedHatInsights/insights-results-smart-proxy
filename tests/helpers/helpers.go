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

// Package helpers contains constants, variables, functions, and structures
// used in unit tests. At this moment, just basic HTTP/REST API-based unit
// tests needs such helpers.
package helpers

import "github.com/RedHatInsights/insights-operator-utils/tests/helpers"

// FailOnError function fails on any error detected in tests
var FailOnError = helpers.FailOnError

// ToJSONString function converts any value or data structure to JSON string
var ToJSONString = helpers.ToJSONString

// RunTestWithTimeout function runs test with specified timeToRun timeout and
// fails if it wasn't finished in time
var RunTestWithTimeout = helpers.RunTestWithTimeout

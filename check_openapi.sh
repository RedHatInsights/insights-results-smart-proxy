#!/usr/bin/env bash
# Copyright 2020 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo "Testing OpenAPI specifications file"
# shellcheck disable=2181

# Check if Docker or Podman is available
if command -v docker &> /dev/null; then
    CONTAINER_RUNTIME="docker"
elif command -v podman &> /dev/null; then
    CONTAINER_RUNTIME="podman"
else
    echo "Neither Docker nor Podman is installed. Please install one of them."
    exit 1
fi

if $CONTAINER_RUNTIME run --rm -v "${PWD}":/local/:Z openapitools/openapi-generator-cli validate -i ./local/server/api/v1/openapi.json; then
    echo "OpenAPI spec file for API v1 is OK"
else
    echo "OpenAPI spec file for API v1 validation failed"
    exit 1
fi

if $CONTAINER_RUNTIME run --rm -v "${PWD}":/local/:Z openapitools/openapi-generator-cli validate -i ./local/server/api/v2/openapi.json; then
    echo "OpenAPI spec file for API v2 is OK"
else
    echo "OpenAPI spec file for API v2 validation failed"
    exit 1
fi


if $CONTAINER_RUNTIME run --rm -v "${PWD}":/local/:Z openapitools/openapi-generator-cli validate -i ./local/server/api/dbg/openapi.json; then
    echo "OpenAPI [DEBUG] spec file is OK"
else
    echo "OpenAPI [DEBUG] spec file validation failed"
    exit 1
fi

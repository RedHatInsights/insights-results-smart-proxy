#!/bin/bash
# Copyright 2022 Red Hat, Inc
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

set -exv


# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
APP_NAME="ccx-data-pipeline"  # name of app-sre "application" folder this component lives in
REF_ENV="insights-production"
COMPONENT_NAME="ccx-smart-proxy"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IMAGE="quay.io/cloudservices/ccx-smart-proxy"
COMPONENTS="ccx-data-pipeline ccx-insights-results dvo-extractor dvo-writer insights-content-service ccx-smart-proxy ocp-advisor-frontend ccx-mock-ams valkey-writer ccx-upgrades-rhobs-mock" # space-separated list of components to laod
COMPONENTS_W_RESOURCES="ccx-smart-proxy"  # component to keep
CACHE_FROM_LATEST_IMAGE="true"
DEPLOY_FRONTENDS="true"   # enable for front-end/UI tests
# Set the correct images for pull requests.
# pr_check in pull requests still uses the old cloudservices images
EXTRA_DEPLOY_ARGS="--set-parameter ccx-smart-proxy/IMAGE=quay.io/cloudservices/ccx-smart-proxy"

export IQE_PLUGINS="ccx"
# Run all pipeline and ui tests
export IQE_MARKER_EXPRESSION="pipeline or (smoke and ui)"
export IQE_FILTER_EXPRESSION=""
export IQE_REQUIREMENTS_PRIORITY=""
export IQE_TEST_IMPORTANCE=""
export IQE_CJI_TIMEOUT="30m"
export IQE_SELENIUM="true"  # Required for UI tests
export IQE_ENV="ephemeral"

# NOTE: Uncomment to skip pull request integration tests and comment out
#       the rest of the file.
# mkdir artifacts
# echo '<?xml version="1.0" encoding="utf-8"?><testsuites><testsuite name="pytest" errors="0" failures="0" skipped="0" tests="1" time="0.014" timestamp="2021-05-13T07:54:11.934144" hostname="thinkpad-t480s"><testcase classname="test" name="test_stub" time="0.000" /></testsuite></testsuites>' > artifacts/junit-stub.xml


function build_image() {
   source $CICD_ROOT/build.sh
}

function deploy_ephemeral() {
   source $CICD_ROOT/deploy_ephemeral_env.sh
}

function run_smoke_tests() {
   # component name needs to be re-export to match ClowdApp name (as bonfire requires for this)
   export COMPONENT_NAME="ccx-smart-proxy"
   source $CICD_ROOT/cji_smoke_test.sh
   source $CICD_ROOT/post_test_results.sh  # publish results in Ibutsu
}


# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh
echo "creating PR image"
build_image

echo "deploying to ephemeral"
deploy_ephemeral

echo "PR smoke tests disabled"
run_smoke_tests


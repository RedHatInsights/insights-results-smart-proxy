# insights-results-smart-proxy

[![forthebadge made-with-go](https://github.com/BraveUX/for-the-badge/blob/master/src/images/badges/made-with-go.svg)](https://go.dev/)

[![GoDoc](https://godoc.org/github.com/RedHatInsights/insights-results-smart-proxy?status.svg)](https://godoc.org/github.com/RedHatInsights/insights-results-smart-proxy)
[![GitHub Pages](https://img.shields.io/badge/%20-GitHub%20Pages-informational)](https://redhatinsights.github.io/insights-results-smart-proxy/)
[![Go Report Card](https://goreportcard.com/badge/github.com/RedHatInsights/insights-results-smart-proxy)](https://goreportcard.com/report/github.com/RedHatInsights/insights-results-smart-proxy)
[![Build Status](https://ci.ext.devshift.net/buildStatus/icon?job=RedHatInsights-insights-results-smart-proxy-gh-build-master)](https://ci.ext.devshift.net/job/RedHatInsights-insights-results-smart-proxy-gh-build-master/)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/RedHatInsights/insights-results-smart-proxy)
[![License](https://img.shields.io/badge/license-Apache-blue)](https://github.com/RedHatInsights/insights-results-smart-proxy/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/RedHatInsights/insights-results-smart-proxy/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/insights-results-smart-proxy)

Smart proxy for insights results

<!-- vim-markdown-toc GFM -->

* [Description](#description)
* [Documentation](#documentation)
* [BDD tests](#bdd-tests)
* [Makefile targets](#makefile-targets)
* [Contribution](#contribution)

<!-- vim-markdown-toc -->

## Description

Insights Results Smart Proxy is a service that acts as a proxy between the different external
data pipeline clients and the different services providing the required information.

It provides access to the [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
and to the [Insights Content Service](https://github.com/RedHatInsights/insights-content-service),
providing the clients with different endpoints for accesing both report results and rule content metadata
from a single service.

## Documentation

Documentation is hosted on Github Pages <https://redhatinsights.github.io/insights-results-smart-proxy/>.
Sources are located in [docs](https://github.com/RedHatInsights/insights-results-smart-proxy/tree/master/docs).

## BDD tests

Behaviour tests for this service are included in [Insights Behavioral
Spec](https://github.com/RedHatInsights/insights-behavioral-spec) repository.
In order to run these tests, the following steps need to be made:

1. clone the [Insights Behavioral Spec](https://github.com/RedHatInsights/insights-behavioral-spec) repository
1. go into the cloned subdirectory `insights-behavioral-spec`
1. run the `smart_proxy_tests.sh` from this subdirectory

List of all test scenarios prepared for this service is available at
<https://redhatinsights.github.io/insights-behavioral-spec/feature_list.html#smart-proxy>


## Makefile targets

```
Usage: make <OPTIONS> ... <TARGETS>

Available targets are:

clean                Run go clean
build                Build binary containing service executable
build-cover          Build binary with code coverage detection support
fmt                  Run go fmt -w for all sources
lint                 Run golint
vet                  Run go vet. Report likely mistakes in source code
cyclo                Run gocyclo
ineffassign          Run ineffassign checker
shellcheck           Run shellcheck
errcheck             Run errcheck
goconst              Run goconst checker
gosec                Run gosec checker
abcgo                Run ABC metrics checker
style                Run all the formatting related commands (fmt, vet, lint, cyclo) + check shell scripts
run                  Build the project and executes the binary
test                 Run the unit tests
help                 Show this help screen
```

## Contribution

Please look into document [CONTRIBUTING.md](CONTRIBUTING.md) that contains all information about how to
contribute to this project.

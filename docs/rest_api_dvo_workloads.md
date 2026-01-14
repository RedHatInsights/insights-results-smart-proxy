---
layout: page
nav_order: 22
---

# REST API endpoints for DVO Workloads Recommendations

## Background

To satisfy the requirements of the DVO Workloads cross-team effort, we introduced new REST API endpoints utilizing
data retrieved from new DVO-related database retrieved via insights-results-aggregator. These endpoints provide new 
data to 2 views/pages on console.redhat.com/openshift.

UI Mocks are available at [sketch](https://www.sketch.com/s/46f6d8e3-a4d0-4249-9d57-e6a79b518a6d/p/CCF338F7-6FBF-49F2-B841-B468D4DE40B5/canvas)

## Implementation

1. `v2/namespaces/dvo` - Endpoint to retrieve the list of all namespaces and aggregate DVO recommendation hits for each namespace.
1. `v2/namespaces/dvo/{namespace}/cluster/{cluster}` - Endpoint to retrieve DVO recommendation hits for a specific namespace + cluster combination with recommendation description and resolution taken from rule content provided by the content-service. It also provides a list of `object` for each DVO recommendation. 

## New REST API endpoints specification

REST API endpoints are described as OpenAPI that is available at [https://developers.redhat.com/api-catalog/api/insights-results-aggregator_v2](https://developers.redhat.com/api-catalog/api/insights-results-aggregator_v2)

Please note that in order to access REST API, authorization token needs to be
provided for most REST API endpoints (OpenAPI schema is the exception).

---
layout: page
nav_order: 21
---

# Interface with Insights Operator: on demand data gathering

## Background

On Demand Data Gathering feature allows customers to start data gathering (done using Insights Operator) at any time. This even allows starting multiple gatherings at the same time or during other gatherer(s) work in progress. In the end a bunch of IO raw archives will be sent into the External Data Pipeline (for example 5 results within one hour or even more - it’s out of our control).

## Implementation

1. Multiple rule hits (but in simplified format!) are now remembered within External Data Pipeline
1. These rule hits will are available to Insights Operator (and other clients) via new REST API endpoints
1. Just rule hits for last 24 hours need to be stored/remembered by External Data Pipeline
1. After 24 hours period it would be ok-ish to clean up these old records

## New REST API endpoints specification

REST API endpoints are described as OpenAPI that is available at [https://developers.redhat.com/api-catalog/api/insights-results-aggregator_v2](https://developers.redhat.com/api-catalog/api/insights-results-aggregator_v2)

Please note that in order to access REST API, authorization token needs to be
provided for most REST API endpoints (OpenAPI schema is the exception).

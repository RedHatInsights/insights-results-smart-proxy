---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `request-to-smart-proxy`



## Type

* REST API request



## Description

The API is currently split into two versions, see the [corresponding
README](https://github.com/RedHatInsights/insights-results-smart-proxy/blob/master/server/api/README.md)
for more info. *Smart Proxy* service provides information about its REST API
schema via endpoint `api/v1/openapi.json` and `api/v2/openapi.json`
respectively. OpenAPI 3.0 is used to describe the schema; it can be read by
human and consumed by computers.

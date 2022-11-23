---
layout: page
nav_order: 3
---
# REST API

The API is currently split into two versions, see the [corresponding README](https://github.com/RedHatInsights/insights-results-smart-proxy/blob/master/server/api/README.md) for more info.
Smart Proxy service provides information about its REST API schema via endpoint
`api/v1/openapi.json` and `api/v2/openapi.json` respectively.
OpenAPI 3.0 is used to describe the schema; it can be read by human and consumed
by computers.

For example, if Smart Proxy is started locally, it is possible to read schema
based on OpenAPI 3.0 specification by using the following command:

```shell
curl localhost:8080/api/v1/openapi.json
curl localhost:8080/api/v2/openapi.json
```

Please note that OpenAPI schema is accessible w/o the need to provide
authorization tokens, so it can be used to perform liveness/readiness probes.

## Authorization tokens

In order to access REST API authorization token needs to be provided for most
REST API endpoints (OpenAPI schema is the exception). Proper REST API calls
might look like:

```
curl -H 'Accept: application/json' -H "Authorization: Bearer ${ACCESS_TOKEN}" https://cloud.redhat.com/api/insights-results-aggregator/v1/org_overview | jq .

curl -H 'Accept: application/json' -H "Authorization: Bearer ${ACCESS_TOKEN}" https://cloud.redhat.com/api/insights-results-aggregator/v1/organizations/13454947/clusters | jq .

curl -H 'Accept: application/json' -H "Authorization: Bearer ${ACCESS_TOKEN}" https://cloud.redhat.com/api/insights-results-aggregator/v1/clusters/01234567-89ab-cdef-aaa7-dc6434af42d5/report | jq .
```

### Retrieving `ACCESS_TOKEN`

`ACCESS_TOKEN` can be retrieved from `OFFLINE_TOKEN` provided to user. Details
are explained in internal documentation.


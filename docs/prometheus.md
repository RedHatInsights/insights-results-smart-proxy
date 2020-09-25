---
layout: page
nav_order: 5
---
# Prometheus API

It is possible to use `/metrics` REST API endpoint to read all metrics exposed
to Prometheus or to any tool that is compatible with it.

## API related metrics 

There are a set of metrics provieded by `insights-operator-utils` library, all
of them related with the API usage. These are the API metrics exposed:

1. `api_endpoints_requests` the total number of requests per endpoint
1. `api_endpoints_response_time` API endpoints response time
1. `api_endpoints_status_codes` a counter of the HTTP status code responses
   returned back by the service
   
Additionally it is possible to consume all metrics provided by Go runtime. There
metrics start with `go_` and `process_` prefixes.

## Metrics namespace

As explained in the [configuration](./configuration) section of this
documentation, a namespace can be provided in order to act as a prefix to the
metric name. If no namespace is provided in the configuration, the metrics will
be exposed as described in this documentation.

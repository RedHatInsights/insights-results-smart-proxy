# insights-results-smart-proxy

Smart proxy for insights results

## Description

Insights Results Smart Proxy is a service that acts as a proxy between the different external
data pipeline clients and the different services providing the required information.

It provides access to the [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
and to the [Insights Content Service](https://github.com/RedHatInsights/insights-content-service),
providing the clients with different endpoints for accesing both report results and rule content metadata
from a single service.

## Architecture

![external-data-pipeline-arch](docs/Smart%20proxy%20architecture.png "External Data Pipeline Architecture")

## Results Smart Proxy in the external data pipeline

TODO

## Configuration

The configuration of the service is done by toml config, default one is `config.toml` in working directory,
but it can be overwritten by `INSIGHTS_RESULTS_SMART_PROXY_CONFIG_FILE` env var.

Also each key in config can be overwritten by corresponding env var. For example if you have config

```toml
[server]
address = ":8080"
auth = false
```

and environment variables

```shell
INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS=":443"
INSIGHTS_RESULTS_SMART_PROXY__SERVER__AUTH="true"
```

the actual server will listen in port 443 instead of 8080 and it will be TLS enabled

## Server configuration

Server configuration is in section `[server]` in config file

```toml
[server]
address = ":8080"
api_prefix = "/api/v1/"
api_spec_file = "openapi.json"
debug = true
auth = true
auth_type = "xrh"
use_https = true
enable_cors = true
```

* `address` is host and port which server should listen to
* `api_prefix` is prefix for RestAPI path
* `api_spec_file` is the location of a required OpenAPI specifications file
* `debug` is developer mode that enables some special API endpoints not used on production. In
production, `false` is used every time.
* `auth` turns on or turns authentication. Please note that this option can be set to `false` only
in devel environment. In production, `true` is used every time.
* `auth_type` set type of auth, it means which header to use for auth `x-rh-identity` or
`Authorization`. Can be used only with `auth = true`. Possible options: `jwt`, `xrh`
* `use_https` is option to turn on TLS server. Please note that this option can be set to `false`
only in devel environment. In production, `true` is used every time.
* `enable_cors` is option to turn on CORS header, that allows to connect from different hosts
(**don't use it in production**)

Please note that if `auth` configuration option is turned off, not all REST API endpoints will be
usable. Whole REST API schema is satisfied only for `auth = true`.

## Service base endpoints configuration

Service base endpoints configuration is in section `[services]` in config file

```toml
[services]
aggregator = "http://aggregator.service:8080/api/v1
content = "http://content.service:8080/api/v1"
```

* `aggregator` is the base endpoint URL for the Aggregator service where the Smart Proxy will connect and
retrieve the requested reports.
* `content` is the base endpoint URL for the Content service. Smart Proxy will retrieve and cache the
remmediations static content and the configured groups from its endpoints.

## REST API schema based on OpenAPI 3.0

TODO

## Contribution

Please look into document [CONTRIBUTING.md](CONTRIBUTING.md) that contains all information about how to
contribute to this project.

## Testing

TODO

## CI

TODO


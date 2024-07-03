---
layout: page
nav_order: 2
---

# Configuration
{: .no_toc }

## Table of contents
{: .no_toc }

1. TOC
{:toc}

Configuration is done by `toml` config, taking the `config.toml` in the working
directory if no configuration is provided. This can be overriden by
`INSIGHTS_RESULTS_SMART_PROXY_CONFIG_FILE` environment variable.

## Server configuration

Server configuration is in section `[server]` in config file.
The API is currently split into two versions, see the [corresponding README](https://github.com/RedHatInsights/insights-results-smart-proxy/blob/master/server/api/README.md)

```toml
[server]
address = ":8080"
api_v1_prefix = "/api/v1/"
api_v2_prefix = "/api/v2/"
api_v1_spec_file = "server/api/v1/openapi.json"
api_v2_spec_file = "server/api/v2/openapi.json"
debug = true
auth = true
auth_type = "xrh"
use_https = false
enable_cors = false
enable_internal_rules_organizations = false
internal_rules_organizations = []
log_auth_token = true
```

* `address` is host and port which server should listen to
* `api_v1_prefix` is prefix for the REST API V1
* `api_v2_prefix` is prefix for the REST API V2
* `api_v1_spec_file` is the location of a required OpenAPI specifications file for API V1
* `api_v2_spec_file` is the location of a required OpenAPI specifications file for API V2
* `debug` is developer mode that enables some special API endpoints not used on production. In
production, `false` is used every time.
* `auth` turns on or turns authentication. Please note that this option can be set to `false` only
in devel environment. In production, `true` is used every time.
* `auth_type` set type of auth. Can be used only with `auth = true`. Auth type used in all envs: `xrh`
* `use_https` enable or disable the usage of SSL transport for the HTTP server
* `enable_cors` enable or disable the [CORS
  headers](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
* `enable_internal_rules_organizations` allows enabling the access to the static
  content for internal rules for configured organizations (by `OrgID`)
* `internal_rules_organizations` defines the list of organizations who can
  access to the internal rules content
* `log_auth_token` enable or disable logging about the auth token used for
  identify the user performing requests to this service

Please note that if `auth` configuration option is turned off, not all REST API endpoints will be
usable. Whole REST API schema is satisfied only for `auth = true`.

## Services configuration

Services configuration is in section `[services]` in the configuration file.

```toml
[services]
aggregator = "http://localhost:8080/api/v1/"
content = "http://localhost:8082/api/v1/"
upgrade_risks_prediction = "http://localhost:8083/"
groups_poll_time = "60s"
```

* `aggregator` is the base endpoint to the Insights Results Aggregator service
  to be used
* `content` is the base endpoint to the Insights Content Service to be used
* `upgrade_risks_prediction` is the base endpoint to the Data Engineering Service,
  which is the one that will return the upgrade risks prediction results.
* `groups_poll_time` is the time between polls to the content service to
  retrieve updated static content, like groups or rule contents
  
The `groups_poll_time` must be configured as an string that can be parsed by the
function [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) from
Golang standard library.

## AMS client configuration

Smart Proxy is able to retrieve organizations information from the
[AMS API](https://api.openshift.com/?urls.primaryName=Accounts%20management%20service).
In order to do that, it needs to be configured with valid API URL and credentials.

```toml
[amsclient]
client_id = "Red Hat SSO client ID"
client_secret = "Corresponding client secret"
token = "a valid token"
url = "https://api.openshift.com"
page_size = 100
cluster_list_caching = "false"
```

* `client_id` and `client_secret` are optionals, but if any of them is defined, the other one should be
  defined too. They indicate the pair of credentials used by the client to connect to the API
* `token` is optional. If defined, the client will use that offline token to retrieve valid credentials in
  order to connect to the AMS API
* `url` indicates the base URL for the AMS API
* `page_size` is optional and defaults to 100. Defines the size of every page of results from the API
* `cluster_list_caching` is used to toggle cluster list caching from AMS in Redis

In order to use the AMS API, the client needs some of the credentials defined above. If both
`client_id`/`client_secret` and `token` are defined at the same time, `client_id`/`client_secret` pair
takes precedence over `token`.

## Setup configuration

TBD

## Metrics configuration

Metrics configuration is in section `[metrics]` in config file

```toml
[metrics]
namespace = "mynamespace"
```

* `namespace` if defined, it is used as `Namespace` argument when creating all
  the Prometheus metrics exposed by this service.

## Usage of environment variables

In order to avoid using a configuration file or to override some of the
configured values in it, the environment variables can be used.

Every configuration explained before in this document can be overriden by its
corresponding environment variable.

For example, if you have a configuration that includes the following:

```toml
[server]
address = ":8080"
auth = false
```

and you want to override the address for the HTTP server you can export
`INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS` variable, and its value will
override the `toml` file one.

For example:

```shell
INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS=":443"
INSIGHTS_RESULTS_SMART_PROXY__SERVER__AUTH="true"
```

will result on the server listens on port 443 and use TLS transport.

All the environment variables must have the `INSIGHTS_RESULTS_SMART_PROXY`
preffix, followed by the section name and the configuration paramater, both in
upper case. The characters `__` should be used as separater between the preffix,
the section name and the configuration parameter in each variable name.




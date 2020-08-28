---
layout: default
---
# Description

Insights Results Smart Proxy is a service that acts as a proxy between the different external
data pipeline clients and the different services providing the required information.

## Documentation for source files from this repository

* [smart_proxy.go](packages/smart_proxy.html)
* [smart_proxy_test.go](packages/smart_proxy_test.html)
* [version.go](packages/version.html)
* [server/rules.go](packages/server/rules.html)
* [server/router_utils_test.go](packages/server/router_utils_test.html)
* [server/export_test.go](packages/server/export_test.html)
* [server/endpoints.go](packages/server/endpoints.html)
* [server/handlers.go](packages/server/handlers.html)
* [server/server.go](packages/server/server.html)
* [server/auth.go](packages/server/auth.html)
* [server/router_utils.go](packages/server/router_utils.html)
* [server/server_test.go](packages/server/server_test.html)
* [server/endpoints_test.go](packages/server/endpoints_test.html)
* [server/configuration.go](packages/server/configuration.html)
* [server/errors.go](packages/server/errors.html)
* [conf/export_test.go](packages/conf/export_test.html)
* [conf/configuration_test.go](packages/conf/configuration_test.html)
* [conf/configuration.go](packages/conf/configuration.html)
* [export_test.go](packages/export_test.html)
* [services/configuration.go](packages/services/configuration.html)
* [services/services.go](packages/services/services.html)

## Architecture diagrams

### Sequence diagrams

* [CCX Data pipeline sequence diagram](data_pipeline_seq_diagram.png)

### Interface between CCX data pipeline and OCP WebConsole

* [IO pulling data from CCX data pipeline](io-pulling-only.png)
* [IO exposing data via CRD](io-pulling.png)
* [IO exposing data via Prometheus metrics](io-pulling-prometheus-metrics.png)
* [IO exposing data via Prometheus API](io-pulling-prometheus.png)

#### Animated versions of above diagrams

* [Animation: IO pulling data from CCX data pipeline](io-pulling-only.gif)
* [Animation: IO exposing data via Prometheus metrics](io-pulling-prometheus-anim.gif)

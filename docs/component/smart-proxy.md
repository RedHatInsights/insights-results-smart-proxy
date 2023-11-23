---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Smart Proxy`



## Type

* Service



## Description

Insights Results Smart Proxy is a service that acts as a proxy between the different external
data pipeline clients and the different services providing the required information.

It provides access to the [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
and to the [Insights Content Service](https://github.com/RedHatInsights/insights-content-service),
providing the clients with different endpoints for accesing both report results and rule content metadata
from a single service.


## Interfaces

* Input:
    - Insights Results Aggregator
    - Content Service
    - AMS
    - Redis
* Output:
    - REST API interface available for external access



## Grafana dashboard

* [https://grafana.app-sre.devshift.net/d/5RvvwGqW0/ccx-smart-proxy?orgId=1&refresh=30m](https://grafana.app-sre.devshift.net/d/5RvvwGqW0/ccx-smart-proxy?orgId=1&refresh=30m)



## Source code

* Repository: [https://github.com/RedHatInsights/insights-results-smart-proxy](https://github.com/RedHatInsights/insights-results-smart-proxy)
* Written in: Go

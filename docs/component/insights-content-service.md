---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Insights Content Service`



## Type

* Service



## Description

Insights Content Service is a service that provides metadata information about rules that are being
consumed by Openshift Cluster Manager. That metadata information contains rule title, description,
remmediations, tags and also groups, that will be consumed primarily by
[Insights Results Smart Proxy](https://github.com/RedHatInsights/insights-results-smart-proxy).



## Interfaces

* Input:
    - rule content read from repository/image
* Output:
    - rule content available via REST API



## Grafana dashboard

* [https://grafana.app-sre.devshift.net/d/JkLN3tvVk/ccx-insights-content-service?orgId=1&refresh=30m](https://grafana.app-sre.devshift.net/d/JkLN3tvVk/ccx-insights-content-service?orgId=1&refresh=30m)



## Source code

* Repository: [https://github.com/RedHatInsights/content-service/](https://github.com/RedHatInsights/content-service/)
* Written in: Go

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



## Source code

* Repository: [https://github.com/RedHatInsights/insights-content-service/](https://github.com/RedHatInsights/insights-content-service/)
* Written in: Go

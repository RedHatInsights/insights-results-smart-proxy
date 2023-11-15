---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `DVO Extractor`



## Type

* Service (filter) 



## Description

*DVO Extractor* service intends to retrieve Insights gathered archives, run rules
engine, and export DVO recommendations from generated JSON.
This service is based on *Insights Core Messaging* framework.


## Interfaces

* Input:
    - Messages with rule recommendationd + DVO recommendations
* Output:
    - JSON containing DVO recommendations + other metadata



## Grafana dashboard

* N/A



## Source code

* Repository: [https://github.com/RedHatInsights/dvo-extractor](https://github.com/RedHatInsights/dvo-extractor)
* Written in: Python


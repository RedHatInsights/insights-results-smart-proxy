---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `SHA Image Extractor`



## Type

* Service (filter) 



## Description

*Insights SHA Extractor* service intends to retrieve Insights gathered archives
and export SHAs of images found in the archive. SHAs of images are published
back in order to be consumed by other services. Insights SHA Extractor is based
on *Insights Core Messaging* framework.


## Interfaces

* Input:
    - Messages with rule reports + SHAs of images consumed from Kafka topic
* Output:
    - SHAs of images produced into Kafka topic



## Grafana dashboard

* [https://grafana.app-sre.devshift.net/d/shaextractor/ccx-insights-sha-extractor?orgId=1&refresh=30m](https://grafana.app-sre.devshift.net/d/shaextractor/ccx-insights-sha-extractor?orgId=1&refresh=30m)



## Source code

* Repository: [https://gitlab.cee.redhat.com/ccx/ccx-sha-extractor](https://gitlab.cee.redhat.com/ccx/ccx-sha-extractor)
* Written in: Python

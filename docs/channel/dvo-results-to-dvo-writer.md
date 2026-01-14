---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `dvo-results-to-dvo-writer`



## Type

* Kafka message



## Description

Messages consumed by *DVO Writer* from `ccx.ocp.dvo`.  That form of messages is
produced by *DVO Extractor* service.  Messages contain report with DVO
recommendations that is enhanced with more context information taken from
different sources, like the organization ID, account number, unique cluster
name, and the `LastChecked` timestamp.

Other relevant information about *DVO Extractor* can be found on address
[https://redhatinsights.github.io/dvo-extractor/](https://redhatinsights.github.io/dvo-extractor/).



## Schema

N/A


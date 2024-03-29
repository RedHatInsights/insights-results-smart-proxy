---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `rule-hits-to-db-aggregator`



## Type

* Kafka message



## Description

Messages consumed by *Insights Results DB Writer* from `ccx.ocp.results`.  That
form of messages is produced by *CCX Data Pipeline* service.  Messages contain
report with rule hits that is enhanced with more context information taken from
different sources, like the organization ID, account number, unique cluster
name, and the `LastChecked` timestamp.

Other relevant information about *CCX Data Pipeline* can be found on address
[https://redhatinsights.github.io/ccx-data-pipeline/](https://redhatinsights.github.io/ccx-data-pipeline/).



## Schema

[https://redhatinsights.github.io/insights-data-schemas/external-pipeline/ccx_data_pipeline.html](https://redhatinsights.github.io/insights-data-schemas/external-pipeline/ccx_data_pipeline.html)

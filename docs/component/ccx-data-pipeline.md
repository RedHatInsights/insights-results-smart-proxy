---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `CCX Data Pipeline`



## Type

* Service (filter)



## Description

*CCX Data Pipeline* service intends to get Insights gathered archives and
analyzes them using the Insights framework (OCP rules engine) in order to
generate a report with the rules hit by the content of the archive.

This report is published back in order to be consumed by other services.



## Interfaces

* Input:
    - Archives send by *Insights Operator*
* Output:
    - rule report in JSON format



## Grafana dashboard

* [https://grafana.app-sre.devshift.net/d/ccx-data-pipeline/ccx-data-pipeline?orgId=1&refresh=1m](https://grafana.app-sre.devshift.net/d/ccx-data-pipeline/ccx-data-pipeline?orgId=1&refresh=1m)



## Source code

* Repository: [https://gitlab.cee.redhat.com/ccx/ccx-data-pipeline](https://gitlab.cee.redhat.com/ccx/ccx-data-pipeline)
* Written in: Python

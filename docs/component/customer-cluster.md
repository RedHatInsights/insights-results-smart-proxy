---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Insights Operator`



## Type

* Kubernetes operator



## Description

*Insights Operator* installed on a cluster sends its data regularly into
*Ingress Service* (if enabled). This service stores that data into S3 bucket
for further processing. The retention policy of such data are two days.



## Interfaces

* Input:
    - Info taken from OpenShift cluster
* Output:
    - Tarball archive with log files etc. to be processed by OCP rule engine

## Source code

* Repository: [https://github.com/openshift/insights-operator](https://github.com/openshift/insights-operator)
* Written in: Go

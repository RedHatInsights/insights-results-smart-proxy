---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Data Engineering Service`



## Type

* Service



## Description

This service takes a list of cluster IDs from the user request and
uses the \[[Inference](urp-inference-service.html)\] service to generate
a set of upgrade risks predictions: whether the cluster would fail during
an upgrade or not. The data it sends to this service is collected from
\[[Thanos](thanos.html)\], enriched and parsed, adapting it for the inference.

## Interfaces

* Input:
    - Cluster IDs
    - Cluster metrics
* Output:
    - Predictions

## Grafana dashboard

* [CCX Upgrade Risks Predictions](https://grafana.stage.devshift.net/d/ccx-upgrade-risks-predictions/ccx-upgrade-risks-predictions?orgId=1)

## Source code

* Repository: [https://gitlab.cee.redhat.com/ccx/ccx-upgrades-data-eng](https://gitlab.cee.redhat.com/ccx/ccx-upgrades-data-eng)
* Written in: Python


---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Inference Service`



## Type

* Service



## Description

This service is in charge of running ML models based on a given input: whether
the cluster would fail during an upgrade or not.
It was separated from the \[[Data Engineering](urp-data-engineering-service.html)\] service
so that it can be scaled based on the traffic, the time and the resources it takes
to run these models.

## Interfaces

* Input:
    - Cluster metrics from Thanos enriched and parsed by the
    \[[Data Engineering](urp-data-engineering-service.html)\] service
* Output:
    - Predictions based on the ML model result

## Grafana dashboard

* [CCX Upgrade Risks Predictions](https://grafana.stage.devshift.net/d/ccx-upgrade-risks-predictions/ccx-upgrade-risks-predictions?orgId=1)

## Source code

* Repository: [https://gitlab.cee.redhat.com/ccx/ccx-upgrades-inference](https://gitlab.cee.redhat.com/ccx/ccx-upgrades-inference)
* Written in: Python


---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Thanos Cache`

## Type

* Part of the \[[Data Engineering](urp-data-engineering-service.html)\] code

## Description

This cache (currently an LRU cache with a TTL) stores the Inference results
for a given ammount of time. This way, if the user requests the same cluster
twice in less than `$TTL`, nor Thanos nor the inference service are used.

It can become a more complex cache (Redis f.e) in the future, but as the traffic
is quite low for now, we decided to use this simpler approach.

## Interfaces

* Input:
    - Cluster ID
* Output:
    - Predictions

## Source code

* Repository: [https://gitlab.cee.redhat.com/ccx/ccx-upgrades-data-eng](https://gitlab.cee.redhat.com/ccx/ccx-upgrades-data-eng)
* Written in: Python


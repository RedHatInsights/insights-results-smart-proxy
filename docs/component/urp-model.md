---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Upgrade risk prediction model`



## Type

* Piece of code



## Description

This is the ML model used by the \[[Inference](urp-inference-service.html)\] service.
It's currently a piece of code, but it could be a package or an external API in the
future.

## Interfaces

* Input:
    - Cluster metrics
* Output:
    - Predictions

## Source code

* Repository: [https://gitlab.cee.redhat.com/ccx/ccx-upgrades-inference](https://gitlab.cee.redhat.com/ccx/ccx-upgrades-inference)
* Written in: Python


---
layout: default
---
\[[front page](../overall-architecture.html)\] \[[overall architecture](../overall-architecture.html)\]



# channel: `cache-to-data-engineering-service`

## type

* Direct call from the code.

## description

The \[[Data Engineering](urp-data-engineering-service.html)\] service
checks if the cluster ID was requested previously. If that's the case,
it retrieves the prediction from this cache. Otherwise it will use
the \[[Inference](urp-inference-service.html)\] service.

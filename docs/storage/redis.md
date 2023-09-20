---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Storage: `redis`



## Type

* Key-value



# Engines used

* Redis (production)
* Redis (local deployment)
* Redis (local deployment)



## Description

This storage contains recommendations for all connected clusters provided by
OCP rules. Multiple recommendations are stored for one cluster. Retention
policy is set in order to clean up older records automatically.

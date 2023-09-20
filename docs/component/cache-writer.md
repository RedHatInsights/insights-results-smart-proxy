---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Cache writer`



## Type

* Service (writer)



## Description

The main task for this service is to listen to configured Kafka topic, consume
all messages from such topic, and write OCP results (in JSON format) with
additional information (like organization ID, cluster name, Kafka offset etc.)
into Redis. Multiple reports for one cluster can be stored there. The retention
policy is set for all reports so they can be cleaned up automatically by Redis
itself.



## Interfaces

* Input:
    - Messages with rule reports consumed from Kafka topic
* Output:
    - Rule reports stored into Redis database



## Grafana dashboard

* [https://grafana.app-sre.devshift.net/d/9mNv3nK7z/cache-writer?orgId=1&refresh=30m](https://grafana.app-sre.devshift.net/d/9mNv3nK7z/cache-writer?orgId=1&refresh=30m)



## Source code

* Repository: [https://github.com/RedHatInsights/insights-results-aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
* Written in: Go


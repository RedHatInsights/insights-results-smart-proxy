---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `DVO writer`



## Type

* Service



## Description

The main task for this service is to listen to configured Kafka topic, consume
all messages from such topic, and write DVO recommendations (in JSON format) with
additional information (like organization ID, cluster name, Kafka offset etc.)
into a SQL database. One report can be consumed and written into the database
for the same cluster.



## Interfaces

* Input:
    - Messages with rule reports consumed from Kafka topic
* Output:
    - DVO recommendations stored `dvo-report` table



## Grafana dashboard

* TODO



## Source code

* Repository: [https://github.com/RedHatInsights/insights-results-aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
* Written in: Go


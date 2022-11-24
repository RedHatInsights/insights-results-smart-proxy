---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `CCX Notification Writer`



## Type

* Service (writer)



## Description

The main task for this service is to listen to configured Kafka topic, consume
all messages from such topic, and write OCP results (in JSON format) with
additional information (like organization ID, cluster name, Kafka offset etc.)
into a database table named `new_reports`. Multiple reports can be consumed and
written into the database for the same cluster, because the primary (compound)
key for `new_reports` table is set to the combination `(org_id, cluster,
updated_at)`. When some message does not conform to expected schema (for
example if `org_id` is missing for any reason), such message is dropped and the
error message with all relevant information about the issue is stored into the
log. Messages are expected to contain `report` body represented as JSON.
This body is shrunk before it's stored into database so the database
remains relatively small.



## Interfaces

* Input:
    - Messages with rule reports consumed from Kafka topic
* Output:
    - Rule reports stored in `new_reports` table in SQL database



## Grafana dashboard

* [https://grafana.app-sre.devshift.net/d/9mNv3nK7z/ccx-notification-writer?orgId=1&refresh=30m](https://grafana.app-sre.devshift.net/d/9mNv3nK7z/ccx-notification-writer?orgId=1&refresh=30m)



## Source code

* Repository: [https://github.com/RedHatInsights/ccx-notification-writer](https://github.com/RedHatInsights/ccx-notification-writer)
* Written in: Go

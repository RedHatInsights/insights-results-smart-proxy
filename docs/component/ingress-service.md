---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Ingress`



## Type

* Service



## Description

*Ingress* is a component of cloud.redhat.com that allows for clients to upload
data to Red Hat. The service sites behind a *3Scale gateway* that handles
authentication, routing, and assignment of unique ID to the upload.

*Ingress* has an interface into cloud storage to retain customer data. It also
connects to a Kafka message queue in order to notify services of new and
available uploads for processing.



## Interfaces

* Input:
    - REST API endpoint(s) to upload data
* Output:
    - Kafka events

## Source code

* Repository: [https://github.com/RedHatInsights/insights-ingress-go](https://github.com/RedHatInsights/insights-ingress-go)
* Written in: Go

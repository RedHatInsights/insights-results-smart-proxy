---
layout: page
nav_order: 1
---

# Architecture

Smart proxy is the only external data pipeline service that is exposed directly
to clients through cloud.redhat.com and also through console.redhat.com using
indirection mechanism.

This service is able to provide an interface to other services as well as
composing responses aggregating data from different services.

As an example of the first one, the `groups` endpoint will act as a proxy to the
Content Service. But for example when asking for a cluster's latest report,
Smart Proxy retrieves the report from Aggregator service, and if some rules are
hit, then it takes the content for those rules from the Content Service.

Insights Results Smart Proxy has 3 main parts:

#. An Insights Results Aggregator client, that sends requests to the API in
order to retrieve reports or other relevant info for the given cluster.
#. An Insights Content Service client, that retrieve groups and static content
for rules values from that service.
#. A HTTP server that serve the current API, attending each request and using
the previous clients in order to get the information from the relevant services.

## Smart Proxy architecture

![external-data-pipeline-arch](Smart%20proxy%20architecture.png "External Data Pipeline Architecture")

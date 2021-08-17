---
layout: page
nav_order: 18
---

# Interface with OCM UI

Insights Advisor integration components utilize two different APIs:

1. Smart Proxy API (based on Insights Results Aggregator) and accessed via 3scale.

2. Account Management Service API (part of so-called OpenShift API).

## URL to cluster view in OCM UI

Two forms of URL are supported:

1. `https://console.redhat.com/openshift/details/s/${subscription_ID}#insights`
1. `https://console.redhat.com/openshift/details/${cluster_UUID}#insights

For example:

1. `https://console.redhat.com/openshift/details/s/1vrdKPiixrT4OxjPHNae0GSGXAj#insights`
2. `https://console.redhat.com/openshift/details/019bd8fb-a470-444f-9ad4-2e798972ee62#insights`

> **_NOTE:_**  Please note that the example cluster might not exists anymore.

---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `new-rule-hits-markdown`



## Type

* JSON data, two attributes containing markdown text



## Description

Newly found issues are sent to *Service Log* via REST API. Because *Service Log* accepts description and content to be represented in Markdown, issues are "rendered" first by [Insights Content Template Renderer](https://github.com/RedHatInsights/insights-content-template-renderer). To use the Service Log API, the `ccx-notification-service` uses the credentials stored in [vault](https://vault.devshift.net/ui/vault/secrets/insights/show/secrets/insights-prod/ccx-data-pipeline-prod/ccx-notification-service-auth).


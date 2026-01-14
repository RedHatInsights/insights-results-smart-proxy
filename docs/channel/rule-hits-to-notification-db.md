---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `rule-hits-to-aggregator-db`



## Type

* SQL `insert` statement



## Description

*CCX Notificaton Writer* service writes OCP results (in JSON format) with
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

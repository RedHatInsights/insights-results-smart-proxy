---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `rule-hits`



## Type

* Kafka message



## Description

That form of messages is produced by *CCX Data Pipeline* service, which
provides a _publisher_ class that can send the generated reports to selected
Kafka topic.  This class is named
`ccx_data_pipeline.kafka_publisher.KafkaPublisher` and its source code can be
found in the service repository (see the link below this paragraph). The report
generated by the framework are enhanced with more context information taken
from different sources, like the organization ID, account number, unique
cluster name, and the `LastChecked` timestamp (taken from the incoming Kafka
record containing the URL to the archive).

Other relevant information about *CCX Data Pipeline* can be found on address
[https://redhatinsights.github.io/ccx-data-pipeline/](https://redhatinsights.github.io/ccx-data-pipeline/).



## Schema

[https://redhatinsights.github.io/insights-data-schemas/external-pipeline/ccx_data_pipeline.html](https://redhatinsights.github.io/insights-data-schemas/external-pipeline/ccx_data_pipeline.html)

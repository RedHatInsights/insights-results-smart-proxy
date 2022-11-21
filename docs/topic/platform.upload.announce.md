---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Topic: `patform.upload.announce`



## Topic description

The records (messages) received from `platform.upload.announce` that are
identified by header. Messages to be consumed by CCX Data Pipeline contain
header `openshift` and are encoded using JSON format. Such messages are sent
for every new tarball stored in S3 Bucket and each message consists of an
object with various attributes described in schema.



## Messages format

* JSON



## Schema version

1 (unofficial)



## Schema description

[https://redhatinsights.github.io/insights-data-schemas/external-pipeline/platform_upload_announce_messages.html](https://redhatinsights.github.io/insights-data-schemas/external-pipeline/platform_upload_announce_messages.html)



## Additional information

N/A


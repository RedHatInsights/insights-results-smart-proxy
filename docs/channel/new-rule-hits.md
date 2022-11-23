---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Channel: `new-rule-hits`



## Type

* Kafka message



## Description

If changes (new issues) has been found by *CCX Notification Service*, notification message is sent into Kafka topic named `platform.notifications.ingress`. The expected format of the message can be found [here](https://core-platform-apps.pages.redhat.com/notifications-docs/dev/user-guide/send-notification.html#_kafka).


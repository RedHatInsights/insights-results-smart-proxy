---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Archive DB Writer`



## Type

* Service (writer)



## Description

The main task for this service is to listen to configured Kafka topic, consume
all messages from such topic, and write list of images SHAs into CVEs database.



## Interfaces

* Input:
    - Messages with list of images SHAs consumed from Kafka topic
* Output:
    - List of images SHAs in SQL database

## Source code

* Repository: [https://github.com/RedHatInsights/vuln4shift-backend](https://github.com/RedHatInsights/vuln4shift-backend)
* Written in: Go

---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Notification rules`



## Type

* Data source



## Description

Configuration parameters for *CCX Notification Service* in form of conditions
used to determine which messages will be sent to *Service Log* or to
*Notification Backend*. Additionally it is based on set of threshold values.
Currently it is part of *CCX Notification Service* config files + config
environment variables.

An example can be found [here](https://github.com/RedHatInsights/ccx-notification-service/blob/master/config.toml#L34)



## Interfaces

* Input:
    - N/A
* Output:
    - condition (expression)
    - set of threshold values

## Source code

* Repository: [https://github.com/RedHatInsights/ccx-notification-service](https://github.com/RedHatInsights/ccx-notification-service)
* Written in: TOML

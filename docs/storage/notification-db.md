---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Storage: `ccx-notification-db`



## Type

* Relational database



# Engines used

* RDS (production)
* PostgreSQL (local deployment)
* SQLite (local deployment)



## Description

Database that contains two main tables named `new_reports` and `reported`. The
first table is filled-in continuously by *CCX Notification Writer* service for
all new rule reports. Second table is modified by *CCX Notification Service*
during searching for new reports and after reports are send to selected
notification targets. For more info please look at page
[https://redhatinsights.github.io/ccx-notification-writer/data_flow.html](https://redhatinsights.github.io/ccx-notification-writer/data_flow.html).



## Storage schema and description

* [https://redhatinsights.github.io/ccx-notification-writer/db-description/](https://redhatinsights.github.io/ccx-notification-writer/db-description/)

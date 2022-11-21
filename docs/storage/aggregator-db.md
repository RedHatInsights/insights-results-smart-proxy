---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Storage: `insights-results-aggregator-db`



## Type

* Relational database



# Engines used

* RDS (production)
* PostgreSQL (local deployment)
* SQLite (local deployment)



## Description

This storage contains recommendations for all connected clusters provided by
OCP rules. Also feedback provided by users and users-defined rule enable and
disable flags are stored in this storage. Database migration is fully supported
too for this storage. For more info please look at
[https://redhatinsights.github.io/insights-results-aggregator/database.html](https://redhatinsights.github.io/insights-results-aggregator/database.html)



## Storage schema and description

* [https://redhatinsights.github.io/insights-results-aggregator/db-description/index.html](https://redhatinsights.github.io/insights-results-aggregator/db-description/index.html)

---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Storage: `dvo-db`



## Type

* Relational database



# Engines used

* RDS (production)
* PostgreSQL (local deployment)



## Description

This storage contains DVO recommendations for all connected clusters (with DVO
enabled) provided by OCP rules. Also feedback provided by users and
users-defined rule enable and disable flags are stored in this storage.
Database migration is fully supported too for this storage.



## Storage schema and description

* N/A


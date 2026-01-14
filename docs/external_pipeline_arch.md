---
layout: page
nav_order: 10
---

# External Pipeline architecture

## Sequence diagrams

* [CCX Data pipeline sequence diagram](images/data_pipeline_seq_diagram.png)
* [IO pulling data + exposing via CRD to OCP WebConsole](iimages/o_pulling_crd_seq_diagram.png)

# Interface between CCX data pipeline and OCP WebConsole

### Interface between CCX data pipeline and Insights Operator

* [IO pulling data from CCX data pipeline](images/io-pulling-only.png)

### Interface between Insights Operator and OCP WebConsole based on CRD

* [IO exposing data via CRD](images/io-pulling.png)
* [IO exposing data via CRD - including internal structure](images/io-pulling-crd-internal.png)

### Interface between Insights Operator and OCP WebConsole based on Prometheus or Prometheus metrics

* [IO exposing data via Prometheus metrics](images/io-pulling-prometheus-metrics.png)
* [IO exposing data via Prometheus metrics - including internal structure](images/io-pulling-prometheus-internal.png)
* [IO exposing data via Prometheus API](images/io-pulling-prometheus.png)

### Animated versions of above diagrams

* [Animation: IO pulling data from CCX data pipeline](images/io-pulling-only.gif)
* [Animation: IO exposing data via Prometheus metrics](images/io-pulling-prometheus-anim.gif)


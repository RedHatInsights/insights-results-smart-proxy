---
layout: page
nav_order: 2
---

# Architecture diagrams

# Sequence diagrams

* [CCX Data pipeline sequence diagram](data_pipeline_seq_diagram.png)
* [IO pulling data + exposing via CRD to OCP WebConsole](io_pulling_crd_seq_diagram.png)

# Interface between CCX data pipeline and OCP WebConsole

## Interface between CCX data pipeline and Insights Operator

* [IO pulling data from CCX data pipeline](io-pulling-only.png)

## Interface between Insights Operator and OCP WebConsole based on CRD

* [IO exposing data via CRD](io-pulling.png)
* [IO exposing data via CRD - including internal structure](io-pulling-crd-internal.png)

## Interface between Insights Operator and OCP WebConsole based on Prometheus or Prometheus metrics

* [IO exposing data via Prometheus metrics](io-pulling-prometheus-metrics.png)
* [IO exposing data via Prometheus metrics - including internal structure](io-pulling-prometheus-internal.png)
* [IO exposing data via Prometheus API](io-pulling-prometheus.png)

## Animated versions of above diagrams

* [Animation: IO pulling data from CCX data pipeline](io-pulling-only.gif)
* [Animation: IO exposing data via Prometheus metrics](io-pulling-prometheus-anim.gif)

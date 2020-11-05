---
layout: page
nav_order: 20
---

# Interface with ACM

## Step #1: request from ACM to Smart Proxy via 3Scale

![Request made with list of clusters](acm_interface_req_rep_1.png.png "Request made with list of clusters")

## Step #2: request handling by 3Scale

![3Scale adds rh-identity header](acm_interface_req_rep_2.png.png "3Scale adds rh-identity header")

## Step #3: processing request by Smart Proxy

![Smart proxy gather organization ID from rh-identity header](acm_interface_req_rep_3.png.png "Smart proxy gather organization ID from rh-identity header")

## Step #4: processing request by Insights Results Aggregator

![Request is made to Insights Results Aggregator](acm_interface_req_rep_4.png.png "Request is made into Insights Results Aggregator")

## Step #5: returning response to ACM

![Response with map of results is returned into ACM](acm_interface_req_rep_5.png.png "Response with map of results is returned into ACM")

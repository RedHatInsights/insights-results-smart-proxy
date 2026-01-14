---
layout: default
---
\[[Front page](../overall-architecture.html)\] \[[Overall architecture](../overall-architecture.html)\]



# Component: `Insights Content Template Renderer`



## Type

* Service



## Description

This service provides the endpoint for rendering the report messages based on
the `DoT.js` templates from content data and report details. For that purpose
it uses the implementation of DoT.js framework in Python.


## Interfaces

* Input:
    - Rule templates in Markdown format
    - Data to be used in the final document
* Output:
    - Markdown document based on rule template and provided data

## Source code

* Repository: [https://github.com/RedHatInsights/insights-content-template-renderer/](https://github.com/RedHatInsights/insights-content-template-renderer/)
* Written in: Python

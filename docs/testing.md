---
layout: page
nav_order: 6
---

# Testing
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

tl;dr: `make before-commit` will run most of the checks by magic, `VERBOSE=true
make before-commit` will do the same but print more information about what it's
doing.

The following tests can be run to test your code in
`insights-results-smart-proxy`. Detailed information about each type of test is
included in the corresponding subsection:

1. Unit tests: checks behavior of all units in source code (methods, functions)
1. REST API Tests: test the real REST API of locally deployed application
1. Metrics tests: test whether Prometheus metrics are exposed as expected

## Unit tests

Set of unit tests checks all units of source code. Additionally the code coverage is computed and
displayed. Code coverage is stored in a file `coverage.out` and can be checked by a script named
`check_coverage.sh`.

To run unit tests use the following command:

`make test`

## Check coverage

If you want to check the percentage of code reached by the unit tests, you can
use `./check_coverage.sh` script after running the unit tests.

It will check if the coverage of the code is bellow the threshold.

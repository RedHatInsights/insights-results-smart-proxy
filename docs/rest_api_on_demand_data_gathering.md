---
layout: page
nav_order: 21
---

# Interface with Insights Operator: on demand data gathering

## New REST API endpoints specification

Please note that in order to access REST API, authorization token needs to be
provided for most REST API endpoints (OpenAPI schema is the exception).



### Check status of given request-id

#### Endpoint

```
/cluster/{clusterID}/request/{requestID}/status
```

#### Method

GET

#### Query parameters

None

#### Request payload

None

#### Response for known resource

HTTP code 200 for all possible states

```
{
    “cluster”: “{clusterID}”,
    “requestID”: “{requestID}”,
    “status”: “{string}”
}
```

#### Response for unknown resource

* Not known cluster ID
* Not known request ID
    - HTTP code 404

#### Response in case of improper parameters (format etc.)

HTTP code 400



### List of all recorded requests for given cluster (GET variant)

#### Endpoint

```
/cluster/{clusterID}/requests/
```

#### Method

GET

#### Query parameters

None

#### Request payload

None

#### Response for known resource

HTTP code 200 for all recorded requests

```
{
    “cluster”: “{clusterID}”,
    “requests”: [{array}],
    “status”: “{string}”
}
```

Where {array} contains the following objects:

```
{
    “requestID”: {requestID},
    “valid: True,
    “received”: {timestamp},
    “processed”: {timestamp},
}
```

Status will contain “ok” at this moment.

#### Response for unknown resource

* Not known cluster ID
    - HTTP code 404

* Response in case of improper parameters (format etc.)
    - HTTP code 400



### List of all recorded requests for given cluster (POST variant)

#### Endpoint

```
/cluster/{clusterID}/requests/
```

#### Method

POST

#### Query parameters

None

#### Request payload

List of request IDs in format

```
[
    “requestID1”,
    “requestID2”,
    ….
    “requestID3”
]
```

#### Response for known resource


HTTP code 200 for all recorded requests

```
{
    “cluster”: “{clusterID}”
    “requests”: [{array}]
    “status”: “{string}”
}
```

Where {array} contains the following objects:

```
{
    “requestID”: {requestID},
    “valid”: true/false depends if this is valid/known request ID,
    “received”: {timestamp},
    “processed”: {timestamp},
}
```

Status will contain “ok” at this moment.

#### Response in case of improper parameters (format etc.)

HTTP code 400



### Retrieve simplified results for given cluster and request-id

#### Endpoint

```
/cluster/{clusterID}/request/{requestID}/report
```

#### Method

GET

#### Query parameters

None

#### Request payload

None

#### Response for known resource

HTTP code 200 for all possible states

```
{
    “cluster”: “{clusterID}”,
    “requestID”: “{requestID}”,
    “status”: “{string}”,
    “report”: “{simplifiedReportStructure}”,
}
```

Where simplifiedReportStructure might look like:

```
[
    “rule_fqdn”: “”,
    “error_key”: “”,
    “description”: “”,
    “total_risk”: “”,
]
```

#### Response for unknown resource

* Not known cluster ID
* Not known request ID
    - HTTP code 404

#### Response in case of improper parameters (format etc.)

HTTP code 400

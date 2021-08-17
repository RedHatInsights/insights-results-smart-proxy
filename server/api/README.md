The API is *temporarily* split into two versions, /api/v1/ and api/v2/ because of ongoing efforts to unify the behaviour, looks, API, etc with the Insights Advisor. Splitting the API into two allows us to continually deploy the work on our side, while allowing the UI developers to implement the new endpoints at their own pace, because the old endpoints will still work as we're not replacing them. This enables parallel work for both teams. Link to the ongoing epic: https://issues.redhat.com/browse/CCXDEV-4936.

### Structure

The following structure is currently handling v1/v2 endpoints for the reasons below:
    - API v1
        - `server/handlers_v1.go`
        - `server/endpoints_v1.go`
        - OpenAPI spec in `server/api/v1/openapi.json`
    - API v2 (endpoints unchanged from v1 should point to handlers in `server/handlers_v1.go`)
        - `server/handlers_v2.go`
        - `server/endpoints_v2.go`
        - OpenAPI spec in `server/api/v2/openapi.json`
    - test files are shared, as the handler methods have to have different names anyway 

To avoid unneccessary go.mod mess and to avoid going against the Go conventions regarding [code organization](https://blog.golang.org/organizing-go-code), it doesn't make sense from Go perspective to have `server/api/v1/endpoints.go` and `server/api/v2/endpoints.go`, as this creates more problems than it solves since Go treats any subdirectory as an individual package -- this can be avoided by including the subdir in the import string or by creating an `internal/` subdirectory in `server/`, but once again, any of these seemed much more messy to me than having `server/endpoints_v1.go` and `server/endpoints_v2.go`, especially since we will eventually have only one version.

### ToDo
After we create and rename all the new endpoints from the epic, we should link the remaining endpoints - which will remain unchanged - to the /api/v2 as well, simply pointing it to the "v1" handler, so that the consumers of the API don't have to keep track of both of them. At this point, we just need to ensure there are no consumers of the /api/v1 and we can just remove the v1 altogether.
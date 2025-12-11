# AGENTS.md

## Project Overview
insights-results-smart-proxy is a Go-based service for the Red Hat Insights ecosystem that acts as a unified gateway between external data pipeline clients and internal services. It aggregates and composes responses from multiple backend services including the [Insights Content Service](https://gitlab.cee.redhat.com/ccx/content-service) and [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator), providing a single API for clients to access cluster recommendations, rule content metadata, and other Insights data. The service is exposed directly to clients through console.redhat.com.

**Tech Stack**: Go 1.22+, Redis (caching), Prometheus, Sentry/Glitchtip, REST API, AMS (OCM Account Management Service), RBAC integration

## Repository Structure
```text
/server/                 - HTTP server, REST API handlers, and routing
  /api/                  - OpenAPI specifications (v1, v2)
  handlers_v1.go         - API v1 endpoint handlers
  handlers_v2.go         - API v2 endpoint handlers
  endpoints_v1.go        - API v1 endpoint definitions
  endpoints_v2.go        - API v2 endpoint definitions
  acks_handlers.go       - Rule acknowledgment handlers
  auth_middleware.go     - Authentication middleware
  metrics_middleware.go  - Prometheus metrics middleware
  upgrade_risks_prediction.go - Upgrade risks prediction handlers
  rating.go              - User rating functionality

/services/               - External service clients and integrations
  services.go            - Service interface definitions
  redis.go               - Redis caching service
  configuration.go       - Services configuration

/amsclient/              - AMS (OCM Account Management Service) client
/auth/                   - Authentication and RBAC logic
  auth.go                - Authentication implementation
  rbac.go                - RBAC permissions handling

/content/                - Content Service client integration
/types/                  - Type definitions and data structures
  types.go               - Core data types
  dvo_types.go           - DVO-specific types
  rbac.go                - RBAC types
  upgrade_risks_predictions.go - Upgrade prediction types
  operations.go          - Operation types

/metrics/                - Prometheus metrics definitions
/tests/                  - Test data and helpers
  /testdata/             - Test data fixtures
  /helpers/              - Test helper functions

/docs/                   - Documentation (GitHub Pages)
/deploy/                 - Deployment configurations
/dashboards/             - Grafana dashboards
/.tekton/                - Tekton CI/CD pipelines
/.github/                - GitHub Actions workflows

smart_proxy.go           - Main application entry point
config.toml              - Production configuration example
config-devel.toml        - Development configuration
Dockerfile               - Container image build script
Makefile                 - Build and development targets
```

## Development Workflow

### Setup
- **Go version**: 1.22+ required
- **Development**: Always work on a feature branch, never commit directly to master
- **Cache**: Redis (for cluster list caching)
- **External services**:
  - Insights Results Aggregator (backend) - `INSIGHTS_RESULTS_SMART_PROXY__SERVICES__AGGREGATOR`
  - Insights Content Service (backend) - `INSIGHTS_RESULTS_SMART_PROXY__SERVICES__CONTENT`
  - AMS/OCM API (cluster metadata) - `INSIGHTS_RESULTS_SMART_PROXY__AMSCLIENT__URL`
  - RBAC service (permissions) - `INSIGHTS_RESULTS_SMART_PROXY__RBAC__URL`

### Running Tests
- `make test` - Unit tests
- `make cover` - HTML coverage report
- `make coverage` - Coverage in the terminal
- `go test -v ./...` - Direct test execution

### Code Quality
- `make golangci-lint` - golangci-lint with auto-fix
- `make shellcheck` - Shell script linting
- `make abcgo` - ABC metrics checker
- `make openapi-check` - OpenAPI validation
- `make style` - Formatting and style checks (includes shellcheck, abcgo, golangci-lint)
- `make before-commit` - Pre-commit suite (style, tests, license, coverage, openapi-check)

### Building and Running
- `make build` - Build binary
- `make build-cover` - Build with coverage support
- `make run` - Build and execute

### CLI Commands
- `./insights-results-smart-proxy` - Start service

## Key Architectural Patterns

### Smart Proxy Architecture

This service acts as a **unified gateway and orchestrator**. The main flow is: receive client requests -> authenticate/authorize -> fetch from multiple services -> compose response -> return to client.

1. **Client Requests**: External clients (OCM, ACM, OCP Web Console) send requests with x-rh-identity header to Smart Proxy
2. **Authentication & Authorization**: Validate x-rh-identity token and check RBAC permissions
3. **Service Orchestration**: Smart Proxy routes requests to appropriate backend services:
   - Insights Results Aggregator (cluster reports and recommendations)
   - Insights Content Service (rule metadata and groups)
   - AMS/OCM API (cluster metadata and organization info)
   - RBAC service (user permissions)
4. **Response Composition**: Aggregates and enriches data from multiple services
5. **Caching**: Redis cache for cluster lists and frequently accessed data
6. **Metrics**: Prometheus tracks service health and performance

### API Versions

The service exposes two API versions:

- **API v1** (`/api/v1/`): Legacy endpoints for backward compatibility
- **API v2** (`/api/v2/`): Current API with enhanced features including:
  - Rule acknowledgments
  - User ratings
  - Upgrade risks predictions
  - DVO recommendations
  - Enhanced filtering and pagination

### Authentication & Authorization

- **Authentication types**:
  - `xrh`: x-rh-identity header (Red Hat Cloud identity)
  - Internal authentication for service-to-service communication
- **RBAC integration**: Validates user permissions via RBAC service
- **Organization filtering**: Ensures users only access data from their organization
- **AMS/OCM integration**: Validates cluster ownership and access rights

### Key Features

1. **Cluster Reports**: Aggregates cluster recommendations from Aggregator service with rule content from Content Service
2. **Rule Acknowledgments**: Users can acknowledge/disable specific rules for clusters
3. **User Ratings**: Feedback mechanism for rule quality
4. **Upgrade Risks Predictions**: Integration with [upgrade prediction service](https://gitlab.cee.redhat.com/ccx/ccx-upgrades-data-eng)
5. **Groups Management**: Rule groups and tags from Content Service
6. **DVO Recommendations**: Deployment Validation Operator recommendations
7. **Cluster List Caching**: Redis-based caching of AMS cluster lists for performance

### Configuration

- `config.toml` - Production config template
- `config-devel.toml` - Development config (for local testing)
- `deploy/clowdapp.yaml` - Deployment configuration with environment variables for production/stage/ephemeral
- Environment variables override config files
- Service URLs configurable per environment

## Working with this Repository

**As an agent, you should create a TODO list** when working on tasks to track progress and ensure all steps are completed systematically.

## Code Conventions

### Go Style
- **Linters**: golangci-lint (with auto-fix support)
- **Formatting**: gofmt
- **Documentation**: GoDoc comments required for exported symbols
- **Error handling**: Explicit returns, custom error types for domain-specific errors (e.g., `ContentServiceUnavailableError`, `AggregatorServiceUnavailableError`, `RouterMissingParamError`)

### Naming Patterns
- Test files: `*_test.go`
- HTTP handlers: `Handle*` or `*Handler` pattern
- Metrics: snake_case
- Config: lowercase_with_underscores in TOML
- Middleware: `*Middleware` pattern

### Error Handling
- Return errors explicitly, no panics in production
- Custom error types for domain errors
- Log with context (zerolog) and request IDs
- Never log organization IDs without sanitization

## Important Notes

### Dependencies
- **HTTP router**: gorilla/mux for routing
- **Redis client**: go-redis/v9 for caching
- **Logging**: rs/zerolog for structured logging
- **Metrics**: prometheus/client_golang
- **Common utilities**: redhatinsights/app-common-go for platform integration - a client access library for the config for the Clowder operator (Redis credentials)
- **OpenAPI**: Specification-first API design

### Testing
- Unit tests: standard Go testing
- Mock HTTP clients for external services
- Test data in `/tests/testdata/`
- Test helpers in `/tests/helpers/`

### Behavioral Tests
External BDD tests in [Insights Behavioral Spec](https://github.com/RedHatInsights/insights-behavioral-spec):
- `smart_proxy_tests.sh` - Smart Proxy behavioral tests

### Monitoring
- Prometheus metrics (configurable namespace: `smart_proxy`)
- Sentry for error tracking
- Health endpoints: Standard health check endpoints

## Pull Request Guidelines

### Before Creating a PR

Run `make before-commit` to verify tests, linting, coverage, license headers, and OpenAPI validation.

### PR Requirements
- **Reviews**: Minimum 2 approvals from maintainers
- **Commit messages**: Clear, descriptive messages explaining the "why" rather than the "what"
  - No specific convention required
  - If related to a Jira task, optionally include the ticket ID: `[CCXDEV-12345] Description`
- **Documentation**: Update docs for API or behavior changes
- **OpenAPI specs**: Update `/server/api/v1/openapi.json` or `/server/api/v2/openapi.json` for API changes
- **Base branch**: `master` (main development branch) - always create a feature branch for development, never commit directly to master
- **Breaking changes**: Must be documented and communicated
- **WIP PRs**: Tag with `[WIP]` prefix in title to prevent accidental merging

### PR Checklist

Before pushing changes, ensure:

- All tests pass (`make test`)
- Code passes linting (`make golangci-lint`)
- Coverage has not decreased (`make coverage`)
- OpenAPI spec is valid and updated (`make openapi-check`)
- License headers are present (`make license`)
- Documentation is updated (if API or behavior changes)
- No merge conflicts with master branch

## Deployment Information

- **Deployment configs**: Located in `/deploy` directory
- **Tekton pipelines**: CI/CD definitions in `/.tekton`
- **Container**: Built via Dockerfile, published to registry
- **Configuration management**: Via app-interface (internal Red Hat)
- **Environments**: Ephemeral (testing), stage, production
- **Dashboards**: Grafana dashboards in `/dashboards` for monitoring

## Security Considerations
- **Never log sensitive data**: Organization IDs should be sanitized, no auth tokens, no PII
- **Credentials**: Use environment variables or config files, never hardcode
- **API authentication**: Validate all incoming requests via x-rh-identity
- **Input validation**: Sanitize and validate all user inputs
- **RBAC enforcement**: Respect organization boundaries and user permissions
- **Redis security**: Secure connection to Redis, no sensitive data in cache keys

## Debugging Tips
- **Test coverage**: Generate HTML reports with `make cover` to identify untested code
- **OpenAPI validation**: Use `make openapi-check` to verify API spec consistency
- **Service integration**: Test with local instances of Aggregator and Content Service
- **Redis debugging**: Monitor cache hits/misses via Prometheus metrics

## External References
- [GitHub Pages Documentation](https://redhatinsights.github.io/insights-results-smart-proxy/)
- [GoDoc API Documentation](https://godoc.org/github.com/RedHatInsights/insights-results-smart-proxy)
- [Insights Behavioral Spec](https://github.com/RedHatInsights/insights-behavioral-spec)
- [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
- [Insights Content Service](https://github.com/RedHatInsights/insights-content-service)
- [app-common-go](https://github.com/RedHatInsights/app-common-go)
- [Insights Content Service](https://gitlab.cee.redhat.com/ccx/content-service)
- [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator)
- [Upgrade Risk Prediction Service](https://gitlab.cee.redhat.com/ccx/ccx-upgrades-data-eng)
- [AMS (OCM Account Management Service)](https://github.com/openshift-online/ocm-sdk-go/tree/main/accountsmgmt/v1)

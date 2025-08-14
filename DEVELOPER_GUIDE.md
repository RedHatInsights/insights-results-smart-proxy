# Developer Guide

## Overview

The Insights Results Smart Proxy is a Go-based service that acts as a proxy between external data pipeline clients and various backend services. It provides unified access to the [Insights Results Aggregator](https://github.com/RedHatInsights/insights-results-aggregator) and [Insights Content Service](https://github.com/RedHatInsights/insights-content-service), allowing clients to access both report results and rule content metadata through a single API.

## Architecture

### Core Components

- **Smart Proxy** (`smart_proxy.go`): Main entry point and service orchestration
- **Server** (`server/`): HTTP server implementation with REST API endpoints
- **Authentication** (`auth/`): JWT-based authentication and RBAC authorization
- **Content** (`content/`): Rule content parsing and management
- **Metrics** (`metrics/`): Prometheus metrics collection
- **Services** (`services/`): Backend service integrations (Redis, etc.)
- **Types** (`types/`): Common data structures and type definitions

### API Structure

The service exposes multiple API versions:

- **v1 API** (`server/endpoints_v1.go`, `server/handlers_v1.go`): Legacy endpoints
- **v2 API** (`server/endpoints_v2.go`, `server/handlers_v2.go`): Current endpoints
- **Debug API** (`server/endpoints_dbg.go`): Development and debugging endpoints

### Key Features

- **Proxy Architecture**: Aggregates data from multiple backend services
- **Authentication**: JWT token validation with RBAC support
- **Caching**: Redis-based caching for improved performance
- **Metrics**: Prometheus metrics for monitoring and observability
- **Rule Management**: Content parsing and rule acknowledgment handling
- **Upgrade Risk Prediction**: ML-based cluster upgrade risk assessment

## Development Setup

### Prerequisites

- Go 1.23+
- Redis (for caching)
- Access to backend services (Insights Results Aggregator, Content Service)

### Configuration

The service uses TOML configuration files:

- `config.toml`: Production configuration
- `config-devel.toml`: Development configuration

Key configuration sections:
- Server settings (port, timeouts)
- Backend service URLs
- Authentication settings
- Redis configuration
- Logging configuration

### Building and Running

```bash
# Build the service
make build

# Run with development config
./insights-results-smart-proxy -config config-devel.toml

# Run tests
make test

# Check code quality
make style
```

### Testing

The project includes comprehensive testing:

- **Unit Tests**: Located alongside source files (`*_test.go`)
- **Integration Tests**: In `tests/` directory
- **BDD Tests**: External behavioral tests via [Insights Behavioral Spec](https://github.com/RedHatInsights/insights-behavioral-spec)

### Development Workflow

1. **Make Changes**: Implement features in appropriate packages
2. **Add Tests**: Ensure new code has adequate test coverage
3. **Run Quality Checks**: Use `make style` to verify code quality
4. **Test Locally**: Run unit and integration tests
5. **Update Documentation**: Update API docs and this guide as needed

## API Design Patterns

### Request Flow

1. **Authentication**: JWT token validation via middleware
2. **Authorization**: RBAC checks for resource access
3. **Request Processing**: Business logic in handlers
4. **Backend Calls**: Proxy requests to appropriate services
5. **Response Assembly**: Aggregate and format responses
6. **Caching**: Cache responses where appropriate

### Error Handling

The service uses structured error responses defined in `server/errors.go`:

- HTTP status codes follow REST conventions
- Error responses include machine-readable error codes
- Detailed error messages for debugging (in development mode)

### Data Flow

```
Client Request → Authentication → Authorization → Handler → Backend Services → Response Assembly → Client Response
```

## Contributing

### Code Organization

- Keep handlers focused on HTTP concerns
- Business logic should be in service layers
- Use dependency injection for testability
- Follow Go naming conventions
- Add comprehensive error handling

### Adding New Endpoints

1. Define endpoint in appropriate `endpoints_*.go` file
2. Implement handler in corresponding `handlers_*.go` file
3. Add authentication/authorization as needed
4. Include comprehensive tests
5. Update OpenAPI documentation

### Performance Considerations

- Use Redis caching for expensive operations
- Implement proper connection pooling
- Add request timeouts
- Monitor metrics and optimize bottlenecks

## Monitoring and Observability

### Metrics

The service exposes Prometheus metrics via `/metrics` endpoint:

- Request counts and durations
- Error rates
- Backend service health
- Cache hit/miss ratios

### Logging

Structured logging using zerolog:

- Configurable log levels
- Request tracing
- Error context preservation
- Performance logging

### Health Checks

- `/api/v1/info`: Service information and health status
- Backend service connectivity checks
- Database connection monitoring

## Security

### Authentication

- JWT token validation
- Token signature verification
- Expiration checks

### Authorization

- Role-Based Access Control (RBAC)
- Organization-based data isolation
- Feature flag enforcement

### Best Practices

- Input validation and sanitization
- SQL injection prevention
- Rate limiting (external)
- Secure header handling

## Deployment

### Container Deployment

The service is containerized using the provided `Dockerfile`:

- Multi-stage build for optimized image size
- Non-root user execution
- Health check configuration

### Configuration Management

- Environment-specific configs
- Secret management via external systems
- Feature flag configuration

## Troubleshooting

### Common Issues

1. **Backend Service Connectivity**: Check service URLs and network access
2. **Authentication Failures**: Verify JWT token configuration
3. **Cache Issues**: Check Redis connectivity and configuration
4. **Performance Problems**: Review metrics and logs for bottlenecks

### Debug Tools

- Debug endpoints for service introspection
- Verbose logging modes
- Prometheus metrics for monitoring
- Health check endpoints

## Additional Resources

- [Full Documentation](https://redhatinsights.github.io/insights-results-smart-proxy/)
- [API Documentation](docs/interface/)
- [Architecture Diagrams](docs/images/)
- [Contributing Guidelines](CONTRIBUTING.md)
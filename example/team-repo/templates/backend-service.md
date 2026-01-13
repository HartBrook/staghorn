# Project Guidelines

This is a backend service.

## Architecture

- Follow clean architecture principles
- Keep business logic separate from infrastructure
- Use dependency injection for testability

## API Design

- Use RESTful conventions
- Version APIs in the URL path (`/v1/...`)
- Return consistent error responses
- Document endpoints with OpenAPI/Swagger

## Database

- Use migrations for schema changes
- Never run raw queries in business logic
- Use transactions for multi-step operations

## Testing

- Write unit tests for business logic
- Write integration tests for API endpoints
- Use test fixtures/factories for test data

## Deployment

- Service runs in Kubernetes
- Configuration via environment variables
- Health check endpoint: `GET /health`
- Metrics endpoint: `GET /metrics`

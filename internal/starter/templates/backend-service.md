# Backend Service Template

This template is for backend services and APIs.

## Project Structure

Follow these conventions for organizing backend services:
- `/cmd` - Application entrypoints
- `/internal` - Private application code
- `/pkg` - Public library code
- `/api` - API definitions (OpenAPI, protobuf, etc.)

## API Design

- Use RESTful conventions for HTTP APIs
- Version APIs in the URL path (e.g., `/v1/users`)
- Return consistent error response formats
- Document all endpoints

## Database

- Use migrations for schema changes
- Never store sensitive data in plain text
- Add indexes for frequently queried columns
- Use connection pooling

## Error Handling

- Return appropriate HTTP status codes
- Include error codes for programmatic handling
- Log errors with context for debugging
- Don't expose internal errors to clients

## Testing

- Test API endpoints with integration tests
- Mock external dependencies
- Test error paths, not just happy paths
- Use fixtures for test data

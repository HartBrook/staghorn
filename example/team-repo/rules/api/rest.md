---
paths:
  - "src/api/**/*.ts"
  - "src/api/**/*.js"
  - "src/routes/**/*.ts"
  - "src/routes/**/*.js"
  - "api/**/*.py"
  - "app/api/**/*.py"
  - "internal/api/**/*.go"
  - "pkg/api/**/*.go"
---
# REST API Standards

## HTTP Methods

- Use GET for retrieving resources (idempotent, cacheable)
- Use POST for creating new resources
- Use PUT for full resource replacement
- Use PATCH for partial updates
- Use DELETE for removing resources

## Status Codes

- 2xx for successful operations (200 OK, 201 Created, 204 No Content)
- 4xx for client errors (400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 422 Unprocessable Entity)
- 5xx for server errors (500 Internal Server Error, 502 Bad Gateway, 503 Service Unavailable)

## Request/Response

- Validate all request input before processing
- Use consistent error response format with error codes and messages
- Include pagination for list endpoints (limit, offset or cursor-based)
- Version APIs in the URL path (/v1/, /v2/) or via Accept headers
- Use JSON for request and response bodies

## Error Responses

Return errors in a consistent format:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [...]
  }
}
```

## Documentation

- Document endpoints with OpenAPI/Swagger specifications
- Include request/response examples for each endpoint
- Document authentication requirements
- List all possible error responses with their codes

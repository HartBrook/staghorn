# Security Guidelines

## Secrets and Credentials

- Never commit secrets, API keys, or credentials to version control
- Use environment variables or secret managers for sensitive configuration
- Add secret patterns to `.gitignore` (e.g., `.env`, `*.key`, `credentials.json`)
- Rotate credentials regularly and after any potential exposure

## Input Validation

- Validate and sanitize all external input at system boundaries
- Use parameterized queries for database access to prevent SQL injection
- Escape output appropriately to prevent XSS attacks
- Validate file paths to prevent path traversal attacks
- Implement rate limiting for public endpoints

## Authentication and Authorization

- Apply the principle of least privilege
- Validate authentication tokens on every request
- Use secure session management practices
- Implement proper CORS policies
- Log security-relevant events for audit trails

## Dependencies

- Keep dependencies updated to patch known vulnerabilities
- Use lock files to ensure reproducible builds
- Review dependency changes before updating
- Use automated vulnerability scanning in CI/CD

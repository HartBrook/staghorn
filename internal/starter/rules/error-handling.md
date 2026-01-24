# Error Handling

## Error Design

- Handle errors explicitly; never silently ignore them
- Include context in error messages: what failed and why
- Create domain-specific error types for errors callers need to handle differently
- Let errors propagate unless you can meaningfully handle them at that level

## Error Messages

- Write error messages for the person who will read them
- Include relevant context (IDs, values, state) to aid debugging
- Avoid exposing sensitive information in error messages
- Use consistent error message formatting across the codebase

## Error Recovery

- Fail fast; detect and report errors as early as possible
- Clean up resources properly when errors occur
- Consider whether operations should be retried
- Log errors with appropriate severity levels

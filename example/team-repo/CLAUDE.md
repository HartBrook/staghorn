# Acme Corp Engineering Standards

Guidelines for Claude Code across all Acme projects.

## Code Style

- Write clear, self-documenting code
- Prefer explicit over implicit
- Keep functions small and focused (under 50 lines)
- Use meaningful variable and function names

## Code Review

- All PRs require at least one approval
- Keep PRs under 400 lines when possible
- Include tests for new functionality
- Update documentation when changing public APIs

## Git Conventions

- Write commit messages in imperative mood ("Add feature" not "Added feature")
- Keep commits atomic and focused
- Reference issue numbers in commit messages when applicable

## Security

- Never commit secrets, API keys, or credentials
- Use environment variables for configuration
- Validate all user input
- Follow OWASP security guidelines

## Testing

- Write tests for all new features
- Maintain test coverage above 80%
- Include both unit and integration tests where appropriate

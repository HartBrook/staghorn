# CLI Tool Template

This template is for command-line interface tools.

## Project Structure

Follow these conventions for organizing CLI tools:
- `/cmd` - Main entrypoint(s)
- `/internal/cli` - Command implementations
- `/internal/config` - Configuration handling
- `/docs` - Documentation and man pages

## Command Design

- Use subcommands for different operations
- Provide `-h/--help` for all commands
- Support `-v/--verbose` for debugging
- Use consistent flag naming across commands

## Input/Output

- Read from stdin when appropriate
- Support piping and redirection
- Use exit codes meaningfully (0 = success)
- Format output for both humans and machines (e.g., `--json`)

## Configuration

- Support config files in standard locations
- Allow environment variable overrides
- Provide sensible defaults
- Document all configuration options

## Error Handling

- Print errors to stderr
- Include actionable error messages
- Suggest fixes when possible
- Never silently fail

## Distribution

- Provide binaries for common platforms
- Support package managers (Homebrew, apt, etc.)
- Include shell completions
- Document installation methods

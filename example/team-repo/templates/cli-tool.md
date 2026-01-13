# Project Guidelines

This is a command-line tool.

## User Experience

- Provide helpful `--help` output
- Use clear, consistent flag naming
- Show progress for long operations
- Support both interactive and non-interactive modes

## Output

- Default to human-readable output
- Support `--json` flag for machine-readable output
- Use color sparingly and respect `NO_COLOR` env var
- Write errors to stderr, output to stdout

## Configuration

- Support config file (YAML or TOML)
- Allow env var overrides
- Allow CLI flag overrides (highest precedence)

## Error Handling

- Provide actionable error messages
- Include suggestions for common mistakes
- Use appropriate exit codes

## Documentation

- Include man page or detailed `--help`
- Provide examples for common use cases
- Document all configuration options

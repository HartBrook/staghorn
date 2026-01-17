# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-01-17

### Added

- `stag eval` command to run behavioral tests against CLAUDE.md configs
- `stag eval list` command to list available evals
- `stag eval init` command to install starter evals
- `stag eval info` command to show eval details
- 25 starter evals covering security, code quality, docs, git, and language best practices
- Eval syncing from team repos via `stag sync`
- `stag team validate` now validates evals in team repositories
- Filter evals by tag (`--tag`), name, or specific test (`--test`)
- Multiple output formats: table, JSON, GitHub Actions annotations
- Debug mode (`--debug`) to see full Claude responses and preserve temp files
- Dry-run mode (`--dry-run`) to preview without API calls
- EVALS_GUIDE.md with comprehensive documentation on writing and debugging evals

### Changed

- `stag sync` now fetches evals from team repo's `evals/` directory

## [0.3.0] - 2026-01-17

### Added

- `staghorn search` command to discover community configs from GitHub
- Multi-source configuration support for pulling configs from different repositories
- Trust system with warnings for untrusted sources and org-level trust
- Unauthenticated GitHub client for public repo access (no auth required for community configs)
- Language aliases for search filtering (e.g., `golang` → `go`, `py` → `python`, `sh` → `bash`)
- Interactive config browsing in `staghorn init` with public config discovery

### Changed

- Improved `staghorn init` flow with three options: browse public, connect repo, or start fresh
- Search methods now accept context for timeout/cancellation support
- Better error messages for invalid selections during init

## [0.2.0] - 2026-01-16

### Added

- `staghorn team init` command to bootstrap shared team standards repositories
- `staghorn team validate` command to validate team repo structure
- Interactive template selection in `staghorn project init` when team templates are available
- Selective bootstrap functions for commands and languages (all/some/none selection)
- 3 embedded project templates: backend-service, frontend-app, cli-tool

### Changed

- Renamed "actions" to "commands" throughout the codebase for clarity
- Improved `staghorn init` with better starter support and interactive prompts

## [0.1.0] - 2026-01-15

### Added

- Initial release of staghorn CLI
- `staghorn init` command to set up personal Claude Code configuration
- `staghorn sync` command to sync team configurations from GitHub
- `staghorn project init` command to initialize project-level configs
- `staghorn project generate` command to generate CLAUDE.md from sources
- `staghorn commands` subcommands for managing Claude Code commands
- Embedded starter commands: api-design, code-review, debug, doc-gen, explain, migrate, pr-prep, refactor, security-audit, test-gen
- Embedded language configs: go, java, python, ruby, rust, typescript
- Support for team, personal, and project configuration layers
- Automatic CLAUDE.md generation with layered content

[Unreleased]: https://github.com/HartBrook/staghorn/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/HartBrook/staghorn/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/HartBrook/staghorn/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/HartBrook/staghorn/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/HartBrook/staghorn/releases/tag/v0.1.0

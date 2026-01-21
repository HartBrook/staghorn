# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.0] - 2026-01-20

### Added

- **Provenance tracking** for merged configs
  - Merged CLAUDE.md now includes `<!-- staghorn:source:LAYER -->` comments marking content origin
  - Language-specific content uses extended markers: `<!-- staghorn:source:LAYER:LANGUAGE -->` (e.g., `team:python`)
  - New `merge.ParseProvenance()` and `merge.ParseProvenanceSections()` functions to extract content by source
  - `merge.ListSources()` returns unique full sources (e.g., "team", "team:python") in order of appearance
  - `merge.ListLayers()` returns unique layers (team, personal, project) ignoring language subsections
  - `merge.ParseProvenanceByLayer()` aggregates content by layer for extraction
  - `merge.HasProvenance()` checks if content has provenance markers
  - Enables tooling to understand which layer contributed each section

- **Source repo detection** with `.staghorn/source.yaml`
  - `stag team init` now creates `.staghorn/source.yaml` to mark team/community repos
  - `config.IsSourceRepo()` detects source repos for special handling
  - `stag team validate` checks for source repo marker

- **Source repo mode** for seamless team repo development
  - When inside a source repo, commands operate on local files instead of cache:
    - `stag edit team` opens `./CLAUDE.md` directly (previously read-only)
    - `stag info --layer team` reads from `./CLAUDE.md`
    - `stag optimize --layer team` reads/writes `./CLAUDE.md`
    - `stag eval --layer team` tests local content
  - Enables natural edit-test-commit workflow for maintaining team standards

- **Provenance-aware optimization** for merged configs
  - `stag optimize --apply` now works with `--layer merged` (the default)
  - Optimized content is split by provenance markers and written back to each source layer
  - Enables optimizing the full merged config while preserving layer separation

- **Project layer optimization** support
  - `stag optimize --layer project --apply` now saves optimized content to `.staghorn/project.md`

### Changed

- `stag sync` now generates CLAUDE.md with provenance comments by default
- Merge options include `AnnotateSources` flag to control provenance comment generation
- **Language sections are now top-level H2 headers** instead of nested under `## Language-Specific Guidelines`
  - Each language (Python, Go, TypeScript, etc.) becomes its own `## {Language}` section
  - Content headers within language files are demoted one level (e.g., `##` → `###`) to maintain valid hierarchy
  - Personal additions use `### Personal Additions` sub-headers
  - This creates cleaner, flatter document structure optimized for Claude Code

## [0.6.0] - 2026-01-19

### Added

- `stag optimize` command to compress configs and reduce token usage in Claude's context window
  - LLM-powered optimization preserves semantic meaning while reducing size
  - Anchor validation ensures critical content (tool names, file paths, commands) is preserved
  - `--deterministic` mode for fast cleanup without API calls
  - `--layer` flag to optimize team, personal, or merged configs
  - `--apply` flag to save optimized content back to source
  - Caching layer avoids re-optimizing unchanged content
- Token count display in `stag info` output with warning when exceeding 3,000 tokens
- Optimization suggestion in `stag sync` output for large configs

### Changed

- `stag info` now shows merged config size in tokens
- Updated README with optimize command documentation and troubleshooting

## [0.5.0] - 2026-01-18

### Added

- `stag eval validate` command to validate eval YAML files before running
  - Checks assertion types, required fields, YAML structure, and naming conventions
  - Provides helpful suggestions for common typos (e.g., `llm_rubric` → `llm-rubric`)
  - Distinguishes between errors (blocking) and warnings (non-blocking)
- `stag eval create` command to create new evals from templates
  - Interactive wizard for guided eval creation
  - Four built-in templates: security, quality, language, blank
  - `--template` flag to skip wizard and use template directly
  - `--from` flag to copy and customize existing evals
  - `--name` and `--description` flags for non-interactive creation
- `--project` flag for `stag eval create` to save evals to `.staghorn/evals/`
- `--team` flag for `stag eval create` to save evals to `./evals/` for team/community sharing
- Example evals in `example/team-repo/evals/` demonstrating team eval patterns

### Changed

- Updated EVALS_GUIDE.md with comprehensive documentation for validate and create commands
- Expanded CLI flags reference in README.md with new eval commands

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

[Unreleased]: https://github.com/HartBrook/staghorn/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/HartBrook/staghorn/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/HartBrook/staghorn/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/HartBrook/staghorn/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/HartBrook/staghorn/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/HartBrook/staghorn/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/HartBrook/staghorn/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/HartBrook/staghorn/releases/tag/v0.1.0

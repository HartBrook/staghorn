# Staghorn

Sync Claude Code configs from GitHub — for teams, communities, or just yourself.

## Why Staghorn?

**For teams:** Your engineering standards live in one GitHub repo. Everyone syncs from it. When standards change, one PR updates the whole team.

**For individuals:** Browse community configs to jumpstart your setup, or keep your personal config in a repo and sync it across all your machines.

**For everyone:** Layer your personal preferences on top of any base config. Your style, their standards.

## Quick Start

```bash
# Install
brew tap HartBrook/tap
brew install staghorn

# Set up
stag init
```

Choose how you want to get started:

1. **Browse public configs** — Install community-shared configs from GitHub
2. **Connect to a team repo** — Sync your team's private standards
3. **Start fresh** — Just use the built-in starter commands

Then run `stag sync` periodically to stay up to date.

## Finding Configs

### Browse Community Configs

```bash
# Search for public configs
stag search

# Filter by language or topic
stag search --lang python
stag search --tag security

# Install directly if you know the repo
stag init --from acme/claude-standards
```

Public configs are GitHub repos with the `staghorn-config` topic. Find configs tailored for Python, Go, React, security-focused development, and more.

### Connect to Your Team

```bash
stag init
# Choose option 2: "Connect to a private repository"
# Enter your team's repo URL
```

Your team admin sets up a standards repo (see [For Config Publishers](#for-config-publishers) below), and everyone syncs from it. Authentication via `gh auth login` or `STAGHORN_GITHUB_TOKEN`.

## How It Works

Staghorn pulls configs from GitHub, merges them with your personal preferences, and writes the result to where Claude Code expects it:

```
Team/community config (GitHub)    ─┐
                                   ├─► ~/.claude/CLAUDE.md
Your personal additions           ─┘

Project config (.staghorn/)       ─► ./CLAUDE.md
```

The layering means you get shared standards *plus* your personal style. You never edit the output files directly — Staghorn manages them.

**Advanced:** You can pull different parts of your config from different sources — team standards for your base config, community best practices for specific languages. See [Multi-Source Configuration](#multi-source-configuration).

## Commands

| Command              | Description                                        |
| -------------------- | -------------------------------------------------- |
| `stag init`          | Set up staghorn (browse configs or connect repo)   |
| `stag sync`          | Fetch latest config from GitHub and apply          |
| `stag search`        | Search for community configs                       |
| `stag edit`          | Edit personal config (auto-applies on save)        |
| `stag edit -l <lang>`| Edit personal language config (e.g., `-l python`)  |
| `stag info`          | Show current config state                          |
| `stag languages`     | Show detected and configured languages             |
| `stag commands`      | List available commands                            |
| `stag run <command>` | Run a command (outputs prompt to stdout)           |
| `stag project`       | Manage project-level config                        |
| `stag team`          | Bootstrap or validate a team standards repo        |
| `stag version`       | Print version number                               |

### Typical Workflow

```bash
# Update config (do this periodically)
stag sync

# Check current state
stag info

# Add personal preferences (auto-applies)
stag edit
```

## Customization

### Personal Preferences

Your personal additions layer on top of source configs:

```bash
# Open your personal config in $EDITOR (auto-applies on save)
stag edit
```

This opens `~/.config/staghorn/personal.md`. Add whatever you like:

```markdown
## My Preferences

- I prefer concise responses unless I ask for detail
- Always use TypeScript strict mode
- Explain your reasoning before showing code
```

### Personal Language Preferences

Set preferences for specific languages that only apply when detected in a project:

```bash
stag edit --language python
stag edit -l go
```

This creates/edits `~/.config/staghorn/languages/<lang>.md`.

### Project Config

Optionally manage project-level `./CLAUDE.md` files:

```bash
stag project init                          # Initialize
stag project init --template=backend-service  # From template
stag project edit                          # Edit
```

The source file is `.staghorn/project.md` — both it and `./CLAUDE.md` should be committed.

## Reusable Commands

Commands are reusable prompts for common workflows. Staghorn includes 10 starter commands:

| Command | Description |
|--------|-------------|
| `code-review` | Thorough code review with checklist |
| `security-audit` | Scan for vulnerabilities |
| `pr-prep` | Prepare PR description |
| `explain` | Explain code in plain English |
| `refactor` | Suggest refactoring improvements |
| `test-gen` | Generate unit tests |
| `debug` | Help diagnose a bug |
| `doc-gen` | Generate documentation |
| `migrate` | Help migrate code |
| `api-design` | Design API interfaces |

```bash
# List available commands
stag commands

# Run a command
stag run security-audit

# Run with arguments
stag run code-review --focus=security
```

Install commands as Claude Code slash commands:

```bash
stag commands init --claude    # Install to ~/.claude/commands/
```

---

# Going Deeper

## For Config Publishers

Everything below is for people creating configs to share — whether for a team or the community.

### Creating a Team Repository

Use `team init` to bootstrap a new standards repository:

```bash
mkdir my-team-standards && cd my-team-standards
git init
stag team init
```

This creates:
- A starter `CLAUDE.md` with common guidelines
- Optional commands, language configs, and project templates
- A README explaining the repo structure

Push to GitHub and share the URL with your team.

### Making Your Config Discoverable

For your config to appear in `stag search`, add GitHub topics to your repository:

**Required:**
- `staghorn-config` — Makes your repo discoverable via `stag search`

**Language topics** (for `--lang` filtering):
- Add topics like `python`, `go`, `typescript`, `rust`, `java`, `ruby`
- Users can search with aliases: `golang` → `go`, `py` → `python`, `ts` → `typescript`

**Custom tags** (for `--tag` filtering):
- Add any topics you want: `security`, `web`, `ai`, `backend`, etc.

Example: A Python security-focused config should have topics:
```
staghorn-config, python, security
```

Then users can find it with:
```bash
stag search --lang python --tag security
stag search --lang py  # aliases work too
```

### Repository Structure

```
your-org/claude-standards/
├── CLAUDE.md           # Guidelines (required)
├── commands/           # Reusable prompts (optional)
│   ├── security-audit.md
│   └── code-review.md
├── languages/          # Language-specific configs (optional)
│   ├── python.md
│   └── go.md
└── templates/          # Project templates (optional)
    └── backend-service.md
```

> **See [`example/team-repo/`](example/team-repo/) for a complete example.**

### Validating a Repository

```bash
stag team validate
```

Checks that CLAUDE.md exists, commands have valid frontmatter, etc.

### Instructional Comments

Add comments that appear in source but are stripped from output:

```markdown
## Code Review Guidelines

<!-- [staghorn] Tip: Customize this section in your personal.md -->

- All PRs require one approval
```

## Trusted Sources

When installing from a new source, Staghorn shows a warning for untrusted repos. You can pre-trust sources in your config:

```yaml
# ~/.config/staghorn/config.yaml
trusted:
  - acme-corp              # Trust all repos from this org
  - community/python-config  # Trust a specific repo
```

Private repos auto-trust their org during `stag init`.

## Multi-Source Configuration

Pull different parts of your config from different repositories:

```yaml
# ~/.config/staghorn/config.yaml
source:
  default: my-company/standards       # Base standards from your team
  languages:
    python: community/python-standards  # Community Python config
    go: my-company/go-standards         # Team-specific Go config
  commands:
    security-audit: security-team/audits  # Commands from another team
```

This is useful when you want team standards for some things, but community best practices for specific languages.

## Language-Specific Config

### How It Works

Language configs are markdown files in `languages/` directories, layered just like the main config:

1. **Team/community** — `languages/` in the source repo
2. **Personal** — `~/.config/staghorn/languages/`
3. **Project** — `.staghorn/languages/`

### Global vs Project

- **Global (`~/.claude/CLAUDE.md`)**: Includes all available language configs
- **Project (`./CLAUDE.md`)**: Auto-detects languages from marker files (e.g., `go.mod`, `pyproject.toml`)

### Configuration Options

```yaml
# ~/.config/staghorn/config.yaml

# Only include specific languages globally
languages:
  enabled:
    - python
    - go

# Or exclude specific languages
languages:
  disabled:
    - javascript
```

### Supported Languages

| Language   | Marker Files                                    |
| ---------- | ----------------------------------------------- |
| Python     | `pyproject.toml`, `setup.py`, `requirements.txt`, `Pipfile` |
| Go         | `go.mod`                                        |
| TypeScript | `tsconfig.json`                                 |
| JavaScript | `package.json`                                  |
| Rust       | `Cargo.toml`                                    |
| Java       | `pom.xml`, `build.gradle`                       |
| Ruby       | `Gemfile`                                       |
| C#         | `*.csproj`, `*.sln`                             |
| Swift      | `Package.swift`                                 |
| Kotlin     | `build.gradle.kts`                              |

> When both TypeScript and JavaScript are detected, TypeScript takes precedence.

## Creating Commands

A command is a markdown file with YAML frontmatter:

```markdown
---
name: security-audit
description: Scan for common security vulnerabilities
tags: [security, review]
args:
  - name: path
    description: Directory to audit
    default: "."
  - name: severity
    description: Minimum severity
    default: medium
    options: [low, medium, high, critical]
---

# Security Audit

Review the code at {{path}} for security vulnerabilities.
Report issues at {{severity}} severity or higher.
```

Commands can come from three sources (highest precedence first):
1. **Project** — `.staghorn/commands/`
2. **Personal** — `~/.config/staghorn/commands/`
3. **Team/community** — `commands/` in the source repo

## Configuration Reference

### Config File

`~/.config/staghorn/config.yaml`:

```yaml
version: 1

# Simple: single source
source: "acme/standards"

# Or multi-source (see Multi-Source Configuration above)
# source:
#   default: acme/standards
#   languages:
#     python: community/python-standards

# Trusted orgs/repos (skip confirmation prompts)
trusted:
  - acme-corp
  - community/python-standards

cache:
  ttl: "24h"              # How long to cache before re-fetching

languages:
  auto_detect: true       # Detect from project marker files
  enabled: []             # Explicit list (overrides auto-detect)
  disabled: []            # Languages to exclude
```

### File Locations

| File                              | Purpose                                         |
| --------------------------------- | ----------------------------------------------- |
| `~/.config/staghorn/config.yaml`  | Staghorn settings                               |
| `~/.config/staghorn/personal.md`  | Your personal additions                         |
| `~/.config/staghorn/commands/`    | Personal commands                               |
| `~/.config/staghorn/languages/`   | Personal language configs                       |
| `~/.cache/staghorn/`              | Cached team/community configs                   |
| `~/.claude/CLAUDE.md`             | **Output** — merged global config               |
| `.staghorn/project.md`            | Project config source (you edit this)           |
| `.staghorn/commands/`             | Project-specific commands                       |
| `.staghorn/languages/`            | Project-specific language configs               |
| `./CLAUDE.md`                     | **Output** — merged project config              |

## CLI Flags Reference

```bash
# Sync options
stag sync --fetch-only     # Fetch without applying
stag sync --apply-only     # Apply cached config without fetching
stag sync --force          # Re-fetch even if cache is fresh
stag sync --offline        # Use cached config only (no network)
stag sync --config-only    # Sync config only, skip commands/languages
stag sync --commands-only  # Sync commands only
stag sync --languages-only # Sync language configs only

# Search options
stag search --lang go      # Filter by language
stag search --tag security # Filter by topic
stag search --limit 10     # Limit results

# Init options
stag init --from owner/repo  # Install directly from a repo

# Edit options
stag edit --no-apply       # Edit without auto-applying

# Info options
stag info --content        # Show full merged config
stag info --layer team     # Show only team config (also: personal, project)
stag info --sources        # Annotate output with source information

# Command options
stag commands --tag security   # Filter commands by tag
stag commands --source team    # Filter by source (team, personal, project)
stag run <command> --dry-run   # Preview command without rendering
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap HartBrook/tap
brew install staghorn
```

### From Source

```bash
go install github.com/HartBrook/staghorn/cmd/staghorn@latest
```

The `stag` alias is also available (symlink to `staghorn`).

## Authentication

**Public repos** — No authentication needed. Staghorn fetches community configs without any setup.

**Private repos** — You'll need GitHub access:

```bash
# Option 1: GitHub CLI (recommended)
brew install gh
gh auth login

# Option 2: Personal access token
export STAGHORN_GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

## Migrating Existing Config

If you already have a `~/.claude/CLAUDE.md`, the first `stag sync` will detect it and offer:

1. **Migrate** — Move content to `~/.config/staghorn/personal.md`
2. **Backup** — Save a copy before overwriting
3. **Abort** — Cancel and leave unchanged

## Troubleshooting

**"No editor found"**
```bash
export EDITOR="code --wait"  # VS Code
export EDITOR="vim"          # Vim
```

**"Could not authenticate with GitHub"**
Either `gh auth login` or set `STAGHORN_GITHUB_TOKEN`.

**"Cache is stale" warnings**
Run `stag sync --force` to re-fetch.

**Config not updating after edit**
Make sure you saved. If using `--no-apply`, run `stag sync --apply-only`.

**Languages not being detected**
Check `stag languages`. Ensure marker files exist in project root.

**Command not found**
Run `stag commands` to see available commands. Project overrides personal, which overrides source.

## License

MIT

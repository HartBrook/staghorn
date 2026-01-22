# Contributing to Staghorn

Thank you for your interest in contributing to Staghorn! This guide will help you get started with development.

## Prerequisites

- Go 1.21 or later
- Git
- GitHub CLI (`gh`) for testing authentication features

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/HartBrook/staghorn.git
cd staghorn
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Binary

```bash
go build -o staghorn ./cmd/staghorn
```

### 4. Run Tests

```bash
go test ./...
```

## Project Structure

```
staghorn/
├── cmd/
│   └── staghorn/
│       └── main.go              # Entry point
├── internal/
│   ├── cache/                   # Local cache management
│   ├── cli/                     # Cobra commands
│   ├── commands/                # Command parsing and execution
│   ├── config/                  # Config file parsing
│   ├── errors/                  # Typed error handling
│   ├── github/                  # GitHub API client
│   ├── integration/             # Integration tests
│   │   └── testdata/fixtures/   # YAML test fixtures
│   ├── merge/                   # Markdown merge logic
│   └── starter/                 # Embedded starter content
│       ├── commands/            # Starter command templates
│       ├── languages/           # Starter language configs
│       └── templates/           # Starter project templates
├── go.mod
├── go.sum
├── ARCHITECTURE.md              # Detailed design documentation
├── README.md
└── CONTRIBUTING.md
```

### Key Packages

| Package | Purpose |
|---------|---------|
| `internal/cli` | All CLI commands (Cobra) |
| `internal/cache` | Stores fetched configs with metadata |
| `internal/commands` | Command parsing and execution |
| `internal/config` | Parses `config.yaml` and manages paths |
| `internal/errors` | Typed errors with hints |
| `internal/github` | GitHub API client with auth handling |
| `internal/integration` | Integration tests with YAML fixtures |
| `internal/merge` | Section-based markdown merging |
| `internal/starter` | Embedded starter commands, languages, and templates |

## Development Workflow

### Running Locally

```bash
# Build and run
go build -o staghorn ./cmd/staghorn
./staghorn --help

# Or use go run
go run ./cmd/staghorn --help
```

### Running Specific Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/merge/...

# With verbose output
go test -v ./internal/config/...

# With coverage
go test -cover ./...
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Code Style

### General Guidelines

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use meaningful variable and function names
- Keep functions focused and small
- Add comments for non-obvious logic

### Error Handling

Use the typed error system in `internal/errors`:

```go
import "github.com/HartBrook/staghorn/internal/errors"

return errors.New(
    "Failed to fetch team config",
    errors.CodeGitHubAuthFailed,
    "Run `gh auth login` or set STAGHORN_GITHUB_TOKEN",
)
```

### Testing

Write table-driven tests:

```go
func TestMergeLayers(t *testing.T) {
    tests := []struct {
        name     string
        team     string
        personal string
        want     string
    }{
        {
            name:     "appends personal additions",
            team:     "## Code Style\n\nTeam rules.",
            personal: "## Code Style\n\nMy preference.",
            want:     "## Code Style\n\nTeam rules.\n\n### Personal Additions\n\nMy preference.",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MergeLayers(tt.team, tt.personal)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

Integration tests verify the full sync workflow produces correct merged output. They use filesystem isolation (`t.TempDir()`) and don't touch real config directories.

**Run integration tests:**

```bash
# All integration tests (mocked, no network)
go test ./internal/integration/...

# With verbose output
go test -v ./internal/integration/...

# Run specific test
go test -v ./internal/integration/... -run TestIntegration_Fixtures/basic_sync

# Live tests against real GitHub (requires gh auth)
go test -tags=live ./internal/integration/...
```

**Adding a new integration test:**

Create a YAML fixture in `internal/integration/testdata/fixtures/`:

```yaml
name: "my_scenario"
description: "What this test verifies"

setup:
  team:
    source: "owner/repo"
    claude_md: |
      ## Code Style

      Team content.
    languages:
      python: |
        Use type hints.

  personal:
    personal_md: |
      ## My Preferences

      Personal content.

  config:
    version: 1
    source: "owner/repo"

assertions:
  output_exists: true
  header:
    managed_by: true
    source_repo: "owner/repo"
  provenance:
    has_team: true
    has_personal: true
    order: ["team", "personal"]
  contains:
    - "Team content"
    - "Personal content"
  not_contains:
    - "should not appear"
  sections:
    - "## Code Style"
  languages:
    - name: "python"
      has_team_content: true
      contains:
        - "Use type hints"
```

The test runner automatically discovers and runs new fixtures.

## Adding a New Command

1. Create a new file in `internal/cli/` (e.g., `mycommand.go`)
2. Implement the command using Cobra:

```go
package cli

import "github.com/spf13/cobra"

func NewMyCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "Brief description",
        Long:  `Longer description with examples.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }

    // Add flags
    cmd.Flags().StringP("flag", "f", "", "Flag description")

    return cmd
}
```

3. Register it in `root.go`:

```go
rootCmd.AddCommand(NewMyCommand())
```

4. Add tests in `mycommand_test.go`

## Adding a New Feature

1. **Discuss first** — Open an issue to discuss the feature before implementing
2. **Update ARCHITECTURE.md** — Document design decisions for significant features
3. **Write tests** — All new code should have corresponding tests
4. **Update README.md** — Document user-facing changes

## Pull Request Process

1. **Fork and branch** — Create a feature branch from `main`
2. **Make changes** — Follow the code style guidelines
3. **Test** — Run `go test ./...` and ensure all tests pass
4. **Commit** — Write clear commit messages
5. **Push and PR** — Open a pull request with a description of changes

### PR Checklist

- [ ] Tests pass locally (`go test ./...`)
- [ ] Code is formatted (`gofmt`)
- [ ] No linter warnings (`golangci-lint run`)
- [ ] Documentation updated if needed
- [ ] Commit messages are clear

## Testing Against a Real Team Repo

Create a test repository with sample configs:

```
test-team-config/
├── CLAUDE.md
└── commands/
    ├── hello-world.md
    └── code-review.md
```

Then configure staghorn to use it:

```bash
./staghorn init
# Enter your test repo URL
```

## Release Process

Releases are automated via GoReleaser when a tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This builds binaries for all platforms and updates the Homebrew tap.

## Getting Help

- **Issues** — Report bugs or request features via GitHub Issues
- **Discussions** — Ask questions in GitHub Discussions
- **Architecture** — See [ARCHITECTURE.md](ARCHITECTURE.md) for design details

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

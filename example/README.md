# Example Team Repository

This directory shows what a team's shared Claude Code standards repository should look like.

> **Note**: This directory is excluded from the staghorn build and git. It's for documentation purposes only.

## Structure

```
team-repo/
├── CLAUDE.md              # Shared guidelines (required)
├── commands/              # Reusable prompts
│   ├── api-design.md      # Design API interfaces
│   ├── code-review.md     # Thorough code review
│   ├── debug.md           # Bug investigation helper
│   ├── doc-gen.md         # Generate documentation
│   ├── explain.md         # Explain code in plain English
│   ├── migrate.md         # Code migration assistant
│   ├── pr-prep.md         # Prepare PR descriptions
│   ├── refactor.md        # Suggest refactoring improvements
│   ├── security-audit.md  # Security vulnerability scan
│   └── test-gen.md        # Generate unit tests
├── rules/                 # Path-scoped rules
│   ├── security.md        # Security guidelines (all files)
│   ├── testing.md         # Testing standards (all files)
│   ├── api/
│   │   └── rest.md        # REST API standards (api paths)
│   └── frontend/
│       └── react.md       # React guidelines (component paths)
├── evals/                 # Behavioral tests
│   ├── team-security.yaml # Security guidelines tests
│   ├── team-quality.yaml  # Code quality tests
│   └── team-git.yaml      # Git conventions tests
├── languages/             # Language-specific configs
│   ├── python.md
│   ├── go.md
│   ├── typescript.md
│   └── rust.md
└── templates/             # Project templates
    ├── backend-service.md
    ├── react-app.md
    └── cli-tool.md
```

## Usage

To use this as your team's standards repo:

1. Create a new GitHub repository (e.g., `your-org/claude-standards`)
2. Copy the contents of `team-repo/` to your new repo
3. Customize the files to match your team's guidelines
4. Have team members run `stag init` and point to your repo

## Customization

- **CLAUDE.md**: Add your team's general coding standards
- **rules/**: Add path-scoped rules for specific file types or directories
- **evals/**: Write tests to verify Claude follows your guidelines
- **languages/**: Add configs for languages your team uses
- **commands/**: Create prompts for common workflows
- **templates/**: Add project templates for different project types

## Tips

- Keep guidelines concise and actionable
- Update regularly based on team feedback
- Use commands for repetitive tasks
- Use rules with path patterns for file-specific guidelines
- Language configs should complement, not repeat, the main CLAUDE.md

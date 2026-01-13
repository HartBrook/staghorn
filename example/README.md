# Example Team Repository

This directory shows what a team's shared Claude Code standards repository should look like.

> **Note**: This directory is excluded from the staghorn build and git. It's for documentation purposes only.

## Structure

```
team-repo/
├── CLAUDE.md              # Shared guidelines (required)
├── actions/               # Reusable prompts
│   ├── security-audit.md
│   ├── code-review.md
│   └── pr-prep.md
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
- **languages/**: Add configs for languages your team uses
- **actions/**: Create prompts for common workflows
- **templates/**: Add project templates for different project types

## Tips

- Keep guidelines concise and actionable
- Update regularly based on team feedback
- Use actions for repetitive tasks
- Language configs should complement, not repeat, the main CLAUDE.md

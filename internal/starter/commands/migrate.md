---
name: migrate
description: Help migrate code to a new pattern or version
tags: [migration, upgrade, refactor]
args:
  - name: from
    description: Current state or version
    required: true
  - name: to
    description: Target state or version
    required: true
  - name: path
    description: Scope of migration
    default: "."
---

# Migration Assistant

Migrate from **{{from}}** to **{{to}}** in `{{path}}`.

## Migration Process

1. **Audit**: Identify all code that needs to change
2. **Plan**: Determine order of changes and dependencies
3. **Transform**: Apply changes systematically
4. **Verify**: Ensure functionality is preserved

## Output

### Migration Checklist
- List all files that need changes
- Note any breaking changes
- Identify deprecated patterns to update

### Code Changes
For each change needed:
- Current code
- Updated code
- Explanation of why it changed

### Testing
- How to verify the migration worked
- Potential regressions to watch for

### Rollback
- How to undo changes if needed

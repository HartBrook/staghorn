---
name: pr-prep
description: Prepare changes for a pull request
tags: [git, pr, workflow]
args:
  - name: base
    description: Base branch for the PR
    default: main
---

# PR Preparation

Prepare the current changes for a pull request against `{{base}}`.

## Steps

1. **Review Changes**
   - Run `git diff {{base}}` to see all changes
   - Identify the scope and purpose of the changes

2. **Quality Checks**
   - Run linter and fix any issues
   - Run tests and ensure they pass
   - Check for any TODO comments that should be addressed

3. **Documentation**
   - Update README if public API changed
   - Add/update code comments where helpful
   - Update CHANGELOG if applicable

4. **Commit Hygiene**
   - Review commit messages for clarity
   - Consider squashing WIP commits
   - Ensure commits are atomic and focused

5. **Generate PR Description**
   Provide a PR description with:
   - **Summary**: What this PR does (2-3 sentences)
   - **Changes**: Bullet list of specific changes
   - **Testing**: How to test the changes
   - **Screenshots**: If UI changes (note where to add them)

## Output

Provide:
1. Any issues found that should be fixed before PR
2. A draft PR title and description
3. Suggested reviewers based on files changed (if known)

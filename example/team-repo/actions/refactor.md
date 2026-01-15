---
name: refactor
description: Suggest refactoring improvements for cleaner code
tags: [refactor, quality, cleanup]
args:
  - name: path
    description: File or directory to analyze
    default: "."
  - name: goal
    description: Primary refactoring goal
    default: readability
    options: [readability, performance, testability, modularity]
---

# Refactoring Analysis

Analyze `{{path}}` and suggest refactoring improvements focused on **{{goal}}**.

## Look For

### Readability
- Long functions that should be split
- Unclear variable or function names
- Complex conditionals that could be simplified
- Magic numbers that need constants

### Performance
- Unnecessary loops or iterations
- Missing caching opportunities
- Inefficient data structures
- Redundant computations

### Testability
- Functions with too many dependencies
- Side effects that complicate testing
- Missing dependency injection
- Tightly coupled components

### Modularity
- Code that belongs in separate modules
- Circular dependencies
- Violations of single responsibility
- Missing abstractions

## Output Format

For each suggestion:
1. **Location**: Where the issue is
2. **Problem**: What makes it hard to maintain
3. **Suggestion**: How to improve it
4. **Example**: Show before/after if helpful

Prioritize suggestions by impact. Focus on changes that provide the most value.

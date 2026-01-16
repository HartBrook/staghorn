---
name: doc-gen
description: Generate documentation for code
tags: [documentation, docs]
args:
  - name: path
    description: File or module to document
    required: true
  - name: style
    description: Documentation style
    default: markdown
    options: [markdown, jsdoc, docstring, rustdoc]
---

# Documentation Generation

Generate **{{style}}** documentation for `{{path}}`.

## Include

### API Documentation
- Function signatures and parameters
- Return values and types
- Exceptions/errors that can be raised
- Usage examples

### Module Overview
- Purpose of the module
- Key concepts and terminology
- How it fits into the larger system

### Examples
- Common use cases
- Code snippets showing typical usage
- Edge cases worth noting

## Style Guidelines

### Markdown
- Use headers for organization
- Include code blocks with syntax highlighting
- Add links to related documentation

### JSDoc/Docstring
- Follow language conventions
- Include type annotations
- Add @example blocks

## Output

Generate complete documentation ready to use.
Focus on clarity over comprehensiveness.

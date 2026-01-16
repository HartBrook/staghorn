---
name: explain
description: Explain how code works in plain English
tags: [learning, documentation]
args:
  - name: path
    description: File or function to explain
    required: true
  - name: depth
    description: Level of detail
    default: medium
    options: [brief, medium, detailed]
---

# Code Explanation

Explain the code at `{{path}}` in plain English.

## Detail Level: {{depth}}

### Brief
- One paragraph summary of what the code does
- Main purpose and key behavior

### Medium
- How the code works step by step
- Important functions and their roles
- Data flow through the code

### Detailed
- Line-by-line explanation where helpful
- Why certain approaches were chosen
- Edge cases and error handling
- Performance characteristics

## Output

Explain as if teaching a developer who is new to this codebase.
Use analogies where they help clarify concepts.

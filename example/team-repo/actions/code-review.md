---
name: code-review
description: Perform a thorough code review
tags: [review, quality]
args:
  - name: path
    description: File or directory to review
    default: "."
  - name: focus
    description: What aspect to focus on
    default: all
    options: [all, logic, style, performance, security]
---

# Code Review

Review the code at `{{path}}` with focus on: **{{focus}}**

## Review Checklist

### Logic & Correctness
- Does the code do what it's supposed to do?
- Are edge cases handled?
- Are there any off-by-one errors?
- Is error handling appropriate?

### Code Quality
- Is the code readable and well-organized?
- Are functions and variables named clearly?
- Is there unnecessary duplication?
- Are comments helpful and accurate?

### Performance
- Are there any obvious performance issues?
- Are expensive operations cached when appropriate?
- Are there N+1 query problems?
- Is memory usage reasonable?

### Security
- Is user input validated?
- Are there potential injection vulnerabilities?
- Is sensitive data protected?

## Output Format

Provide feedback in these categories:
1. **Must Fix**: Issues that need to be addressed before merging
2. **Should Fix**: Improvements that would make the code better
3. **Consider**: Suggestions and minor improvements
4. **Praise**: Things done well (important for morale!)

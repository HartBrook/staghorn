---
name: code-review
description: Review code for correctness, clarity, and security
tags: [review]
args:
  - name: path
    description: File or directory to review
    default: "."
---

Review the code at `{{path}}` for correctness, clarity, and security issues.
Flag bugs and security problems as must-fix. Note style improvements separately.

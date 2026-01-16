---
name: debug
description: Help diagnose and fix a bug
tags: [debug, troubleshooting]
args:
  - name: symptom
    description: What's going wrong
    required: true
  - name: path
    description: Relevant file or area
    default: "."
---

# Bug Investigation

Investigate: **{{symptom}}**

Focus area: `{{path}}`

## Investigation Steps

1. **Reproduce**: Understand exactly when the bug occurs
2. **Isolate**: Narrow down where the problem originates
3. **Trace**: Follow the data/control flow
4. **Identify**: Find the root cause
5. **Fix**: Propose a solution

## What to Look For

- Off-by-one errors
- Null/undefined handling
- Race conditions
- State mutations
- Incorrect assumptions
- Missing error handling
- Type mismatches

## Output

Provide:
1. **Root Cause**: What's actually causing the issue
2. **Evidence**: Code references showing the problem
3. **Fix**: Suggested code changes
4. **Verification**: How to confirm the fix works
5. **Prevention**: How to prevent similar bugs

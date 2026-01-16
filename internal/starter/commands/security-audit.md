---
name: security-audit
description: Scan codebase for common security vulnerabilities
tags: [security, review, audit]
args:
  - name: path
    description: Directory or file to audit
    default: "."
  - name: severity
    description: Minimum severity level to report
    default: medium
    options: [low, medium, high, critical]
---

# Security Audit

Review the code at `{{path}}` for security vulnerabilities. Report issues at **{{severity}}** severity or higher.

## Check for these issues:

### Critical
- Hardcoded secrets, API keys, or credentials
- SQL injection vulnerabilities
- Command injection vulnerabilities
- Path traversal vulnerabilities

### High
- Cross-site scripting (XSS) vulnerabilities
- Insecure deserialization
- Missing authentication checks
- Broken access control

### Medium
- Sensitive data exposure in logs
- Missing input validation
- Insecure direct object references
- Security misconfiguration

### Low
- Missing security headers
- Verbose error messages
- Outdated dependencies with known vulnerabilities

## Output Format

For each issue found, provide:
1. **Location**: File and line number
2. **Severity**: Critical/High/Medium/Low
3. **Description**: What the vulnerability is
4. **Recommendation**: How to fix it

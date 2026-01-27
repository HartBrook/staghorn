---
name: security-audit
description: Scan codebase for common security vulnerabilities
tags: [security, audit]
allowed-tools: Read Grep Glob
context: fork
agent: Explore
args:
  - name: scope
    description: What to audit
    default: all
    options: [all, injection, auth, secrets, dependencies]
---

# Security Audit

Perform a security audit of the codebase, focusing on: **{{scope}}**

## Audit Areas

### Injection Vulnerabilities
- SQL injection (look for string concatenation in queries)
- Command injection (look for shell command construction)
- XSS vulnerabilities (look for unescaped user input in HTML)
- Path traversal (look for user input in file paths)

### Authentication & Authorization
- Hardcoded credentials or API keys
- Weak password requirements
- Missing or improper session management
- Insecure token handling
- Missing authorization checks

### Secrets & Configuration
- Secrets in source code or config files
- Sensitive data in logs
- Overly permissive file permissions
- Insecure default configurations

### Dependencies
- Known vulnerable dependencies
- Outdated packages with security patches
- Unnecessary dependencies increasing attack surface

## Output Format

Report findings by severity:

1. **Critical**: Immediate exploitation risk - fix before deploy
2. **High**: Significant security risk - fix soon
3. **Medium**: Security concern - address in next sprint
4. **Low**: Minor issue - fix when convenient
5. **Informational**: Best practice suggestion

For each finding include:
- Location (file and line number)
- Description of the vulnerability
- Potential impact
- Recommended fix

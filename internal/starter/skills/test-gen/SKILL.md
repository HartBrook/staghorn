---
name: test-gen
description: Generate unit tests for existing code
tags: [testing, quality]
allowed-tools: Read Grep Glob Write Edit
args:
  - name: path
    description: File or function to generate tests for
    required: true
  - name: framework
    description: Testing framework to use
    default: auto
    options: [auto, jest, vitest, pytest, go, rust]
  - name: coverage
    description: What to cover
    default: core
    options: [core, edge-cases, full]
---

# Test Generation

Generate unit tests for: `{{path}}`
Framework: **{{framework}}** | Coverage: **{{coverage}}**

## Test Generation Guidelines

### Test Structure
1. Read the source code to understand:
   - Function inputs and outputs
   - Side effects
   - Error conditions
   - Edge cases

2. Generate tests that cover:
   - Happy path (normal operation)
   - Edge cases (empty inputs, boundaries)
   - Error handling (invalid inputs, failures)
   - Integration points (if applicable)

### Framework-Specific Conventions

#### JavaScript/TypeScript (Jest/Vitest)
- Use `describe` blocks to group related tests
- Use `it` or `test` with descriptive names
- Use `beforeEach`/`afterEach` for setup/cleanup
- Mock external dependencies

#### Python (pytest)
- Use fixtures for test setup
- Use parametrize for multiple test cases
- Follow `test_` naming convention
- Use `@pytest.mark` for test categorization

#### Go
- Use table-driven tests
- Follow `Test` prefix convention
- Use `t.Run` for subtests
- Use `testify` assertions if available

#### Rust
- Use `#[test]` attribute
- Use `#[should_panic]` for error tests
- Use `#[ignore]` for slow tests
- Organize tests in a `tests` module

## Output Format

Generate tests that:
1. Are well-organized and readable
2. Have descriptive test names explaining what's tested
3. Include setup comments if complex
4. Follow the project's existing test patterns

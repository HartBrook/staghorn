---
name: test-gen
description: Generate unit tests for existing code
tags: [testing, quality]
args:
  - name: path
    description: File or function to test
    required: true
  - name: framework
    description: Test framework to use
    default: auto
    options: [auto, pytest, jest, go, rspec]
---

# Test Generation

Generate unit tests for the code at `{{path}}` using **{{framework}}** framework.

## Test Coverage Goals

1. **Happy Path**: Normal expected usage
2. **Edge Cases**: Boundary conditions, empty inputs, max values
3. **Error Cases**: Invalid inputs, expected failures
4. **Integration Points**: Mock external dependencies

## Guidelines

- Each test should test one thing
- Use descriptive test names that explain the scenario
- Follow AAA pattern: Arrange, Act, Assert
- Keep tests independent of each other
- Mock external services and I/O

## Output

Provide complete, runnable test code including:
- Necessary imports
- Test fixtures or setup
- Individual test cases with clear names
- Teardown if needed

Add comments explaining what each test verifies.

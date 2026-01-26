# Testing Standards

## Test Coverage

- Write tests for new functionality and bug fixes
- Test behavior, not implementation details
- Cover error paths and edge cases, not just happy paths
- Aim for meaningful coverage, not just high percentages

## Test Quality

- Keep tests deterministic; avoid flaky tests that depend on timing or external state
- Use descriptive test names that explain the scenario and expected outcome
- Isolate tests; each test should set up its own state
- Clean up test resources after each test
- Test one thing per test case

## Test Organization

- Colocate tests with the code they test when possible
- Group related tests logically using describe/context blocks
- Use test fixtures for complex setup scenarios
- Mock external dependencies at system boundaries
- Use factories or builders for test data creation

## Integration Tests

- Test critical user flows end-to-end
- Use realistic test data
- Test against actual dependencies when feasible
- Include performance benchmarks for critical paths

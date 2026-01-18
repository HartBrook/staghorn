# Writing and Debugging Evals

This guide covers how to write effective evals for your CLAUDE.md configs, interpret results, and debug failures.

## What Evals Test

Evals verify that your CLAUDE.md configuration produces the behavior you expect from Claude. They answer questions like:

- Does Claude follow my security guidelines?
- Does Claude use the coding patterns I specified?
- Does Claude respond in the style I configured?

Evals are **behavioral tests**, not unit tests. They test the emergent behavior that results from your system prompt, not specific code paths.

## Creating Evals

### Quick Start with Templates

The fastest way to create a new eval is with the `create` command:

```bash
# Interactive wizard
stag eval create

# Use a specific template
stag eval create --template security
stag eval create --template quality
stag eval create --template language
stag eval create --template blank

# Copy from an existing eval
stag eval create --from security-secrets --name my-security

# Save to project instead of personal directory
stag eval create --project

# Save to ./evals/ for team/community repos
stag eval create --team
```

**Destination options:**
- Default: `~/.config/staghorn/evals/` (personal evals)
- `--project`: `.staghorn/evals/` (project-specific evals)
- `--team`: `./evals/` (team/community evals for sharing via git)

Available templates:
- **security** — Tests for hardcoded secrets, injection vulnerabilities
- **quality** — Tests for naming conventions, code duplication
- **language** — Language-specific best practices template
- **blank** — Minimal template to start from scratch

### Validating Evals

Before running evals (which consume API credits), validate them:

```bash
# Validate all evals
stag eval validate

# Validate a specific eval
stag eval validate my-custom-eval
```

Validation checks for:
- Valid assertion types (`llm-rubric`, `contains`, `regex`, etc.)
- Required fields (`name`, `prompt`, `assert`)
- Proper YAML structure
- Naming conventions

Example output:
```
Validating 25 eval(s)...

✓ security-secrets (4 tests)
✓ security-injection (3 tests)
✗ my-custom-eval (2 tests)
  error: tests[0].assert[0].type: invalid assertion type "llm_rubric" (did you mean "llm-rubric"?)
  warning: tests[1]: test "check-patterns" should have a description

23 valid, 1 invalid, 1 warning
```

The validator provides helpful suggestions for common typos (e.g., `llm_rubric` → `llm-rubric`).

## Anatomy of an Eval

```yaml
name: security-secrets
description: Verify config detects and warns about hardcoded secrets
tags: [security, critical]

# Optional: specify which config layers to test
context:
  layers: [merged]  # team, personal, project, or merged (default)
  languages: [python]  # include these language configs

# Optional: override the default model
provider:
  model: ${STAGHORN_EVAL_MODEL:-claude-sonnet-4-20250514}

tests:
  - name: warns-about-api-keys
    description: Should warn when code contains hardcoded API keys
    prompt: |
      Review this code:
      ```python
      API_KEY = "<OPENAI_API_KEY_HERE>"
      client = OpenAI(api_key=API_KEY)
      ```
    assert:
      - type: llm-rubric
        value: Response must warn about hardcoded API key and suggest environment variables
```

### Required Fields

| Field | Description |
|-------|-------------|
| `name` | Unique identifier for the eval (used in CLI) |
| `tests` | Array of test cases |
| `tests[].name` | Unique name for the test |
| `tests[].prompt` | The user message to send to Claude |
| `tests[].assert` | Array of assertions to validate the response |

### Optional Fields

| Field | Description |
|-------|-------------|
| `description` | Human-readable description |
| `tags` | Array of tags for filtering (`--tag security`) |
| `context.layers` | Config layers to test against |
| `context.languages` | Language configs to include |
| `provider.model` | Override the default model |
| `tests[].description` | Description shown in output |
| `tests[].vars` | Custom variables for the prompt |

## Assertion Types

### `llm-rubric` (Recommended for Most Cases)

Uses an LLM to grade whether the response meets a criterion. Most flexible and human-like evaluation.

```yaml
assert:
  - type: llm-rubric
    value: Response should include type hints in function signatures
```

**Best for:**
- Subjective quality checks
- Style and tone verification
- Complex behavioral requirements

**Tips:**
- Be specific about what you're looking for
- Avoid vague criteria like "good code"
- Include examples of what passes/fails if ambiguous

### `contains`

Checks if the response contains an exact string (case-sensitive).

```yaml
assert:
  - type: contains
    value: "def factorial"
```

**Best for:**
- Verifying specific syntax or keywords
- Checking for required imports or patterns

### `contains-any`

Passes if the response contains any of the listed strings.

```yaml
assert:
  - type: contains-any
    value: ["pytest", "unittest", "test_"]
```

**Best for:**
- Checking for one of several acceptable patterns
- Flexible keyword matching

### `contains-all`

Passes only if the response contains all listed strings.

```yaml
assert:
  - type: contains-all
    value: ["import os", "os.environ", "API_KEY"]
```

**Best for:**
- Ensuring multiple required elements are present

### `not-contains`

Passes if the response does NOT contain the string.

```yaml
assert:
  - type: not-contains
    value: "password123"
```

**Best for:**
- Security checks (no hardcoded secrets)
- Ensuring deprecated patterns aren't used

### `regex`

Matches against a regular expression.

```yaml
assert:
  - type: regex
    value: "def \\w+\\(.*: .*\\) -> .*:"
```

**Best for:**
- Pattern matching with flexibility
- Validating specific formats

### `javascript`

Custom JavaScript assertion function for complex logic.

```yaml
assert:
  - type: javascript
    value: |
      output.includes('async') && !output.includes('callback')
```

**Best for:**
- Complex conditional logic
- Custom validation that other types can't handle

## Writing Effective Tests

### 1. Test One Thing at a Time

**Bad:**
```yaml
- name: good-code
  prompt: Write a Python function
  assert:
    - type: llm-rubric
      value: Code should have type hints, docstrings, error handling, and tests
```

**Good:**
```yaml
- name: uses-type-hints
  prompt: Write a Python function to calculate factorial
  assert:
    - type: llm-rubric
      value: Function should include type hints

- name: includes-docstring
  prompt: Write a Python function to calculate factorial
  assert:
    - type: llm-rubric
      value: Function should have a docstring explaining its purpose
```

### 2. Be Specific in Prompts

**Bad:**
```yaml
prompt: Write some code
```

**Good:**
```yaml
prompt: |
  Write a Python function that reads a configuration file
  and returns the settings as a dictionary.
```

### 3. Use Concrete Examples in Assertions

**Bad:**
```yaml
assert:
  - type: llm-rubric
    value: Should handle errors properly
```

**Good:**
```yaml
assert:
  - type: llm-rubric
    value: Should wrap file operations in try/except and handle FileNotFoundError
```

### 4. Test Both Positive and Negative Cases

```yaml
tests:
  - name: recommends-env-vars
    prompt: How should I store my API key?
    assert:
      - type: contains-any
        value: ["environment variable", "os.environ", ".env"]

  - name: warns-against-hardcoding
    prompt: |
      Is this okay?
      API_KEY = "sk-secret123"
    assert:
      - type: llm-rubric
        value: Should warn against hardcoding secrets
```

### 5. Match Test Complexity to Config Complexity

If your CLAUDE.md just says "use type hints", a simple `contains` assertion works:

```yaml
assert:
  - type: contains
    value: ") ->"
```

If your CLAUDE.md has nuanced guidelines, use `llm-rubric`:

```yaml
assert:
  - type: llm-rubric
    value: Type hints should use modern Python 3.10+ syntax (X | Y not Union[X, Y])
```

## Debugging Failed Tests

### Step 0: Validate First

Before debugging runtime failures, ensure your eval is valid:

```bash
stag eval validate my-eval
```

This catches common issues like typos in assertion types (e.g., `llm_rubric` instead of `llm-rubric`) without making API calls.

### Step 1: Run with `--debug`

```bash
stag eval security-secrets --debug
```

This shows:
- Full Claude response for failed tests
- Path to preserved temp files

### Step 2: Check the Response

Look at what Claude actually said. Common issues:

| Symptom | Likely Cause |
|---------|--------------|
| Response is correct but test fails | Assertion is too strict |
| Response ignores your guidelines | Guidelines aren't in the tested config layer |
| Response is generic/unhelpful | Prompt is too vague |

### Step 3: Inspect Debug Artifacts

The debug directory contains:

```
/tmp/staghorn-eval-xxx/eval-xxx/
├── promptfooconfig.yaml  # Generated Promptfoo config
└── output.json           # Full results with all responses
```

**View the generated config:**
```bash
cat /tmp/staghorn-eval-xxx/eval-xxx/promptfooconfig.yaml
```

This shows exactly what was sent to Promptfoo, including the system prompt (your CLAUDE.md).

**View full results:**
```bash
cat /tmp/staghorn-eval-xxx/eval-xxx/output.json | jq '.results[0].response.output'
```

### Step 4: Test Specific Cases

Run a single test to iterate quickly:

```bash
stag eval security-secrets --test warns-about-api-keys --debug
```

### Step 5: Check Your Config Layer

If tests pass with `--layer merged` but fail with `--layer team`, your team config might be missing guidelines.

```bash
# See what config is being tested
stag info --layer team --content
```

## Common Debugging Scenarios

### "Test passes locally but fails in CI"

**Possible causes:**
1. Different config layers (CI might not have personal config)
2. Model differences (check `STAGHORN_EVAL_MODEL`)
3. Cached responses (Promptfoo caches by default)

**Fix:**
```bash
# Test the exact layer CI uses
stag eval --layer team

# Clear Promptfoo cache
npx promptfoo cache clear
```

### "llm-rubric assertions are inconsistent"

LLM-based grading has some variance. To reduce flakiness:

1. Make rubrics more specific
2. Use multiple assertions
3. Combine with deterministic checks

```yaml
assert:
  # Deterministic check
  - type: contains
    value: "os.environ"
  # LLM check for nuance
  - type: llm-rubric
    value: Explains why environment variables are more secure
```

### "Response is correct but doesn't match assertion"

The response might use different wording. Use flexible assertions:

```yaml
# Instead of exact match
- type: contains
  value: "use environment variables"

# Use alternatives
- type: contains-any
  value: ["environment variable", "env var", "os.environ", "os.getenv"]
```

### "Test expects behavior not in my config"

Starter evals test for common best practices. If your config doesn't enforce a behavior, either:

1. **Add the guideline** to your CLAUDE.md
2. **Remove the test** if you don't want that behavior
3. **Modify the test** to match your actual guidelines

## Eval Organization

### By Category

```
evals/
├── security-secrets.yaml
├── security-injection.yaml
├── quality-naming.yaml
├── quality-simplicity.yaml
└── lang-python.yaml
```

### By Feature

```
evals/
├── authentication.yaml   # All auth-related tests
├── api-design.yaml       # API guidelines
└── error-handling.yaml   # Error handling patterns
```

### Tags for Filtering

Use tags to run subsets of evals:

```yaml
tags: [security, critical, ci]
```

```bash
stag eval --tag critical    # Run only critical tests
stag eval --tag security    # Run security tests
stag eval --tag ci          # Run tests suitable for CI
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Eval Config
on: [push, pull_request]

jobs:
  eval:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install staghorn
        run: |
          brew tap HartBrook/tap
          brew install staghorn

      - name: Install Promptfoo
        run: npm install -g promptfoo

      - name: Run evals
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: stag eval --output github --tag ci
```

### Recommended CI Strategy

1. **Tag tests for CI** - Not all tests need to run on every commit
2. **Use `--layer team`** - Test the shared config, not personal additions
3. **Cache results** - Promptfoo caches responses by default
4. **Set budget alerts** - Evals consume API credits

```yaml
# Run fast, critical tests on every commit
stag eval --tag ci --layer team

# Run full suite on main branch or manually
stag eval --layer team
```

## Cost Optimization

Each test case = one API call. To minimize costs:

1. **Use cheaper models for development:**
   ```bash
   export STAGHORN_EVAL_MODEL=claude-3-haiku-20240307
   stag eval
   ```

2. **Run specific tests while iterating:**
   ```bash
   stag eval lang-python --test uses-type-hints
   ```

3. **Use `--dry-run` to preview:**
   ```bash
   stag eval --dry-run
   ```

4. **Leverage Promptfoo caching** - Repeated runs with same prompts use cached responses

## Quick Reference

### Commands

```bash
# Run evals
stag eval                           # Run all evals
stag eval security-secrets          # Run specific eval
stag eval --tag security            # Filter by tag
stag eval --test "warns-*"          # Filter by test name pattern
stag eval --debug                   # Show full responses

# Create and validate
stag eval create                    # Interactive wizard
stag eval create --template security
stag eval create --project          # Save to .staghorn/evals/
stag eval create --team             # Save to ./evals/ for team sharing
stag eval validate                  # Validate all evals
stag eval validate my-eval          # Validate specific eval

# List and inspect
stag eval list                      # List available evals
stag eval info security-secrets     # Show eval details
stag eval init                      # Install starter evals
```

### Valid Assertion Types

| Type           | Description                          |
|----------------|--------------------------------------|
| `llm-rubric`   | AI-graded evaluation (most flexible) |
| `contains`     | Exact string match                   |
| `contains-any` | Any of the listed strings            |
| `contains-all` | All of the listed strings            |
| `not-contains` | String must not appear               |
| `regex`        | Regular expression match             |
| `javascript`   | Custom JS assertion function         |

## Further Reading

- [Promptfoo Documentation](https://promptfoo.dev/docs/intro)
- [Promptfoo Assertion Types](https://promptfoo.dev/docs/configuration/expected-outputs/)
- [Anthropic Prompt Engineering Guide](https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering)

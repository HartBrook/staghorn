## Python Guidelines

- Use type hints for all function signatures
- Prefer f-strings over `.format()` or `%` formatting
- Use `pytest` for testing
- Follow PEP 8 style guide
- Use `ruff` for linting and formatting

## Dependencies

- Use `uv` or `poetry` for dependency management
- Pin dependency versions in production
- Keep `pyproject.toml` as the single source of truth

## Project Structure

```
src/
  package_name/
    __init__.py
    ...
tests/
  test_*.py
pyproject.toml
```

## Common Commands

- Run tests: `pytest`
- Run tests with coverage: `pytest --cov`
- Format code: `ruff format .`
- Lint code: `ruff check .`

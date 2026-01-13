## Rust Guidelines

- Follow the Rust API Guidelines
- Use `rustfmt` for formatting
- Use `clippy` for linting
- Prefer safe code; document any `unsafe` blocks thoroughly

## Error Handling

- Use `Result<T, E>` for recoverable errors
- Use `?` operator for error propagation
- Create custom error types with `thiserror`
- Use `anyhow` for application-level errors

## Project Structure

```
src/
  lib.rs        # Library crate
  main.rs       # Binary crate (if applicable)
  module/
    mod.rs
    ...
tests/          # Integration tests
Cargo.toml
```

## Dependencies

- Keep dependencies minimal
- Audit dependencies for security issues
- Use workspace for multi-crate projects

## Common Commands

- Run tests: `cargo test`
- Build: `cargo build --release`
- Lint: `cargo clippy`
- Format: `cargo fmt`
- Check: `cargo check`

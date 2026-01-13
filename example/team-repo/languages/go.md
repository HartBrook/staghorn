## Go Guidelines

- Follow Effective Go and Go Code Review Comments
- Use `gofmt` and `goimports` for formatting
- Keep package names short and lowercase
- Prefer composition over inheritance

## Error Handling

- Always handle errors explicitly
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use custom error types for domain errors
- Never ignore errors silently

## Project Structure

```
cmd/
  appname/
    main.go
internal/
  package/
    ...
pkg/           # Only for truly reusable code
go.mod
go.sum
```

## Testing

- Use table-driven tests
- Place tests in the same package (`_test.go` files)
- Use `testify/assert` for assertions when helpful
- Run tests: `go test ./...`

## Common Commands

- Run tests: `go test ./...`
- Run tests with coverage: `go test -cover ./...`
- Build: `go build ./cmd/...`
- Lint: `golangci-lint run`

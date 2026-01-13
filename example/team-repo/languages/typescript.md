## TypeScript Guidelines

- Enable strict mode in `tsconfig.json`
- Prefer `interface` over `type` for object shapes
- Use `readonly` for immutable data
- Avoid `any` - use `unknown` when type is truly unknown

## Naming Conventions

- PascalCase for types, interfaces, and classes
- camelCase for variables, functions, and methods
- SCREAMING_SNAKE_CASE for constants
- Prefix interfaces with `I` only when necessary to avoid conflicts

## Imports

- Use absolute imports with path aliases
- Group imports: external, internal, relative
- Avoid circular dependencies

## React (if applicable)

- Use functional components with hooks
- Prefer named exports over default exports
- Keep components small and focused
- Use TypeScript generics for reusable components

## Common Commands

- Run tests: `npm test` or `pnpm test`
- Type check: `tsc --noEmit`
- Lint: `eslint .`
- Format: `prettier --write .`

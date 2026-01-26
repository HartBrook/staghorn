---
paths:
  - "src/components/**/*.tsx"
  - "src/components/**/*.jsx"
  - "src/pages/**/*.tsx"
  - "src/pages/**/*.jsx"
  - "app/**/*.tsx"
  - "app/**/*.jsx"
---
# React Guidelines

## Component Design

- Use functional components with hooks
- Keep components small and focused (under 100 lines)
- Extract reusable logic into custom hooks
- Prefer composition over inheritance
- Use TypeScript for type safety

## Props and State

- Define explicit prop types with TypeScript interfaces
- Keep state as local as possible
- Lift state up only when necessary for sharing
- Use context sparingly for truly global state
- Consider state management libraries for complex state

## Performance

- Memoize expensive calculations with useMemo
- Memoize callbacks passed to children with useCallback
- Use React.memo for components that render often with same props
- Avoid inline object/array creation in render
- Lazy load components and routes when appropriate

## Accessibility

- Use semantic HTML elements
- Include proper ARIA attributes when needed
- Ensure keyboard navigation works
- Maintain sufficient color contrast
- Test with screen readers

## Testing

- Test component behavior, not implementation
- Use React Testing Library for component tests
- Test user interactions and accessibility
- Colocate tests with components

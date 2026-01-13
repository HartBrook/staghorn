# Project Guidelines

This is a React frontend application.

## Architecture

- Use functional components with hooks
- Organize by feature, not by type
- Keep components small and focused
- Use React Query or SWR for server state

## State Management

- Use React Context for global UI state
- Use React Query for server state
- Avoid prop drilling more than 2 levels

## Styling

- Use CSS modules or styled-components
- Follow mobile-first responsive design
- Use design system tokens for colors/spacing

## Testing

- Write unit tests for utility functions
- Write component tests with React Testing Library
- Write E2E tests for critical user flows

## Performance

- Lazy load routes and heavy components
- Memoize expensive computations
- Use React.memo sparingly and intentionally

## Accessibility

- Use semantic HTML elements
- Include ARIA labels where needed
- Ensure keyboard navigation works
- Test with screen readers

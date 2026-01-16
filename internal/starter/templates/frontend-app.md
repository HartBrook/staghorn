# Frontend Application Template

This template is for frontend web applications.

## Project Structure

Follow these conventions for organizing frontend apps:
- `/src/components` - Reusable UI components
- `/src/pages` or `/src/views` - Page-level components
- `/src/hooks` - Custom React/Vue hooks
- `/src/utils` - Utility functions
- `/src/styles` - Global styles and themes

## Component Guidelines

- Keep components small and focused
- Extract reusable logic into hooks
- Use TypeScript for type safety
- Document component props

## State Management

- Keep state as local as possible
- Lift state only when necessary
- Use context for truly global state
- Consider server state vs client state

## Styling

- Use consistent spacing and sizing
- Follow the design system
- Support dark mode where applicable
- Ensure responsive design

## Accessibility

- Use semantic HTML elements
- Add ARIA labels where needed
- Ensure keyboard navigation works
- Test with screen readers

## Performance

- Lazy load routes and heavy components
- Optimize images and assets
- Minimize bundle size
- Use appropriate caching strategies

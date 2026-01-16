---
name: api-design
description: Design or review an API interface
tags: [api, design, architecture]
args:
  - name: resource
    description: The resource or feature to design
    required: true
  - name: style
    description: API style
    default: rest
    options: [rest, graphql, grpc, cli]
---

# API Design

Design a **{{style}}** API for: **{{resource}}**

## Design Principles

- Consistency with existing APIs
- Intuitive naming and structure
- Appropriate error handling
- Versioning strategy
- Authentication/authorization

## REST Guidelines
- Use nouns for resources
- HTTP methods match operations (GET, POST, PUT, DELETE)
- Consistent response format
- Proper status codes

## GraphQL Guidelines
- Clear type definitions
- Efficient query structure
- Mutation naming conventions
- Error handling patterns

## CLI Guidelines
- Intuitive command hierarchy
- Consistent flag naming
- Helpful error messages
- Good defaults

## Output

Provide:
1. **Endpoints/Commands**: Full specification
2. **Request/Response**: Example payloads
3. **Errors**: Common error cases and responses
4. **Examples**: Usage examples with curl/code

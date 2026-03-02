---
name: api-documenter
description: Analyze backend codebases to generate comprehensive API documentation. Includes endpoint details, request/response examples, and practical use cases. Use this skill when the user wants to document an existing API or Backend project, or update existing documentation.
---

# API Documenter

## Overview

This skill analyzes backend source code to generate detailed, structured API documentation. It focuses on extracting not just the schema, but the *intent* and *usage* of each endpoint.

## Capabilities

1.  **Code Analysis**: Identifies routes, controllers, and models in various frameworks (Express, Flask, Django, Laravel, Fastify, etc.).
2.  **Schema Extraction**: Infers request bodies, query parameters, and response structures from code logic and validation schemas (e.g., Zod, Joi, Pydantic).
3.  **Use Case Generation**: Derives practical use cases based on the endpoint's logic and context.
4.  **Example formatting**: Generates realistic JSON examples for requests and responses.

## Workflow

### 1. Discovery Phase
*   **Identify Framework**: specific framework patterns (e.g., `@Controller` in NestJS, `app.get` in Express, `router.post` in Go).
*   **Locate Entry Points**: specific root files (e.g., `app.js`, `server.py`, `routes/api.php`).
*   **Map Routes**: Create a list of all available endpoints and their HTTP methods.

### 2. Analysis Phase (Per Endpoint)
For each identified endpoint:
1.  **Trace Execution**: Follow the handler function to understand the flow.
2.  **Identify Inputs**: Look for query params, path params, and body parsing.
3.  **Identify Outputs**: Look for `return`, `res.json()`, `jsonify()`, or serializer usage.
4.  **Infer Use Cases**: Ask "Why would a client call this?" based on the operation (e.g., "Create Order" implies "User purchasing items").

### 3. Generation Phase
*   Use the [API Template](references/api_template.md) as the structure.
*   Populate details for every endpoint.
*   Ensure every JSON example is valid and representative.

## Usage Guidelines

*   **Be Comprehensive**: Do not skip error states (400, 401, 403, 404, 500). Document what triggers them.
*   **Infer Types**: If explicit types aren't available, infer them from usage (e.g., `req.body.email` implies a string field `email`).
*   **Contextual Examples**: If the API is about "Books", use book-related data in examples, not "foo/bar".

## Reference

See [references/api_template.md](references/api_template.md) for the strictly required output format.

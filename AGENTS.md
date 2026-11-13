
# HALISI — Codex Review Instructions

## Project Overview

HALISI is a Go-based LPG marketplace and delivery platform connecting customers, verified gas vendors, and delivery riders.

The system includes:
- Customer registration and authentication
- Vendor onboarding and KYC verification
- LPG product listings and inventory
- Customer orders and order tracking
- Payments and delivery management
- Rider onboarding and verification
- Administrative management

## Technology Stack

- Backend: Go
- Database: PostgreSQL
- Database driver: pgx/pgxpool
- API: RESTful HTTP endpoints
- Deployment: Render
- Version control: GitHub

Review code against the actual dependencies and architecture present in the repository. Do not assume unimplemented features exist.

## Code Quality

- Follow idiomatic Go conventions.
- Prefer clear, maintainable code.
- Keep handlers, business logic, and database access appropriately separated.
- Avoid unnecessary dependencies and abstractions.
- Check error handling and propagation.
- Ensure resources are properly closed.
- Check for race conditions and goroutine leaks.
- Avoid logging sensitive information.

## Database

- Use parameterized SQL queries.
- Prevent SQL injection.
- Validate database inputs.
- Use transactions for operations requiring atomicity.
- Verify foreign keys and data integrity.
- Ensure migrations are reversible where appropriate.
- Check indexes for frequently queried columns.
- Propagate context to database operations.
- Handle database failures gracefully.

## Authentication and Authorization

- Validate authentication on protected endpoints.
- Enforce role-based access control.
- Prevent customers, vendors, and riders from accessing unauthorized resources.
- Verify resource ownership.
- Protect administrator-only operations.
- Handle tokens and sessions securely.
- Never expose credentials or sensitive authentication data.

## KYC and Document Security

- Protect identity documents and personal information.
- Restrict access to verification documents.
- Validate document uploads.
- Prevent unauthorized document downloads.
- Require administrative approval where specified.
- Review vendor and rider verification workflows for authorization bypasses.

## Payments and Orders

- Prevent duplicate payment processing.
- Validate payment callbacks and transaction references.
- Ensure payment amounts cannot be manipulated by clients.
- Enforce valid order status transitions.
- Review concurrent inventory updates.
- Ensure failed operations do not leave inconsistent records.

## Delivery and Maps

- Validate rider assignments.
- Enforce appropriate rider verification requirements.
- Protect location information.
- Restrict access to delivery tracking data.
- Handle external mapping API failures safely.

## Testing

- Require meaningful tests for new functionality.
- Review edge cases and failure scenarios.
- Check authorization and validation tests.
- Verify database transaction behavior.
- Check for regressions.
- Do not claim tests passed unless they were actually executed.

## Pull Request Reviews

Prioritize:
1. Security vulnerabilities
2. Data loss and database corruption
3. Authentication and authorization bypasses
4. Payment and order integrity
5. Concurrency and reliability problems
6. Incorrect business logic
7. Missing critical tests
8. Maintainability concerns

For every actionable finding:
- Identify the affected file and line.
- Explain the actual risk.
- Describe how the issue can occur.
- Suggest a practical fix.
- Indicate severity.

Avoid speculative findings, unnecessary stylistic complaints, and unrelated refactoring.

Only report issues introduced by the pull request or materially worsened by it.
Do regression tests on every push.

## Git Workflow

- `main` is the stable release branch.
- `develop` is the integration branch.
- Feature branches are created for individual tasks.
- Feature branches merge into `develop`.
- `develop` merges into `main` when a release is ready.

Review changes against the pull request's target branch.

Do not require feature branches to merge directly into `main`.

## Review Principles

- Prioritize correctness and security.
- Respect the scope of the issue being implemented.
- Do not request unrelated features.
- Avoid introducing unnecessary complexity.
- Provide constructive, specific feedback.
- Do not approve unsafe implementations merely because tests pass.

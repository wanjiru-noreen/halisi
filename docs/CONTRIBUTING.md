# Contributing to Halisi

Thank you for your interest in contributing to Halisi! This guide explains how to set up your development environment, follow the project's conventions, and submit changes.

## Table of Contents

- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Adding New Features](#adding-new-features)
- [Database Changes](#database-changes)

## Development Workflow

1. **Fork** the repository.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/halisi.git
   cd halisi
   ```
3. **Add** the upstream remote:
   ```bash
   git remote add upstream https://github.com/<original-owner>/halisi.git
   ```
4. **Create** a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
5. **Make** your changes following the guidelines below.
6. **Run** tests to ensure nothing is broken:
   ```bash
   go test ./...
   ```
7. **Commit** your changes with clear messages.
8. **Push** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
9. **Open** a Pull Request against the `main` branch.

## Code Style

### Go Code

- Follow the [Effective Go](https://go.dev/doc/effective_go) guidelines.
- Run `gofmt` to format code before committing:
  ```bash
  gofmt -w .
  ```
- Use `go vet` to check for common issues:
  ```bash
  go vet ./...
  ```
- Import order: standard library → third-party → local imports, with blank lines between groups.
- Use meaningful variable and function names. Avoid abbreviations unless they are widely understood (e.g., `id`, `url`, `http`).

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Packages | lowercase, short | `handlers`, `services`, `middleware` |
| Structs | PascalCase, noun | `UserService`, `Order`, `KYCDocument` |
| Functions | PascalCase (exported), camelCase (unexported) | `CreateOrder`, `validateInput` |
| Variables | camelCase | `userID`, `shopName` |
| Constants | UPPER_SNAKE_CASE | `MaxUploadSize` |
| Files | snake_case | `order_service.go`, `auth.go` |

### HTML Templates

- Use semantic HTML5 elements.
- Keep templates simple — put business logic in handlers and services, not in templates.
- Use template actions sparingly; prefer passing pre-computed data from handlers.

### CSS

- Follow the existing CSS structure in `web/static/css/`.
- Use descriptive class names (BEM-style preferred).

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code formatting (no logic change)
- `refactor`: Code restructuring (no feature change)
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(handlers): add order cancellation endpoint
fix(services): handle nil shop in order creation
docs(api): update KYC document review endpoint
refactor(middleware): extract role check into helper
```

## Pull Request Process

1. **Update** your branch with the latest `main`:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```
2. **Ensure** all tests pass:
   ```bash
   go test ./...
   ```
3. **Fill out** the PR template with:
   - A clear description of the changes.
   - Screenshots for UI changes.
   - Steps to test the changes.
4. **Request** a review from a maintainer.
5. **Address** any review feedback.
6. PRs are squash-merged to keep the history clean.

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/services/...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Writing Tests

- Place test files alongside the code they test (e.g., `service_test.go` next to `service.go`).
- Use table-driven tests for multiple test cases.
- Test both happy paths and error cases.
- Use meaningful test names that describe the scenario:
  ```go
  func TestCreateOrder_InvalidShopID_ReturnsError(t *testing.T) {
      // ...
  }
  ```

### Example Table-Driven Test

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {
            name:    "valid email",
            email:   "user@example.com",
            wantErr: false,
        },
        {
            name:    "missing @ symbol",
            email:   "userexample.com",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

## Adding New Features

When adding a new feature, follow the layered architecture:

1. **Model** (`internal/models/`): Define the Go struct if a new entity is needed.
2. **Schema** (`internal/database/schema.sql`): Add the database table or columns.
3. **Queries**: Add SQL queries for sqlc to generate.
4. **Repository** (`internal/repository/`): Implement data access methods.
5. **Service** (`internal/services/`): Implement business logic.
6. **Handler** (`internal/handlers/`): Implement the HTTP handler.
7. **Routes** (`cmd/api/main.go`): Register the new route with appropriate middleware.
8. **Template** (`web/templates/`): Create the HTML template.
9. **Tests**: Add tests for the new functionality.
10. **Documentation**: Update relevant docs (API.md, DATABASE.md, etc.).

## Database Changes

1. Update `internal/database/schema.sql` with the new schema.
2. Add `ALTER TABLE` statements to `internal/database/database.go`'s `createTables()` function for existing databases.
3. Add SQL queries in `internal/database/queries/` for new operations.
4. Run `sqlc generate` to regenerate the Go code.
5. Update the repository layer to use the new generated queries.
6. Update `docs/DATABASE.md` to reflect the schema changes.

## Code of Conduct

- Be respectful and inclusive.
- Provide constructive feedback.
- Help others learn and grow.
- Focus on what is best for the community and the project.

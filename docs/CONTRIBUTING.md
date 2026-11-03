# Contributing to Halisi

Thank you for contributing to Halisi! This document outlines our development workflows, coding standards, branch conventions, and pull request procedures.

---

## 📌 Table of Contents

- [Branching Strategy & Workflow](#branching-strategy--workflow)
- [Local Setup & Pre-requisites](#local-setup--pre-requisites)
- [Code Generation (Templ & sqlc)](#code-generation-templ--sqlc)
- [Coding Conventions & Standards](#coding-conventions--standards)
- [Commit Message Conventions](#commit-message-conventions)
- [Testing & Quality Verification](#testing--quality-verification)
- [Pull Request Guidelines](#pull-request-guidelines)

---

## 🌿 Branching Strategy & Workflow

Halisi adheres to a structured branching model:

* **`main`**: Production-ready, stable releases. Never commit directly to `main`.
* **`development`**: Integration branch for new features and bug fixes. All feature pull requests must target `development`.
* **`feature/<feature-name>`**: New functionality (e.g. `feature/daraja-stk-push`, `feature/google-maps-pins`).
* **`fix/<bug-description>`**: Bug fixes (e.g. `fix/templ-brace-parsing`).
* **`chore/<task-name>`**: Refactoring, tool updates, or infrastructure maintenance.

### Typical Contribution Steps
1. Fork the repository and clone your fork locally.
2. Ensure your local `development` branch is up to date:
   ```bash
   git checkout development
   git pull origin development
   ```
3. Create your feature branch:
   ```bash
   git checkout -b feature/my-feature-name
   ```
4. Implement your changes and write unit tests.
5. Verify build, tests, and formatting locally with `make check`.
6. Push to your fork and submit a PR targeting `development`.

---

## 💻 Local Setup & Pre-requisites

Ensure you have installed:
* **Go 1.26+**
* **Docker & Docker Compose**
* **templ CLI**: `go install github.com/a-h/templ/cmd/templ@latest`
* **sqlc CLI**: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

Start local database and mail services:
```bash
docker compose up -d postgres mailpit
```

---

## ⚙️ Code Generation (Templ & sqlc)

Halisi relies on compile-time code generation for both frontend components and database access:

### 1. Templ Components
Whenever modifying `.templ` files in `web/templ/`:
```bash
templ generate
```
Never manually edit generated `*_templ.go` files; always edit the corresponding `.templ` file and re-run generation.

### 2. SQL Queries (sqlc)
Whenever adding or modifying SQL queries in `internal/database/queries/` or updating `schema.sql`:
```bash
sqlc generate
```
All database queries must be type-safe and compiled into `internal/database/queried/`. Do not write raw SQL strings in handlers or repositories.

---

## 📐 Coding Conventions & Standards

### Go Standards
* Adhere to [Effective Go](https://go.dev/doc/effective_go) and [Go CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments).
* Always run `gofmt` on modified Go files.
* Propagate `context.Context` through HTTP handlers, domain services, and database queries.
* Use structured logging with standard library `log/slog` rather than unstructured `log.Print`.
* Never ignore returned errors; handle them explicitly or wrap them with context: `fmt.Errorf("failed to fetch order %d: %w", orderID, err)`.

### Templ & Frontend Standards
* Keep component logic declarative. Business calculations belong in `internal/services/`, not inside `.templ` expressions.
* Pass strongly-typed structs from `internal/templdata/` to components instead of anonymous maps.
* Ensure all form inputs include CSRF protection tokens and mobile-responsive CSS classes.

---

## 📝 Commit Message Conventions

Halisi strictly enforces [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <short description in imperative present tense>

[optional detailed body]

[optional footer: Closes #12]
```

### Allowed Types
* `feat`: A new user-facing feature or domain capability.
* `fix`: A bug fix.
* `docs`: Documentation improvements or updates.
* `style`: White-space, formatting, missing semi-colons (no code logic changes).
* `refactor`: Code reorganization without fixing bugs or adding features.
* `perf`: Performance optimizations.
* `test`: Adding or updating test cases.
* `chore`: Build tooling, dependency upgrades, or CI workflow changes.

### Examples
* `feat(payments): implement Safaricom Daraja STK push client`
* `fix(templ): resolve missing closing brace in admin template`
* `docs(readme): add environment variables and quickstart guide`
* `chore(ci): update GitHub Actions workflow for PostgreSQL 16`

---

## 🧪 Testing & Quality Verification

Before submitting a pull request, run the comprehensive verification suite via the `Makefile`:

```bash
# 1. Format code
make fmt

# 2. Check formatting, module hygiene, and vet
make quality

# 3. Run race-detector unit and repository tests
make test

# 4. Check for known vulnerabilities in dependencies
make security

# 5. Run all checks at once
make check
```

Pull requests will automatically trigger the GitHub Actions CI pipeline, which enforces that `make check` passes without warnings.

---

## 🚀 Pull Request Guidelines

1. **Title**: Follow the Conventional Commit format (e.g. `feat(maps): add Google Maps interactive shop view`).
2. **Description**:
   * Explain the *why* and *what* of the change.
   * Include screenshots or screen recordings for UI modifications.
   * Reference the related GitHub issue number (e.g. `Closes #29`).
3. **Scope**: Keep pull requests focused on a single concern or issue. Avoid mixing unrelated refactors into feature PRs.
4. **Review**: All PRs require review from a project maintainer before being merged into `development`.

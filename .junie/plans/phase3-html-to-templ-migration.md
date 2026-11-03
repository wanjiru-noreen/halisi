---
sessionId: session-261101-234226-yxd3
---

# Requirements

### Overview
Migrate all 14 HTML templates in `web/templates/` to type-safe `.templ` components in `web/templ/`, and update `internal/handlers/handlers.go` to render via templ instead of `html/template`. This is **Phase 3** of the migration plan (Phase 1: SQLite→PostgreSQL ✓, Phase 2: SQL→sqlc ✓).

### Goals
- Replace `html/template` rendering with templ-generated components
- Gain compile-time type safety for all template data bindings
- Preserve all existing HTML output, CSS, and JavaScript behavior
- Remove the `renderTemplate()` helper and `template.ParseFiles()` calls

### Scope
**In Scope:**
- Convert 13 page templates + 2 navbar partials to `.templ` files
- Update `handlers.go` to call templ components instead of `renderTemplate()`
- Remove `html/template` import and the `renderTemplate` helper
- Remove HTML files after migration

**Out of Scope:**
- Changing any UI design, CSS, or JavaScript
- Modifying business logic in services or repositories
- Adding new features

# Technical Design

### Current Implementation

**Templates** (`web/templates/`): 14 HTML files using `{{define}}`/`{{template}}` for partials, `{{range}}`, `{{if}}`, and `{{.Field}}` bindings.

**Rendering** (`internal/handlers/handlers.go`):
- `renderTemplate(w, name, data, files...)` parses HTML at request time via `template.ParseFiles()` and executes via `ExecuteTemplate()`.
- 12 call sites: `Register`, `RenderOrderPage`, `Home`, `Login`, `Dashboard`, `Shops`, `Shop`, `Orders`, `OrderConfirmation`, `AdminDashboard`, `ShopOrders`, `ShopManage`.

**Data types passed to templates:**

| Template | Data type |
|---|---|
| `index.html` | `nil` (static) |
| `login.html` | `nil` (static) |
| `register.html` | `nil` (static) |
| `dashboard.html` | `nil` (static, 901 lines) |
| `shops.html` | `struct{ Shops []models.Shop }` + navbar |
| `shop.html` | `models.Shop` + navbar |
| `order.html` | `struct{ Shops []models.Shop, SelectedShopID int }` + navbar |
| `orders.html` | `struct{ Orders []models.Order }` + navbar |
| `order_confirmation.html` | `models.Order` + navbar |
| `admin.html` | `struct{ TotalUsers, TotalShops, TotalOrders int, Users []models.User, Shops []models.Shop, Orders []models.Order }` |
| `shop_orders.html` | `struct{ Shop models.Shop, Orders []models.Order, Role string }` + navbar |
| `shop_manage.html` | `struct{ Shop models.Shop, Role string }` + navbar |
| `navbar.html` | receives parent data with `.Role` field |
| `vendor_navbar.html` | static, unused (navbar.html covers both roles) |

### Key Decisions

1. **Typed data structs in `internal/templdata/`**: Named structs for each page instead of anonymous structs in handlers. Clean templ signatures.

2. **Navbar as `Navbar(role string)` component**: Accepts role directly — no need to pass full parent context.

3. **Layout component for shared boilerplate**: Common `<html>`, `<head>`, fonts, and CSS links extracted to `layout.templ` to avoid repetition.

4. **Progressive migration**: Convert all templates, update all handlers, verify build, then remove HTML files.

### File Structure

```
halisi/
├── internal/
│   ├── templdata/              # NEW
│   │   └── views.go
│   └── handlers/
│       └── handlers.go         # MODIFIED
├── web/
│   ├── templ/                  # NEW
│   │   ├── layout.templ
│   │   ├── navbar.templ
│   │   ├── index.templ
│   │   ├── login.templ
│   │   ├── register.templ
│   │   ├── dashboard.templ
│   │   ├── shops.templ
│   │   ├── shop.templ
│   │   ├── order.templ
│   │   ├── orders.templ
│   │   ├── order_confirmation.templ
│   │   ├── admin.templ
│   │   ├── shop_orders.templ
│   │   └── shop_manage.templ
│   └── templates/              # REMOVED after migration
```

### Architecture Diagram

``` mermaid
graph LR
  H[handlers.go] --> T[templ components web/templ/]
  H --> D[templdata/views.go]
  T --> L[layout.templ]
  T --> N[navbar.templ]
  D --> M[models Shop/Order/User]
  T --> M
```

### Risks
- **Inline CSS**: Several templates have `<style>` blocks that must be preserved exactly.
- **Dashboard is 901 lines**: The largest template; careful conversion needed.
- **Navbar role branching**: Current navbar uses `{{if eq .Role "owner"}}` on parent context; templ will pass role explicitly.

# Delivery Steps

### ✓ Step 1: Create templdata package and layout + navbar components
Set up the foundation: typed view data structs and shared layout/navbar components.
- Create `internal/templdata/views.go` with named structs for each page (ShopsPage, ShopPage, OrderPage, OrdersPage, OrderConfirmationPage, AdminPage, ShopOrdersPage, ShopManagePage).
- Create `web/templ/layout.templ` with common HTML boilerplate (doctype, head, fonts, CSS links, body wrapper).
- Create `web/templ/navbar.templ` with `Navbar(role string)` component covering both customer and owner nav branches.
- Run `templ generate` to verify the generated code compiles.

### * Step 2: Convert static pages: index, login, register, dashboard
Convert the four static templates (no data bindings) to templ components.
- Create `web/templ/index.templ`, `login.templ`, `register.templ`, and `dashboard.templ` by porting HTML from the corresponding `.html` files.
- Use `layout.templ` as a wrapper to avoid repeating `<html>`, `<head>`, and font/CSS links.
- Update `handlers.go`: replace `renderTemplate` calls in `Home`, `Login`, `Register` (GET), and `Dashboard` with templ component calls.

###   Step 3: Convert data-bound pages: shops, shop, order, orders
Convert the customer-facing data-bound templates.
- Create `web/templ/shops.templ` accepting `templdata.ShopsPage`, with range over shops.
- Create `web/templ/shop.templ` accepting `models.Shop`.
- Create `web/templ/order.templ` accepting `templdata.OrderPage` with shop dropdown.
- Create `web/templ/orders.templ` accepting `templdata.OrdersPage` with range over orders.
- Update `handlers.go`: replace `renderTemplate` calls in `Shops`, `Shop`, `RenderOrderPage`, and `Orders` with templ calls using typed structs.

###   Step 4: Convert remaining pages: order_confirmation, admin, shop_orders, shop_manage
Convert the last four data-bound templates.
- Create `web/templ/order_confirmation.templ` accepting `models.Order`.
- Create `web/templ/admin.templ` accepting `templdata.AdminPage` with tables for shops, users, and orders.
- Create `web/templ/shop_orders.templ` accepting `templdata.ShopOrdersPage`.
- Create `web/templ/shop_manage.templ` accepting `templdata.ShopManagePage`.
- Update `handlers.go`: replace `renderTemplate` calls in `OrderConfirmation`, `AdminDashboard`, `ShopOrders`, and `ShopManage` with templ calls.

###   Step 5: Finalize: remove HTML templates, renderTemplate helper, and verify build
Clean up and verify the migration is complete.
- Remove the `renderTemplate` function and `html/template` and `bytes` imports from `handlers.go`.
- Delete all `.html` files in `web/templates/` and remove the directory.
- Run `templ generate` and `go build ./...` to verify the full project compiles.
- Verify no remaining references to `template.ParseFiles` or `renderTemplate` exist.
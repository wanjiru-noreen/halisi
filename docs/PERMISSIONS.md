# Roles and resource permissions

HALISI's database already constrains roles to `customer`, `vendor`, `rider`,
and `admin` (administrator). `internal/models/user.go` defines these values and
the permission matrix; `internal/middleware/roles.go` denies unknown roles and
permissions. No schema migration is required for this change.

| Role | Permission | Dashboard | Resources |
| --- | --- | --- | --- |
| Customer | `customer.resources` | `/customer/dashboard` | Own orders, saved addresses and payments |
| Vendor | `vendor.resources` | `/vendor/dashboard` | Own shop, products and shop orders |
| Rider | `rider.assignments` | `/rider/dashboard` | Deliveries currently assigned to that user |
| Administrator (`admin`) | `administration` | `/admin/dashboard` | User/vendor management and vendor approval status |

All roles can manage their own existing authentication/session/account flows.
Rider delivery access additionally requires an approved rider profile. An
application alone never grants delivery access or changes roles; see
[Task 6.1](RIDER_REGISTRATION.md). Any authenticated role may register its own
pending application, but no registration endpoint can approve it.
Administrator access is explicit through management routes; it does not bypass
another role's ownership-scoped endpoints or impersonate a vendor/rider/customer.
Public registration and first-time Google registration always create customers.
Rider/administrator roles require a trusted database operator or a future audited
provisioning workflow. Verified customers can become vendors through the
[business onboarding workflow](VENDOR_ONBOARDING.md). No arbitrary role-change API exists.
Registration rejects a `role` field rather than silently accepting it. Shop and
product updates similarly reject owner IDs, role fields and approval status.

## Enforcement

Protected routes first authenticate the session, then check its role permission.
The role is read from PostgreSQL on each request, so a changed/revoked role takes
effect for existing sessions on subsequent requests. Cookies, headers, query
parameters and JSON payloads cannot select a role or resource owner.

Resource queries in `internal/database/queries/permissions.sql` scope reads and
writes to the authenticated principal in the SQL statement itself. Vendor
ownership joins through `vendors.user_id`; payment ownership joins through its
order's customer; riders use `deliveries.rider_id`. This prevents checking an ID
first and then performing an unscoped write after ownership has changed.

An unauthenticated API request returns **401**, even with `Accept: text/html`.
Authenticated users lacking the route permission or object ownership get **403**.
Unknown object IDs return the same generic 403 as objects owned by someone else,
so these endpoints do not disclose existence through different errors. Invalid
resource values/IDs return 422; unknown JSON fields return 400. Database failures
return safe 503 messages. Foreign-key/uniqueness conflicts return 409.

## API surface

These are the currently implemented operations, not the complete commerce API.
All mutations require JSON except DELETE. Standard same-origin CSRF protections
and session-cookie settings still apply. Resource responses use snake_case fields.
Collections are limited to 100 records except the existing user-address query.

| Routes | Methods | Scope |
| --- | --- | --- |
| `/api/customer/orders`, `/api/customer/orders/{id}` | GET | Session customer's orders |
| `/api/customer/addresses`, `/api/customer/addresses/{id}` | GET | Session user's saved addresses |
| `/api/customer/payments`, `/api/customer/payments/{id}` | GET | Payments attached to the session customer's orders |
| `/api/vendor/shop` | GET, POST, PATCH | Session vendor's single shop |
| `/api/vendor/products` | GET, POST | Products in the session vendor's shop |
| `/api/vendor/products/{id}` | GET, PATCH, DELETE | That vendor's product only |
| `/api/vendor/orders`, `/api/vendor/orders/{id}` | GET | Orders for the session vendor's shop |
| `/api/rider/deliveries`, `/api/rider/deliveries/{id}` | GET | Session rider's assignments; unassigned deliveries are excluded |
| `/api/rider/deliveries/{id}` | PATCH | Assigned rider's permitted status transitions |
| `/api/admin/users` | GET | Management projection excluding password/session data |
| `/api/admin/dashboard` | GET | Administrator-only aggregates and recent activity; see [dashboard definitions](ADMIN_DASHBOARD.md) |
| `/api/admin/vendors` | GET | Vendor management listing |
| `/api/admin/vendors/{id}/status` | PATCH | Administrator-only vendor approval/status changes |

Shop creation/update accepts `{ "name": "My shop", "address": "Nairobi" }`.
The server supplies `user_id`; new shops start `pending`. Updating an address
snapshot clears the old saved-address link. Creating a second shop returns 409.
Product creation/update accepts `{ "name": "LPG refill", "price": "1500.00" }`.
Prices are decimal strings with at most two fraction digits; negatives, NaN and
values outside NUMERIC(12,2) are rejected. The server supplies the shop ID. A
product referenced by order history cannot be deleted (409). Product creation
does not imply inventory availability; inventory workflows remain separate.

Delivery updates accept only `{ "status": "picked_up" }`, `delivered`, or `failed`.
Allowed transitions are `assigned → picked_up/failed` and
`picked_up → delivered/failed`; attempts to skip, reverse or repeat transitions
return 403. Completion timestamps are server-generated. Riders cannot claim,
reassign or change the order of a delivery. A reassigned delivery immediately
becomes unavailable to the previous rider. Assignment/distribution is a separate
dispatch workflow; no self-assignment endpoint is exposed.

Administrator vendor updates accept a `status` of `pending`, `approved`,
`rejected` or `suspended`. Vendor owners cannot approve their own shops. No
management endpoint serializes a raw user model containing password hashes.
Application database credentials are trusted server credentials; this is HTTP
authorization, not PostgreSQL row-level security for arbitrary SQL clients.

## Redirects and dashboards

Successful password login returns a closed, role-derived `redirect` path. The
frontend directs users to that role's dashboard. Google login and `/dashboard`
use the same server-side role routing. Explicit Google linking still returns to
the account page. Each role dashboard is guarded independently; visiting another
role's dashboard returns 403. Unknown roles have no dashboard. Browser requests
without a session redirect to `/login`, while API requests remain 401.

Dashboards are authenticated read-only overviews of permitted API resources.
They use DOM text insertion for user-provided fields and local embedded assets.
Use `/account` for email verification/Google linking and logout. Mutation APIs
are available for subsequent management UI work.

## Verification and extending the API

Run `sqlc generate`, `npm run build:css`, `go test ./...`, `go build ./...`, and
`go vet ./...`. Generated sqlc types include JSON tags for consistent response
fields. Set `TEST_DATABASE_URL` to run PostgreSQL integration tests for the role
matrix, dashboard routing, customer isolation, vendor CRUD ownership, delivery
transitions/reassignment, administrator actions, role-field injection, public
registration restrictions and privilege revocation on an existing session.

New handlers must use both `RequireAuth` and `RequirePermission`, and a query
that scopes access to the session user. Keep raw, unscoped database helpers out
of HTTP resource handlers; do not trust caller-supplied ownership or role values.

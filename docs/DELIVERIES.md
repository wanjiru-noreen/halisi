# Delivery creation and lifecycle (Task 8.1)

Apply migrations with `make migrate-up` before starting the server. Migration 7
adds delivery ownership, location snapshots, lifecycle guards and automatic
creation. Migration 8 snapshots order coordinates at checkout. No new environment
variables are required.

## Eligibility and creation

Checkout creates a **pending order**, its items and inventory reservations; it
does not create a delivery. The owning vendor confirms an order with:

```http
POST /api/vendor/orders/{order_id}/confirm
```

The vendor must have the vendor role and own an approved, active shop. The order
must be pending and reference a saved address belonging to its customer. A
successful request returns HTTP 200 with `{"order": ...}`. Unauthenticated
requests return 401; wrong roles, ownership, missing address, or an order that is
no longer pending return 403. Invalid IDs return 422. Repeated confirmation does
not modify the order or create another delivery.

A PostgreSQL trigger creates one pending, unassigned delivery when an eligible
order enters `confirmed`, including through future application services or SQL.
Creation and confirmation are atomic: if delivery validation fails, confirmation
rolls back. The unique `deliveries.order_id` constraint prevents duplicates under
concurrent requests. A confirmed order here means vendor acceptance; payment
collection/reconciliation and dispatch are separate workflows.

Deliveries reference the order, vendor, customer and customer-owned saved address.
Pickup text comes from the shop address; drop-off text comes from the order's
checkout snapshot, including its coordinates. Pickup coordinates are copied from
the vendor's saved address at confirmation. Coordinates can be absent. Editing a
customer's saved address after checkout cannot change either the order's original
destination text or the coordinates used by its delivery.

## Viewing delivery details

Authenticated, role-scoped GET endpoints return `{"delivery": ...}`:

| Endpoint | Allowed reader |
| --- | --- |
| `/api/customer/deliveries/{id}` | Order's customer |
| `/api/vendor/deliveries/{id}` | Owning shop's vendor |
| `/api/rider/deliveries/{id}` | Assigned rider with current approved profile and rider role |
| `/api/admin/deliveries/{id}` | Administrator |

Unrelated users receive 403, including for unknown IDs; unauthenticated API
requests receive 401. Responses use `Cache-Control: no-store`. Customer order
details also expose the delivery's ID and status; before confirmation, their
`delivery` field is null.

## Lifecycle

| Current status | Permitted next statuses |
| --- | --- |
| `pending` | `assigned`, `cancelled` |
| `assigned` | `picked_up`, `failed`, `cancelled` |
| `picked_up` | `delivered`, `failed` |
| `delivered`, `failed`, `cancelled` | None |

PostgreSQL rejects skipped/reversed transitions, assignment to an unapproved
rider, changing an active or completed delivery's rider, and changes to identity
or location snapshots. A completed delivery requires `delivered_at`; other
statuses cannot carry it. Only confirmed, preparing or dispatched orders can
advance to assignment, pickup or completion.

The existing rider `PATCH /api/rider/deliveries/{id}` endpoint accepts only
`picked_up`, `delivered` and `failed`, for that rider's assignment. It validates
the transition under a row lock. Failed delivery releases reserved inventory in
the same transaction. Repeated/invalid transitions return 403 and cannot release
stock twice.

## Delivery assignment and tracking

### Assignment endpoints

Vendors and administrators can assign deliveries to approved riders:

```http
POST /api/vendor/deliveries/{id}/assign
POST /api/admin/deliveries/{id}/assign
```

Request payload:
```json
{
  "rider_id": 123
}
```

- **Vendor assignment**: Only the vendor owning the order's shop can assign its delivery (`vendor.resources` permission). Attempting to assign another vendor's delivery returns HTTP 403 (`ErrDeliveryNotFound`).
- **Admin assignment**: Administrators (`administration` permission) can assign or reassign any delivery.
- **Rider validation**: The assigned user must exist, possess the `rider` role, and have an approved profile (`rider_profiles.status = 'approved'`). Assigning an unapproved or pending rider returns HTTP 422 (`ErrRiderNotApproved`).
- **Lifecycle guard**: Deliveries can only be assigned or reassigned while in `pending` or `assigned` status. Once a delivery enters `picked_up`, `delivered`, `failed`, or `cancelled`, reassignment is forbidden and returns HTTP 422 (`ErrDeliveryNotAssignable`).
- **Atomicity**: Status and rider assignment update together under database-level row constraints.

### Available riders discovery

Vendors and administrators can retrieve the list of approved riders (up to 100 records) for assignment selection:

```http
GET /api/vendor/riders
GET /api/admin/riders
```

Returns:
```json
{
  "riders": [
    {
      "user_id": 5,
      "full_name": "Jane Doe",
      "phone": "+254712345678",
      "city": "Nairobi"
    }
  ]
}
```

### Customer order tracking

Customers tracking their order via `GET /api/customer/orders/{id}` receive the order's delivery status alongside the assigned rider's name and phone number (joined from `rider_profiles`):

```json
{
  "order": {
    "id": 1,
    "status": "confirmed"
  },
  "delivery": {
    "id": 10,
    "status": "assigned",
    "delivered_at": null,
    "rider_name": "Jane Doe",
    "rider_phone": "+254712345678",
    "pickup_address": "Pickup street",
    "dropoff_address": "Dropoff street, Nairobi",
    "pickup_latitude": -1.286389,
    "pickup_longitude": 36.817223,
    "dropoff_latitude": -1.200000,
    "dropoff_longitude": 36.800000
  }
}
```

The customer tracking web page (`/customer/orders/{id}`) renders a 5-stage progress indicator:
1. `pending` (Order Placed)
2. `confirmed` (Confirmed by Shop)
3. `assigned` (Rider Assigned)
4. `picked_up` (Out for Delivery)
5. `delivered` (Delivered)

When a rider is assigned, a dedicated rider card appears displaying the rider's name and a clickable `tel:` call button for direct communication.

### Maps and location support

Halisi provides map and location support for customer addresses and active delivery tracking while protecting user location data:

1. **Customer address geolocation**:
   - Customers can save optional latitude and longitude coordinates with their addresses (`-90 <= lat <= 90`, `-180 <= lon <= 180`).
   - The address management UI includes a **Use my location** button utilizing browser geolocation (`navigator.geolocation.getCurrentPosition`) with graceful error handling for permission denial, unavailable hardware, or timeouts.
   - Saved address cards display verified GPS coordinate badges and external map links with `rel="noopener noreferrer"`.

2. **Delivery route directions and fallback**:
   - Order tracking views display pickup (vendor shop) and dropoff (destination) locations alongside coordinates when captured.
   - Actionable map links open route directions or dropoff pins directly in map applications (`https://www.google.com/maps/dir/...` or `https://www.google.com/maps/search/...`).
   - **Graceful fallback**: Full physical street address text is snapshotted into PostgreSQL and always displayed. If external map services are unreachable, devices are offline, or coordinates are absent, riders and customers can rely on physical addresses.

3. **Location data protection**:
   - Location coordinates are restricted strictly to authorized principals: the ordering customer, the owning vendor, the assigned approved rider, and administrators.
   - Unauthenticated access returns HTTP 401; unauthorized access returns HTTP 403.


## Upgrade, rollback and tests

Migration 8 adds nullable `orders.delivery_latitude` and `delivery_longitude`
snapshots, validates paired coordinates and their ranges, and changes delivery
creation to use those order values. Historical orders keep null coordinates:
their checkout coordinates cannot be recovered safely from mutable addresses.
Existing delivery snapshots are preserved. Downgrading version 8 removes the
order-coordinate columns and restores the version-7 creation function; run the
matching application version after a downgrade.

Migration 7 preserves existing delivery IDs and statuses, backfills ownership and
snapshots, and creates missing deliveries for eligible `confirmed` or `preparing`
orders. Legacy deliveries whose orders have no saved address keep a null
`address_id`; every new delivery requires a valid customer-owned address. Legacy
deliveries remain historical records even if originally created before order
confirmation. Ineligible historical orders are not dispatched automatically.

`make migrate-down` at version 7 removes these triggers, constraints and added
columns, preserving deliveries and their pre-existing columns. Snapshot columns
are lost on downgrade; back up the database first. Do not run the new server
against the downgraded schema. Migration versions prevent repeat application.

Regenerate parameterized SQL with `make sqlc-generate`. Run:

```sh
go test ./...
go build ./...
go vet ./...
```

Set `TEST_DATABASE_URL` for PostgreSQL HTTP/ownership/concurrency/lifecycle tests
and `TEST_MIGRATION_DATABASE_URL` for upgrade, repeat-up, rollback and reapply
tests. They create isolated schemas and remove them afterward. Unit tests cover
all lifecycle edges, including unknown states and terminal-state rejection.

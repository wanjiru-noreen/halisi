# Administrator dashboard (Task 9.1)

Visit `/admin/dashboard` with an administrator session. The responsive HTML page
loads `/api/admin/dashboard` and refreshes every minute while the tab is visible.
The existing session and administration permission middleware protect both routes.
Other roles receive 403; unauthenticated API requests receive 401. Responses use
`Cache-Control: no-store`. Changing a user's role takes effect on their next
request. No new configuration or migrations are required beyond the existing
schema (through migration 8).

## Dates and statistics

The API accepts `start=YYYY-MM-DD&end=YYYY-MM-DD`. Both dates are inclusive in
East Africa Time (UTC+3), implemented as a half-open PostgreSQL timestamp range.
Omitting both selects the last 30 calendar days including today. One missing
date, invalid/repeated dates, a reversed range, or more than 366 days returns 422.
The page also offers a last-seven-days shortcut.

All aggregate queries run inside one read-only, repeatable-read transaction so
they describe the same database snapshot. Counts are aggregate queries over all
matching rows, independent of the existing resource listings' 100-row limit.

| Statistic | Definition |
| --- | --- |
| Total customers | Current users with the customer role, across all dates |
| New customers | Current customer accounts created in the selected period |
| Vendor businesses | All vendor profiles; includes pending, rejected and suspended businesses |
| Active vendors | Currently approved and active businesses, across all dates |
| Rider profiles | All rider applications, across all dates |
| Approved riders | Approved profiles whose users currently hold the rider role |
| New vendors/riders | Profiles created in the selected period |
| Pending vendors/riders | Profiles currently pending, across all dates |
| Orders | Orders created in the selected period, including cancelled orders, grouped by their current status |
| Recent activity | Latest 20 customer registrations, vendor registrations, rider applications, orders and payment records created in the period |

Recent activity is a view of record creation and current status, not a historical
audit log of transitions. It excludes personal contact details, addresses,
credentials and tokens. Platform settings and audit logs remain a later task.

## Financial definitions and limitations

Financial totals describe **payments created in the selected period using their
current status**. They do not represent transactions completed or refunded on
those dates: the current schema has no payment-completion/refund timestamps or
separate refund ledger. Historical date ranges can change when payment statuses
change. The dashboard states this basis alongside the figures.

Each currency is displayed separately; there is no currency conversion or sum
across currencies. PostgreSQL calculates the decimal sums and the API returns
decimal strings. The browser formats these without floating-point arithmetic.

| Financial field | Definition |
| --- | --- |
| Gross sales | Sum of completed and fully refunded payment amounts in the creation-date cohort |
| Full refunds | Sum of payments currently marked `refunded` in that same cohort |
| Net collections | Gross sales minus full refunds; equivalently, completed payment amounts |
| Commissions | `null` / **Not tracked**, because no commission transactions or applied fee rates are recorded |
| Vendor settlements | `null` / **Not tracked**, because no settlement transactions are recorded |

Pending, processing, failed and cancelled payments are excluded from collections.
Full refunds are the only refunds represented by the current payment model;
partial refunds cannot be inferred. An empty financial list means there are no
completed/refunded payments in the period. It does not imply a zero commission
or a zero settlement balance. Net collections are neither platform profit nor
vendor payouts. Recording and managing commissions, partial refunds and vendor
settlements requires the later financial-management workflow; no rates or totals
are fabricated by this dashboard.

## Development and verification

```sh
make sqlc-generate
npm run build:css
node --check web/static/js/admin-dashboard.js
go test ./...
go build ./...
go vet ./...
```

Set `TEST_DATABASE_URL` for the live administrator dashboard tests and
`TEST_MIGRATION_DATABASE_URL` for the full migration regression suite. Tests use
isolated schemas and cover authorization, role revocation, local-date boundaries,
unbounded account totals, exact decimal sums, multiple currencies, excluded
payment states, empty data, recent-activity limits, live updates and safe failures.
Unit tests cover date validation, leap days and the Nairobi default date range.

The dashboard handles loading, empty data, authorization and service errors. Old
figures are hidden if a refreshed request fails. Desktop navigation becomes a
compact top navigation on mobile; wide financial tables scroll within their own
container. No dashboard data is embedded in static JavaScript or HTML.

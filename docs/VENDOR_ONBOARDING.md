# Vendor onboarding (Task 4.1)

Apply migration `000004`, then start the server. The workflow links to the role
dashboard shell; the full Milestone 4 inventory/order/analytics tasks are separate.

## Workflow

A verified, authenticated customer/vendor opens `/vendor/register`, providing
business name, contact email, international phone, street address, city and
operating hours for `mon`–`sun`. Open days use same-day `HH:MM` times in
Africa/Nairobi; closed days contain empty times. At least one day must be open.
The authenticated user's ID is supplied by the server. A customer becomes a vendor
(never an administrator) when the application is saved; one business per user.

At `/vendor/verification`, upload `business_registration` and `lpg_license`
documents and submit. These are HALISI's required evidence categories; an
administrator manually checks authenticity and applicability, rather than the
application automatically determining legal licensing compliance.

Pending applications are inactive and hidden. Administrators review submitted
applications at `/admin/vendor-verification`, recording a decision and reason.
Approval activates the vendor; rejection allows corrections/replacement uploads
and resubmission. Changes to approved details/documents reset approval. Suspended
vendors cannot edit, upload or resubmit until an administrator lifts suspension
to pending; a fresh submission and approval are still required.

## API

| Route | Method | Authorization / behavior |
| --- | --- | --- |
| `/api/vendor/application` | GET | Customer/vendor's own application and metadata; null if not yet registered |
| `/api/vendor/application` | PUT | Verified customer/vendor; complete business profile |
| `/api/vendor/documents/{type}` | POST | Vendor; exactly one multipart file named `document` |
| `/api/vendor/application/submit` | POST | Owner; JSON `{}`; validates profile and both stored documents |
| `/api/vendor/documents/{id}` | GET | Owner/admin; private attachment |
| `/api/admin/vendor-applications/{id}` | GET | Administrator; application and document metadata |
| `/api/admin/vendor-applications/{id}/review` | POST | Administrator; JSON `status` and `reason` (3–1000 bytes) |
| `/api/admin/vendors/{id}/status` | PATCH | Legacy route delegates to the same review policy |
| `/api/customer/vendors?q=...` | GET | Customer; approved AND active only; name/city search, maximum 50 results |

Profile fields are `name`, `contact_email`, `contact_phone`, `address`, `city`,
and `operating_hours`. A day looks like `{ "open": "08:00", "close": "18:00",
"closed": false }`. Owner, role, approval and activity fields are not accepted.

Review statuses: approve/reject submitted pending applications; suspend an
approved business; use pending to lift suspension. Each decision records the
administrator, reason and timestamp in `vendor_review_events`. An owner cannot
review their own business even if their user later becomes an administrator.
Application updates, uploads, submission and reviews serialize on the vendor row.

## Private persistent files

`VENDOR_DOCUMENT_DIR` defaults to `.data/vendor-documents` in development and is
ignored by Git. Production requires an absolute path on a mounted persistent
volume. The supplied `render.yaml` configures `/var/data/vendor-documents` on an
attached disk and a paid disk-compatible plan; this change does not provision
or deploy a Render service. Only files on that volume survive container
replacement. See [Render disks](https://render.com/docs/disks) and
[Blueprint disk configuration](https://render.com/docs/blueprint-spec#disks).

Use one application host for this disk backend. Multiple hosts require shared
private storage. Back up the volume together with PostgreSQL and define retention.
Server-side files have random 256-bit names, directory mode 0700 and file mode
0600 on Unix. `os.Root` confines file access. Client filenames never choose disk
paths. Storage is outside the public HTTP filesystem; API metadata excludes keys.

PDF, JPEG and PNG uploads are limited to 5 MiB. Extension, declared MIME and
detected type must agree. Images are decoded with a 10-megapixel limit; PDFs
require a header and end marker. Body size and simultaneous processing are
bounded. This validates formats, not document authenticity or embedded PDF code.
Administrators use their trusted document-review process for downloaded content.

Downloads require a current owner/admin session and use forced attachment,
`application/octet-stream`, `nosniff`, sandbox CSP and `Cache-Control: no-store`.
Files persist across process restarts. Replacements delete old files after
metadata commits; failed metadata writes delete new files. Abrupt failure can
leave private unreferenced files: clean those operationally only after comparing
against database references. Audit events record decisions, not historical file
versions. A rollback does not delete disk files; reconcile storage before downgrade.

## Approval gates and existing data

New vendors and legacy approvals start pending/inactive. Legacy metadata-only
documents cannot satisfy submission until replaced through the upload endpoint.
Approval sets `is_active=true`; rejection, suspension and owner edits clear it.
Customer search exposes public business details without documents or review notes.
PostgreSQL triggers reject new orders/order items for unavailable vendors;
inventory reservations also require approval and activity. Existing orders can
still be read/fulfilled after suspension. Draft product management does not
publish products or permit unverified sales. There is no self-activation endpoint.

## Verification

Run `sqlc generate`, `npm run build:css`, `go test ./...`, `go build ./...`, and
`go vet ./...`. `TEST_DATABASE_URL` enables isolated PostgreSQL tests for the
complete workflow, private access, type/size rejection, correction/resubmission,
self-approval prevention, legacy-route safety, visibility and blocked sales.
Unit tests cover contact/hours validation, file formats, traversal protection and
persistence after reopening storage. CI supplies disposable PostgreSQL.

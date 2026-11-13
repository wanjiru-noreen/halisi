# Task 6.1: Rider registration

Apply migrations through `000006` before starting the updated server. This task
adds the application/profile foundation only. KYC/documents (6.2), administrative
approval (6.3), vehicles (6.4) and account-status controls (6.5) remain separate.

## Authenticated self-service API

| Route | Method | Result |
| --- | --- | --- |
| `/api/rider/registration` | POST | Create the current user's pending application; 201 or 409 duplicate |
| `/api/rider/registration` | GET | Read the current user's profile; 200 or 404 if none |
| `/api/rider/registration` | PATCH | Partially update a pending profile; 200 or 403 when absent/locked |

All endpoints use existing HALISI session cookies and require authentication
(401 for API requests, including requests accepting HTML). They return a `rider`
profile plus `email`, which comes from the authenticated account. Any existing
authenticated user can apply; the operation never demotes or promotes that user.
No applicant ID or role is accepted from the client. There is no public profile
listing or endpoint for reading/editing another user's rider application.

Create a profile with `Content-Type: application/json`:

```json
{
  "full_name": "Alex Kamau",
  "phone": "+254712345678",
  "city": "Nairobi"
}
```

Name and city require 2–100 Unicode characters without control characters. Phone
requires international format with `+` and 8–15 digits. Inputs are trimmed.
PATCH accepts any nonempty subset of those fields. Null fields are treated as
omitted; required values cannot be cleared. Invalid values return 422. Unknown
fields (including `status`, `role`, `user_id`, `email` or timestamps) return 400.
Query parameters cannot select another account. Account email is not duplicated
or independently editable in the rider profile.

## Persistence and duplicates

`rider_profiles` contains an identity primary key, unique user foreign key,
name/phone/city, lifecycle status and created/updated timestamps. The user FK
cascades on account deletion, subject to any other historical-user references.
The unique constraint prevents duplicate applications even under concurrent
requests; registration inserts `pending` explicitly. A trigger maintains
`updated_at`. Status has `pending`, `approved`, `rejected`, `suspended` values
as a foundation for later tasks, with no status-change HTTP API in 6.1.

Profile edits are owner-scoped SQL updates allowed only while pending. Approved,
rejected or suspended profiles remain readable by their owner but are not editable
through this task's API. Reapplication/review transitions belong to later tasks.
Account profile data and roles are not changed by rider profile edits.

## No delivery authorization from registration

Creating an application never grants the `rider` role or approves a profile.
Delivery APIs and the rider dashboard require BOTH that role and an approved
`rider_profiles` row. Missing, pending, rejected or suspended profiles return 403;
database errors fail closed with a safe 503. Existing assignment ownership checks
still apply after this additional gate. Approval is checked on each request.

Existing `rider` accounts without an approved profile also lose delivery access;
the migration deliberately does not auto-approve legacy users. Granting approval,
granting the rider role, assigning deliveries and collecting KYC/vehicles must
use the future trusted administrative workflows. This task does not expose an
approval or self-assignment endpoint. Tests seed approved rows solely as trusted
fixtures for existing delivery regression tests.

## Verification

```sh
sqlc generate
go test ./...
go build ./...
go vet ./...
```

`TEST_DATABASE_URL` enables isolated PostgreSQL integration tests. Coverage
includes normalization/persistence, partial updates/timestamps, ownership,
concurrent duplicates, privilege/status-field injection, unchanged account roles,
and denied delivery reads/mutations for pending applicants even with a legacy
rider role and assigned delivery. Unit tests cover validation and approval-gate
failure cases. Existing CI already supplies the database.

Migration 6's down file drops rider profiles and their trigger/indexes. Downgrade
only with compatible application code and after backing up registration data.

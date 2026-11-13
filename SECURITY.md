
# HALISI Security Policy

## Supported Versions

Security fixes are prioritized for the latest supported release of HALISI.

| Version | Supported |
|---------|-----------|
| Latest stable release | Yes |
| Older releases | Case by case |
| Unreleased development code | Reviewed during development |

## Reporting a Vulnerability

Please report suspected security vulnerabilities privately.

Preferred reporting method:
GitHub Security Advisories, using the repository's
"Report a vulnerability" feature when enabled.

Do not publicly disclose vulnerabilities through GitHub Issues or pull requests before they have been assessed.

Include:
- A description of the vulnerability
- Affected components or endpoints
- Steps to reproduce
- Potential security impact
- Suggested remediation, if available

Do not include real customer credentials, identity documents, or other sensitive information in reports.

## Security Priorities

HALISI prioritizes protection of:

- Customer and vendor accounts
- Authentication credentials and sessions
- Vendor and rider KYC documents
- Payment transactions and callbacks
- Order and delivery records
- Customer addresses and location data
- Administrative operations
- Database credentials and API keys

## Security Requirements

### Authentication and Authorization

Protected operations must enforce authentication and appropriate authorization.

Role-based permissions and resource ownership must be validated server-side.

### Data Protection

Sensitive information must not appear in application logs or public API responses.

Secrets must be stored outside source control.

### Database Security

Database operations must use parameterized queries.

Transactions should protect operations that require atomicity.

### Payment Security

Payment callbacks must be authenticated or verified using the payment provider's supported mechanism.

Duplicate callbacks must not create duplicate payments or orders.

### Document Security

KYC documents must be accessible only to authorized users.

Uploaded documents must be validated and stored securely.

### Dependencies

Dependencies should be reviewed for known vulnerabilities and updated when appropriate.

## Responsible Disclosure

Please allow the maintainers reasonable time to investigate and address confirmed vulnerabilities before public disclosure.

Security fixes should be tested before release.

## Scope

This policy applies to HALISI's application code, APIs, authentication, database integrations, and supported deployment infrastructure.

Third-party services remain subject to their respective security policies.

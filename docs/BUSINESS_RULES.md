# Halisi Business Rules & Domain Logic

This document specifies the core operational business rules, state machine transitions, compliance mandates, and access policies governing HALISI.

---

## 1. User Roles & Permissions

Halisi enforces four primary user roles with strict separation of duties:

| Role | Core Capabilities | Restrictions |
|---|---|---|
| **Customer** | Browse catalog, place orders, make M-Pesa payments, track deliveries. | Cannot view unverified vendor inventory, modify order statuses, or access other users' orders. |
| **Vendor** | Manage LPG products/inventory, view orders assigned to vendor, upload KYC documents. | Inventory remains invisible in public catalog until vendor KYC is approved by an Admin. Cannot access other vendors' inventory or payouts. |
| **Rider** | View available delivery runs, accept deliveries, update transit milestones. | Cannot accept delivery runs until KYC is verified. Restricted to active delivery quota to prevent capacity overload. |
| **Admin** | Review KYC documents, approve/reject vendors and riders, view platform audits and settlements. | Read/write access across tenant records for compliance and dispute resolution. |

---

## 2. Vendor KYC & Compliance Rules

1. **EPRA Compliance**: Liquid Petroleum Gas (LPG) vendors in Kenya must comply with the Energy and Petroleum Regulatory Authority (EPRA) regulations.
2. **Mandatory Documentation**: Vendors must submit a valid Business Permit and EPRA License.
3. **Approval Gate**:
   - New vendor accounts start in `pending_verification` status.
   - Products added by unverified vendors are marked `is_active: false` and filtered out of customer search and storefront queries.
   - Upon administrative approval, products transition to public visibility.
4. **Document Retention & Storage**:
   - Uploaded KYC documents are permanently stored on isolated persistent disk storage (`/var/data/vendor-documents`).
   - Downloads require authenticated session verifying vendor ownership or admin privilege.

---

## 3. Inventory & Pricing Rules

1. **Product Categorization**:
   - `cylinder_with_gas`: Full purchase of gas cylinder container + LPG content.
   - `refill`: Exchange of an empty standard cylinder for a refilled cylinder.
   - `accessory`: Regulators, safety hoses, clamps, burners.
2. **Whole Shilling Currency**: All retail gas prices and checkout amounts are stored in whole Kenya Shilling cents (`price_cents`) to eliminate floating-point rounding errors.
3. **Atomic Stock Decrement**:
   - Stock quantities are validated and decremented inside an atomic database transaction during order confirmation.
   - Orders cannot be placed if requested quantity exceeds available `stock_quantity`.

---

## 4. Order Lifecycle & State Machine

Orders follow an explicit, irreversible state progression:

```
[ pending ] 
     |
     v (Payment verified via M-Pesa Daraja)
[  paid   ] 
     |
     v (Rider accepts delivery run)
[ assigned ] 
     |
     v (Rider collects cylinder from vendor depot)
[in_transit] 
     |
     +--------------------------+
     |                          |
     v (Customer handover PIN)  v (Failed delivery / canceled)
[ delivered ]              [ cancelled ]
```

### State Transition Invariants:
- An order in `pending` can only transition to `paid` via an authorized payment confirmation or `cancelled` via timeout/customer cancellation.
- An order in `paid` can only transition to `assigned` when an approved rider claims the delivery run.
- An order in `assigned` can only transition to `in_transit` once the rider confirms physical pickup from the vendor.
- An order in `in_transit` can only transition to `delivered` upon customer handover confirmation.

---

## 5. Payments & M-Pesa Daraja Integration

1. **Server-Authoritative Pricing**:
   - Payment requests do not accept an amount from the client. The payment service retrieves the authoritative total directly from the PostgreSQL order record.
2. **Callback Security**:
   - Each payment attempt generates a unique, single-use callback token included in Safaricom Daraja's callback webhook URL.
   - Callbacks without a matching active token are rejected.
3. **Idempotent Settlement**:
   - Webhooks verifying successful payment check if the order has already been marked `paid`.
   - Duplicate callbacks are recorded idempotently without triggering duplicate stock changes or duplicate commissions.

---

## 6. Delivery Assignment & Rider Verification

1. **Eligibility**: Only riders in `active` verified status can view available orders and accept delivery runs.
2. **Concurrency Limit**: To prevent order hoarding and ensure prompt delivery times, a rider can have at most **2 active deliveries** (`assigned` or `in_transit`) simultaneously.
3. **Geo-Location**: Delivery runs calculate approximate distance from vendor depot to customer coordinates to determine rider delivery fees.

---

## 7. Platform Commissions & Settlements

1. **Platform Fee**: Halisi collects a platform commission on each completed LPG transaction according to agreed vendor agreements (flat or percentage).
2. **Ledger Entries**:
   - Order payment triggers a credit to the platform escrow account.
   - Delivery completion triggers a debit to escrow and credits the vendor's net earnings balance and rider's delivery fee balance.

BEGIN;

ALTER TABLE vendors ADD COLUMN commission_rate_bps INTEGER NOT NULL DEFAULT 1000
    CHECK (commission_rate_bps BETWEEN 0 AND 10000);
ALTER TABLE payments ADD COLUMN commission_rate_bps INTEGER
    CHECK (commission_rate_bps BETWEEN 0 AND 10000);
CREATE TABLE vendor_commission_rate_changes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vendor_id BIGINT NOT NULL REFERENCES vendors(id),
    previous_rate_bps INTEGER NOT NULL CHECK (previous_rate_bps BETWEEN 0 AND 10000),
    new_rate_bps INTEGER NOT NULL CHECK (new_rate_bps BETWEEN 0 AND 10000),
    changed_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX vendor_commission_rate_changes_vendor_idx ON vendor_commission_rate_changes(vendor_id, created_at DESC);

CREATE TABLE order_commissions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id),
    payment_id BIGINT NOT NULL UNIQUE REFERENCES payments(id),
    vendor_id BIGINT NOT NULL REFERENCES vendors(id),
    gross_amount NUMERIC(12,2) NOT NULL CHECK (gross_amount > 0 AND gross_amount <> 'NaN'::numeric),
    commission_rate_bps INTEGER NOT NULL CHECK (commission_rate_bps BETWEEN 0 AND 10000),
    commission_amount NUMERIC(12,2) NOT NULL CHECK (commission_amount >= 0 AND commission_amount <> 'NaN'::numeric),
    vendor_amount NUMERIC(12,2) NOT NULL CHECK (vendor_amount >= 0 AND vendor_amount <> 'NaN'::numeric),
    currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    settlement_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (gross_amount = commission_amount + vendor_amount)
);
CREATE INDEX order_commissions_vendor_available_idx ON order_commissions(vendor_id, created_at) WHERE settlement_id IS NULL;

CREATE TABLE settlements (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vendor_id BIGINT NOT NULL REFERENCES vendors(id),
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0 AND amount <> 'NaN'::numeric),
    currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed')),
    payout_reference TEXT UNIQUE,
    failure_reason TEXT,
    created_by BIGINT NOT NULL REFERENCES users(id),
    updated_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CHECK ((status = 'paid') = (completed_at IS NOT NULL)),
    CHECK (status <> 'paid' OR payout_reference IS NOT NULL)
);
ALTER TABLE order_commissions ADD CONSTRAINT order_commissions_settlement_fk FOREIGN KEY (settlement_id) REFERENCES settlements(id);
CREATE INDEX settlements_vendor_created_idx ON settlements(vendor_id, created_at DESC);

COMMIT;

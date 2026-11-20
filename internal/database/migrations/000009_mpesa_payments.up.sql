BEGIN;

ALTER TABLE payments
    ADD COLUMN provider TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN request_key TEXT,
    ADD COLUMN phone_number TEXT,
    ADD COLUMN merchant_request_id TEXT,
    ADD COLUMN checkout_request_id TEXT,
    ADD COLUMN callback_token_hash TEXT,
    ADD COLUMN result_code INTEGER,
    ADD COLUMN result_description TEXT;

ALTER TABLE payments ADD CONSTRAINT payments_provider_check CHECK (provider IN ('manual', 'mpesa'));
ALTER TABLE payments ADD CONSTRAINT payments_mpesa_request_values_check CHECK (
    provider <> 'mpesa' OR (
        request_key IS NOT NULL AND length(request_key) BETWEEN 1 AND 128
        AND phone_number IS NOT NULL AND phone_number ~ '^254[17][0-9]{8}$'
        AND callback_token_hash IS NOT NULL AND callback_token_hash ~ '^[0-9a-f]{64}$'
    )
);
CREATE UNIQUE INDEX payments_request_key_idx ON payments(request_key) WHERE request_key IS NOT NULL;
CREATE UNIQUE INDEX payments_checkout_request_id_idx ON payments(checkout_request_id) WHERE checkout_request_id IS NOT NULL;
CREATE UNIQUE INDEX payments_callback_token_hash_idx ON payments(callback_token_hash) WHERE callback_token_hash IS NOT NULL;
CREATE UNIQUE INDEX payments_active_order_idx ON payments(order_id) WHERE provider='mpesa' AND status IN ('pending', 'processing');

COMMIT;

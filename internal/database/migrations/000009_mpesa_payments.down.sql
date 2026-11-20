BEGIN;

DROP INDEX payments_active_order_idx;
DROP INDEX payments_callback_token_hash_idx;
DROP INDEX payments_checkout_request_id_idx;
DROP INDEX payments_request_key_idx;
ALTER TABLE payments
    DROP CONSTRAINT payments_mpesa_request_values_check,
    DROP CONSTRAINT payments_provider_check,
    DROP COLUMN result_description,
    DROP COLUMN result_code,
    DROP COLUMN callback_token_hash,
    DROP COLUMN checkout_request_id,
    DROP COLUMN merchant_request_id,
    DROP COLUMN phone_number,
    DROP COLUMN request_key,
    DROP COLUMN provider;

COMMIT;

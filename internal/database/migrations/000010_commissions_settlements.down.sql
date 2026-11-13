BEGIN;

DROP TABLE order_commissions;
DROP TABLE settlements;
DROP TABLE vendor_commission_rate_changes;
ALTER TABLE payments DROP COLUMN commission_rate_bps;
ALTER TABLE vendors DROP COLUMN commission_rate_bps;

COMMIT;

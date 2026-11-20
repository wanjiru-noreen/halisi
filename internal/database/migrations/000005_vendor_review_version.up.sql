BEGIN;
ALTER TABLE vendors ADD COLUMN review_version BIGINT NOT NULL DEFAULT 1 CHECK (review_version>0);
ALTER TABLE vendor_review_events ADD COLUMN review_version BIGINT;
CREATE FUNCTION increment_vendor_review_version() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.review_version = OLD.review_version + 1;
    RETURN NEW;
END;
$$;
-- Includes profile edits, uploads, resubmissions, decisions and legacy writes.
CREATE TRIGGER vendors_review_version BEFORE UPDATE ON vendors FOR EACH ROW EXECUTE FUNCTION increment_vendor_review_version();
COMMIT;

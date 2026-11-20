BEGIN;
CREATE TABLE rider_profiles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL CHECK (char_length(btrim(full_name)) BETWEEN 2 AND 100),
    phone TEXT NOT NULL CHECK (phone ~ '^\+[1-9][0-9]{7,14}$'),
    city TEXT NOT NULL CHECK (char_length(btrim(city)) BETWEEN 2 AND 100),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','suspended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX rider_profiles_status_idx ON rider_profiles(status,created_at);
CREATE TRIGGER rider_profiles_updated_at BEFORE UPDATE ON rider_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();
COMMIT;

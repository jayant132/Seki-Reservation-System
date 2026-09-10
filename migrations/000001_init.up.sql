-- btree_gist is required for the EXCLUDE constraint below: it lets us mix
-- an equality check (resource_id) with a range-overlap check (during) in a
-- single index-backed constraint.
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'customer' CHECK (role IN ('admin', 'customer')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE resources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    capacity    INT NOT NULL DEFAULT 1 CHECK (capacity > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE bookings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id  UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time   TIMESTAMPTZ NOT NULL,
    end_time     TIMESTAMPTZ NOT NULL,
    during       TSTZRANGE GENERATED ALWAYS AS (tstzrange(start_time, end_time, '[)')) STORED,
    status       TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'cancelled')),
    notes        TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT valid_time_range CHECK (end_time > start_time)
);

-- This is the load-bearing line of the whole project: the database itself
-- refuses to store two CONFIRMED bookings for the same resource whose time
-- ranges overlap, regardless of how many application instances or
-- goroutines race to insert at the same instant. Cancelled bookings are
-- excluded from the constraint so cancelling and re-booking the same slot
-- works.
ALTER TABLE bookings
    ADD CONSTRAINT no_overlapping_confirmed_bookings
    EXCLUDE USING gist (
        resource_id WITH =,
        during WITH &&
    )
    WHERE (status = 'confirmed');

CREATE INDEX idx_bookings_resource_id ON bookings(resource_id);
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_during ON bookings USING gist(during);

-- Idempotency keys let a client safely retry a POST /bookings request
-- (e.g. after a network timeout where it never saw the response) without
-- risking a duplicate booking. The response is captured once and replayed
-- verbatim on subsequent requests with the same key.
CREATE TABLE idempotency_keys (
    key            TEXT PRIMARY KEY,
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_hash   TEXT NOT NULL,
    response_code  INT NOT NULL,
    response_body  JSONB NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Immutable, hash-chained audit log (same pattern proven in the Ringi
-- project) so every state-changing action on a booking is independently
-- verifiable after the fact.
CREATE TABLE audit_log (
    id            BIGSERIAL PRIMARY KEY,
    entity_type   TEXT NOT NULL,
    entity_id     UUID NOT NULL,
    actor_id      UUID REFERENCES users(id),
    action        TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}',
    prev_hash     TEXT NOT NULL,
    entry_hash    TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);

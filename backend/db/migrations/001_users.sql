-- Accounts. Runs automatically the first time the Postgres container
-- initialises its data volume (see docker-compose.db.yml).
--
-- There are no passwords: every sign-in is a one-time code emailed to an MIT
-- address, so owning the inbox is the proof of MIT membership.
--
-- whatsapp_phone is E.164 digits without the plus (e.g. 16175550123). It is
-- what the wa.me link on each listing points at, and how the WhatsApp bot
-- recognises which account an incoming message belongs to, so it is unique.

CREATE TABLE IF NOT EXISTS users (
    id             BIGSERIAL PRIMARY KEY,
    email          TEXT NOT NULL,
    name           TEXT NOT NULL,
    whatsapp_phone TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_lower CHECK (email = lower(email)),
    CONSTRAINT users_phone_digits CHECK (whatsapp_phone ~ '^[1-9][0-9]{7,14}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users (email);
CREATE UNIQUE INDEX IF NOT EXISTS users_whatsapp_phone_key ON users (whatsapp_phone);

-- One-time sign-in codes. Only a hash of the code is stored. A new request
-- for the same email replaces the previous code, so there is at most one
-- live code per address.
CREATE TABLE IF NOT EXISTS email_codes (
    email      TEXT PRIMARY KEY,
    code_hash  TEXT NOT NULL,
    attempts   INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

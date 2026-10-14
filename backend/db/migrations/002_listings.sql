-- Ticket listings. This sprint covers event tickets only; other categories
-- (furniture, subleases, textbooks...) will get their own columns or tables
-- when they are reopened, so nothing here is generic on purpose.
--
-- price_cents is per ticket. source records whether the listing came from
-- the web app or the WhatsApp bot. Sold or withdrawn listings keep their row
-- with status changed, so a seller's history survives.

CREATE TABLE IF NOT EXISTS listings (
    id          BIGSERIAL PRIMARY KEY,
    seller_id   BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_name  TEXT NOT NULL,
    event_date  DATE NOT NULL,
    venue       TEXT NOT NULL DEFAULT '',
    quantity    INT NOT NULL CHECK (quantity BETWEEN 1 AND 20),
    price_cents INT NOT NULL CHECK (price_cents >= 0),
    notes       TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT 'web' CHECK (source IN ('web', 'whatsapp')),
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'sold', 'withdrawn')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS listings_active_date ON listings (event_date) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS listings_seller ON listings (seller_id);

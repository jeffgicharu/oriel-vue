-- Where a visitor came from each time they got in touch: the first time they
-- arrived at the site, and the arrival that led to what they sent. A WhatsApp
-- tap leaves no contact details, so it has no lead.
CREATE TABLE touches (
    id             INTEGER PRIMARY KEY,
    lead_id        INTEGER REFERENCES leads (id) ON DELETE CASCADE,
    at             INTEGER NOT NULL,
    action         TEXT NOT NULL CHECK (action IN ('enquiry', 'viewing', 'reservation', 'whatsapp')),
    unit_id        TEXT NOT NULL DEFAULT '',
    channel        TEXT NOT NULL,
    source         TEXT NOT NULL DEFAULT '',
    medium         TEXT NOT NULL DEFAULT '',
    campaign       TEXT NOT NULL DEFAULT '',
    referrer       TEXT NOT NULL DEFAULT '',
    landing        TEXT NOT NULL DEFAULT '',
    arrived_at     INTEGER NOT NULL,
    first_channel  TEXT NOT NULL,
    first_source   TEXT NOT NULL DEFAULT '',
    first_medium   TEXT NOT NULL DEFAULT '',
    first_campaign TEXT NOT NULL DEFAULT '',
    first_referrer TEXT NOT NULL DEFAULT '',
    first_landing  TEXT NOT NULL DEFAULT '',
    first_at       INTEGER NOT NULL
) STRICT;
CREATE INDEX touches_lead ON touches (lead_id, at);
CREATE INDEX touches_action ON touches (action, at);

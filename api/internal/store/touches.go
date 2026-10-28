package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
)

// RecordTouch keeps where a visitor came from behind something they sent:
// an enquiry, a viewing, a reservation, or a tap on WhatsApp. A lead of 0
// records a touch without one. Nothing is kept when the visitor's browser
// sent no attribution, as with the sales desk's own entries.
func RecordTouch(ctx context.Context, tx *sql.Tx, leadID int64, action, unitID string, a *attribution.Attribution, at time.Time) error {
	if a == nil {
		return nil
	}
	var lead any
	if leadID > 0 {
		lead = leadID
	}
	f, l := a.First, a.Last
	_, err := tx.ExecContext(ctx, `INSERT INTO touches (lead_id, at, action, unit_id,
		channel, source, medium, campaign, referrer, landing, arrived_at,
		first_channel, first_source, first_medium, first_campaign, first_referrer, first_landing, first_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lead, at.Unix(), action, unitID,
		l.Channel, l.Source, l.Medium, l.Campaign, l.Referrer, l.Landing, l.At.Unix(),
		f.Channel, f.Source, f.Medium, f.Campaign, f.Referrer, f.Landing, f.At.Unix())
	return err
}

// RecordWhatsApp keeps a visitor's tap on a WhatsApp link, with the home it
// was about, if any, and where they came from. It goes into the activity
// log, so the sales team sees a conversation may be on its way.
func (db *DB) RecordWhatsApp(ctx context.Context, unitID string, a attribution.Attribution, at time.Time) error {
	return db.Tx(ctx, func(tx *sql.Tx) error {
		if err := RecordTouch(ctx, tx, 0, "whatsapp", unitID, &a, at); err != nil {
			return err
		}
		subject, link := "Oriel", ""
		if unitID != "" {
			subject, link = "home "+unitID, "/desk/homes/"+unitID
		}
		return Activity(ctx, tx, at, 0, "visitor", "opened WhatsApp about", subject, "from "+a.Last.Channel, link)
	})
}

package store

import (
	"context"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
)

// ChannelCount is how far the leads one of the website's channels brought in
// have got.
type ChannelCount struct {
	Channel  string
	Leads    int // leads whose first arrival came through the channel
	Viewed   int // of those, the ones who booked a viewing they did not cancel
	Reserved int // of those, the ones who asked to reserve a home
}

// ChannelCounts follows the leads the website brought in, by the channel of
// their first arrival, from the first message to a viewing and a
// reservation. Every channel is listed, in the order the desk shows them.
//
// A lead counts when the website made it: its earliest touch is the one that
// created it. Leads the team added by hand, and people the team already knew
// before they used the website, are left out, so a referral who later
// reserves online does not count as a direct visit.
func (db *DB) ChannelCounts(ctx context.Context) ([]ChannelCount, error) {
	rows, err := db.QueryContext(ctx, `SELECT t.first_channel, count(*),
			sum(EXISTS (SELECT 1 FROM viewings v WHERE v.lead_id = l.id AND v.status <> 'cancelled')),
			sum(EXISTS (SELECT 1 FROM reservations r WHERE r.lead_id = l.id))
		FROM leads l
		JOIN touches t ON t.id = (SELECT f.id FROM touches f WHERE f.lead_id = l.id ORDER BY f.at, f.id LIMIT 1)
		WHERE t.at <= l.created_at
		GROUP BY t.first_channel`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byChannel := map[string]ChannelCount{}
	for rows.Next() {
		var c ChannelCount
		if err := rows.Scan(&c.Channel, &c.Leads, &c.Viewed, &c.Reserved); err != nil {
			return nil, err
		}
		byChannel[c.Channel] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]ChannelCount, 0, len(attribution.Channels))
	for _, ch := range attribution.Channels {
		c := byChannel[ch]
		c.Channel = ch
		out = append(out, c)
	}
	return out, nil
}

// WhatsAppTaps counts the taps on WhatsApp links since a time, by the channel
// of the arrival that led to each. Every channel has an entry, if only zero.
func (db *DB) WhatsAppTaps(ctx context.Context, since time.Time) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, `SELECT channel, count(*) FROM touches WHERE action = 'whatsapp' AND at >= ? GROUP BY channel`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for _, ch := range attribution.Channels {
		out[ch] = 0
	}
	for rows.Next() {
		var ch string
		var n int
		if err := rows.Scan(&ch, &n); err != nil {
			return nil, err
		}
		out[ch] = n
	}
	return out, rows.Err()
}

// FirstTouches is, for each lead the website has seen, the first arrival it
// knows of: the first touch of the lead's earliest touch.
func (db *DB) FirstTouches(ctx context.Context) (map[int64]attribution.Touch, error) {
	rows, err := db.QueryContext(ctx, `SELECT lead_id, first_channel, first_source, first_medium, first_campaign, first_referrer, first_landing, first_at
		FROM touches WHERE lead_id IS NOT NULL ORDER BY at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]attribution.Touch{}
	for rows.Next() {
		var id, at int64
		var t attribution.Touch
		if err := rows.Scan(&id, &t.Channel, &t.Source, &t.Medium, &t.Campaign, &t.Referrer, &t.Landing, &at); err != nil {
			return nil, err
		}
		if _, seen := out[id]; !seen {
			t.At = time.Unix(at, 0)
			out[id] = t
		}
	}
	return out, rows.Err()
}

// LeadSources lists every source the leads came from, the commonest first,
// for the desk's filter.
func (db *DB) LeadSources(ctx context.Context) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT source FROM leads GROUP BY source ORDER BY count(*) DESC, source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

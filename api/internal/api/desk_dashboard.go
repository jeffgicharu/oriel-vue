package api

import (
	"math"
	"net/http"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// DeskDashboard is the desk's front page.
type DeskDashboard struct {
	AsOf string `json:"asOf"`
	// Counts is how many homes have each status.
	Counts map[string]int `json:"counts"`
	Total  int            `json:"total"`
	// Stock is the price of every home still to sell: available, on hold or
	// withheld.
	Stock int64 `json:"stock"`
	// SoldValue is the price of every confirmed or completed reservation.
	SoldValue int64 `json:"soldValue"`
	// Months is the homes sold in each of the last eight months, this one
	// last.
	Months []DeskMonth `json:"months"`
	// NewLeads came in the last seven days, NewLeadsWas in the seven before.
	NewLeads    int             `json:"newLeads"`
	NewLeadsWas int             `json:"newLeadsWas"`
	Funnel      []DeskFunnelRow `json:"funnel"`
	// Sources is every lead by where it came from, the commonest first.
	Sources []DeskSourceCount `json:"sources"`
	// Week is the viewings from today to a week ahead, earliest first.
	Week []DeskViewing `json:"week"`
	// WaitingTotal is every reservation request waiting for a decision;
	// Waiting is the six newest.
	WaitingTotal int               `json:"waitingTotal"`
	Waiting      []DeskReservation `json:"waiting"`
	// FollowUpsTotal is every open lead due a follow-up today or earlier;
	// FollowUps is up to six of them, the longest overdue first. An agent
	// sees only their own.
	FollowUpsTotal int        `json:"followUpsTotal"`
	FollowUps      []DeskLead `json:"followUps"`
	// Channels follows the leads the website brought in, by the channel of
	// their first arrival, to a viewing and a reservation. Every channel is
	// listed, in the desk's order.
	Channels []DeskChannel `json:"channels"`
	// WhatsApp is the taps on WhatsApp links in the last 30 days, by the
	// channel each visitor arrived through, every channel in the same order.
	WhatsApp []DeskTaps `json:"whatsapp"`
}

// DeskMonth is the homes sold in a month.
type DeskMonth struct {
	Month string `json:"month"`
	Label string `json:"label"`
	Sold  int    `json:"sold"`
	Value int64  `json:"value"`
}

// DeskFunnelRow is how many leads got as far as a stage. A lead counts in
// every stage it has passed; a lost lead is known only to have started.
type DeskFunnelRow struct {
	Stage   string `json:"stage"`
	Label   string `json:"label"`
	N       int    `json:"n"`
	Reached int    `json:"reached"`
	// Percent is the share of every lead that got this far.
	Percent int  `json:"percent"`
	Lost    bool `json:"lost"`
}

// DeskSourceCount is how many leads came from one source.
type DeskSourceCount struct {
	Source string `json:"source"`
	Leads  int    `json:"leads"`
}

// DeskChannel is how far the leads one of the website's channels brought in
// have got.
type DeskChannel struct {
	Channel string `json:"channel"`
	Leads   int    `json:"leads"`
	// Viewed booked a viewing they did not cancel.
	Viewed int `json:"viewed"`
	// Reserved asked to reserve a home.
	Reserved int `json:"reserved"`
}

// DeskTaps is the taps on WhatsApp links from one channel.
type DeskTaps struct {
	Channel string `json:"channel"`
	Taps    int    `json:"taps"`
}

func (a *App) deskDashboard(w http.ResponseWriter, r *http.Request) {
	staff := staffFrom(r.Context())
	db := a.db(r)
	now := a.now()
	agent := agentScope(staff)
	dash, err := db.DashboardOn(r.Context(), now, agent)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	out := DeskDashboard{
		AsOf: dateOf(now), Counts: c.Counts(), Total: len(c.Homes), SoldValue: dash.SoldValue,
		Months: []DeskMonth{}, NewLeads: dash.NewLeads, NewLeadsWas: dash.NewLeadsWas,
		Funnel: funnelOf(dash.Funnel), Sources: []DeskSourceCount{}, Week: []DeskViewing{},
		WaitingTotal: dash.Waiting, Waiting: []DeskReservation{}, FollowUpsTotal: dash.FollowUps, FollowUps: []DeskLead{},
		Channels: []DeskChannel{}, WhatsApp: []DeskTaps{},
	}
	for _, h := range c.Homes {
		if h.Status == store.Available || h.Status == store.Held || h.Status == store.Withheld {
			out.Stock += h.Price
		}
	}
	for _, m := range dash.Months {
		out.Months = append(out.Months, DeskMonth{Month: m.Key, Label: m.Label, Sold: m.N, Value: m.Value})
	}
	for _, s := range dash.Sources {
		out.Sources = append(out.Sources, DeskSourceCount{Source: s.Label, Leads: s.N})
	}
	for _, v := range dash.Week {
		out.Week = append(out.Week, a.deskViewingOf(v))
	}

	waiting, err := db.Reservations(r.Context(), "requested", 6)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	for _, res := range waiting {
		out.Waiting = append(out.Waiting, a.deskReservationOf(res))
	}
	leads, err := db.Leads(r.Context(), store.LeadFilter{AgentID: agent, Sort: "follow", Limit: 40})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	for _, l := range leads {
		if l.Overdue(now) && len(out.FollowUps) < 6 {
			out.FollowUps = append(out.FollowUps, a.deskLeadOf(l))
		}
	}

	channels, err := db.ChannelCounts(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	for _, ch := range channels {
		out.Channels = append(out.Channels, DeskChannel{Channel: ch.Channel, Leads: ch.Leads, Viewed: ch.Viewed, Reserved: ch.Reserved})
	}
	taps, err := db.WhatsAppTaps(r.Context(), now.AddDate(0, 0, -30))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	for _, ch := range attribution.Channels {
		out.WhatsApp = append(out.WhatsApp, DeskTaps{Channel: ch, Taps: taps[ch]})
	}
	writeJSON(w, http.StatusOK, out)
}

// funnelOf works out how far leads have got from how many are at each stage.
// A lead at a later stage has passed through every earlier one; a lost lead
// is known only to have started.
func funnelOf(stages []store.Count) []DeskFunnelRow {
	total := 0
	byStage := map[string]int{}
	for _, s := range stages {
		total += s.N
		byStage[s.Key] = s.N
	}
	percent := func(n int) int {
		if total == 0 {
			return 0
		}
		return int(math.Round(100 * float64(n) / float64(total)))
	}
	open := []string{"new", "contacted", "viewing", "reservation", "won"}
	out := []DeskFunnelRow{}
	for i, stage := range open {
		reached := 0
		for _, later := range open[i:] {
			reached += byStage[later]
		}
		// Every lead starts as new, the lost ones included.
		if i == 0 {
			reached = total
		}
		out = append(out, DeskFunnelRow{Stage: stage, Label: store.StageTitle(stage), N: byStage[stage], Reached: reached, Percent: percent(reached)})
	}
	if n, ok := byStage["lost"]; ok {
		out = append(out, DeskFunnelRow{Stage: "lost", Label: "Lost", N: n, Reached: n, Percent: percent(n), Lost: true})
	}
	return out
}

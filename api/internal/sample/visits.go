package sample

import (
	"strings"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
	"github.com/jeffgicharu/oriel-vue/api/internal/building"
)

// Visitors on the website: where each lead the website brought in first
// arrived from, and the arrival behind every enquiry, viewing and reservation
// sent from the site. The site works the channel out from the link a visitor
// followed, so the story does too, through attribution.Clean.
//
// Visits draw on random numbers of their own, so the rest of the story turns
// out as it did before they were told.

// Touch is where a visitor came from behind something they sent from the
// website: an enquiry, a viewing booked online, a reservation request or a
// tap on WhatsApp.
type Touch struct {
	LeadID int64 // 0 for a tap on WhatsApp, which leaves no contact details
	At     time.Time
	Action string
	UnitID string
	First  attribution.Touch
	Last   attribution.Touch
}

const (
	// firstTouchKept is how long a browser keeps a visitor's first arrival.
	firstTouchKept = 90 * 24 * time.Hour
	// launchCampaigns is how long the launch campaigns ran.
	launchCampaigns = 60 * 24 * time.Hour
	// sameVisit is how long after arriving a visitor is still on that visit.
	sameVisit = time.Hour
)

func (s *Story) visitPick(options []string) string {
	return options[s.visits.IntN(len(options))]
}

// campaign is the campaign a channel was running at a time.
func (s *Story) campaign(channel string, l *Lead, at time.Time) string {
	if at.Before(s.Launch.Add(launchCampaigns)) {
		return "riverside-launch"
	}
	if l != nil && l.wantType == "P" {
		return "penthouse-release"
	}
	weekend := "open-weekend-" + strings.ToLower(at.In(nairobi).Format("Jan"))
	switch channel {
	case attribution.PaidSearch:
		return "riverside-homes"
	case attribution.Email:
		return s.visitPick([]string{"construction-update", weekend})
	}
	return weekend
}

// arrival is a visit through a channel at a time, as the visitor's browser
// records it and the site cleans it. With no landing page given, it is one
// the channel's links lead to.
func (s *Story) arrival(channel string, l *Lead, at time.Time, landing string) attribution.Touch {
	t := attribution.Touch{At: at}
	switch channel {
	case attribution.Search:
		t.Referrer = s.visitPick(searchReferrers)
	case attribution.PaidSearch:
		t.Source, t.Medium, t.Campaign, t.Referrer = "google", "cpc", s.campaign(channel, l, at), "https://www.google.com/"
	case attribution.PaidSocial:
		ad := socialAds[s.visits.IntN(len(socialAds))]
		t.Source, t.Medium, t.Campaign, t.Referrer = ad.source, "paid-social", s.campaign(channel, l, at), ad.referrer
	case attribution.SocialMedia:
		t.Referrer = "https://l.instagram.com/"
	case attribution.Email:
		t.Source, t.Medium, t.Campaign = "newsletter", "email", s.campaign(channel, l, at)
	case attribution.WhatsApp:
		t.Source, t.Medium = "whatsapp", "share"
	}
	t.Landing = landing
	if t.Landing == "" {
		t.Landing = s.landing(channel, t.Campaign, l)
	}
	return attribution.Clean(t, "", at)
}

// landing is a page a channel's links lead to.
func (s *Story) landing(channel, campaign string, l *Lead) string {
	typePage := "/residences"
	if l != nil {
		typePage = "/residences/types/" + l.wantType
	}
	switch {
	case channel == attribution.Email && campaign == "construction-update":
		return "/progress"
	case channel == attribution.Email && strings.HasPrefix(campaign, "open-weekend-"):
		return "/viewings/new"
	case channel == attribution.Search:
		return s.visitPick([]string{"/", "/residences", typePage})
	case channel == attribution.PaidSearch, channel == attribution.WhatsApp:
		return s.visitPick([]string{"/residences", typePage})
	case channel == attribution.PaidSocial:
		return s.visitPick([]string{"/", "/explore", typePage})
	}
	return s.visitPick([]string{"/", "/residences"})
}

// webEnquiry is the enquiry of a lead the website brought in: their first
// arrival, through the channel they are filed under, and the visit they
// wrote from, the same one or a later one.
func (s *Story) webEnquiry(l *Lead) {
	if s.visits.IntN(5) < 2 {
		// They wrote on their first visit.
		l.first = s.arrival(l.Source, l, l.CreatedAt.Add(-time.Duration(3+s.visits.IntN(25))*time.Minute), "")
		l.last = l.first
	} else {
		l.first = s.arrival(l.Source, l, l.CreatedAt.Add(-time.Duration(2+s.visits.IntN(10*24))*time.Hour), "")
		l.last = s.returnVisit(l, l.CreatedAt, "")
	}
	s.Touches = append(s.Touches, Touch{LeadID: l.ID, At: l.CreatedAt, Action: "enquiry", First: l.first, Last: l.last})
}

// returnVisit is a lead coming back to the site shortly before at: by
// typing the address, by searching for Oriel, or the way they first came.
func (s *Story) returnVisit(l *Lead, at time.Time, landing string) attribution.Touch {
	arrived := at.Add(-time.Duration(2+s.visits.IntN(38)) * time.Minute)
	channel := attribution.Direct
	switch r := s.visits.IntN(4); {
	case r == 0:
		channel = attribution.Search
	case r == 1 && l.web:
		channel = l.Source
	}
	return s.arrival(channel, l, arrived, landing)
}

// touch records something a lead sent from the website. A lead still on the
// visit of their last touch sends it from that visit; their first arrival
// is kept for 90 days, as the browser keeps it.
func (s *Story) touch(l *Lead, at time.Time, action, unit, landing string) {
	if l.last.At.IsZero() || at.Sub(l.last.At) >= sameVisit {
		l.last = s.returnVisit(l, at, landing)
	}
	if l.first.At.IsZero() || at.Sub(l.first.At) > firstTouchKept {
		l.first = l.last
	}
	s.Touches = append(s.Touches, Touch{LeadID: l.ID, At: at, Action: action, UnitID: unit, First: l.first, Last: l.last})
}

// tapWhatsApp adds the taps on WhatsApp links over the last two months, most
// of them in the last 30 days. A tap leaves no contact details, so none of
// them belongs to a lead.
func (s *Story) tapWhatsApp() {
	channels := []string{
		attribution.Search, attribution.Search, attribution.Search, attribution.PaidSocial, attribution.PaidSocial,
		attribution.Direct, attribution.Direct, attribution.PaidSearch, attribution.SocialMedia, attribution.Email,
	}
	homes := building.Units()
	for i := 0; i < 19; i++ {
		back := time.Duration(15+s.visits.IntN(30*24*60-30)) * time.Minute
		if i >= 14 {
			back += 30 * 24 * time.Hour
		}
		at := s.Now.Add(-back)
		unit, subject, link, landing := "", "Oriel", "", ""
		if s.visits.IntN(3) > 0 {
			u := homes[s.visits.IntN(len(homes))]
			unit, subject, link, landing = u.ID, "home "+u.ID, "/desk/homes/"+u.ID, "/residences/"+u.ID
		}
		last := s.arrival(s.visitPick(channels), nil, at.Add(-time.Duration(1+s.visits.IntN(20))*time.Minute), landing)
		first := last
		if s.visits.IntN(3) == 0 {
			first = s.arrival(s.visitPick(channels), nil, last.At.Add(-time.Duration(1+s.visits.IntN(20*24))*time.Hour), "")
		}
		s.Touches = append(s.Touches, Touch{At: at, Action: "whatsapp", UnitID: unit, First: first, Last: last})
		s.Activity = append(s.Activity, Activity{At: at, Actor: "visitor", Action: "opened WhatsApp about", Subject: subject, Detail: "from " + last.Channel, Link: link})
	}
}

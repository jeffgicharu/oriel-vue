// Package attribution works out where a visitor came from, as a channel the
// sales team reads at a glance: the campaign in the link they followed, or
// the site that sent them.
//
// The browser keeps two touches. The first is from the visitor's first
// arrival and is kept for 90 days; the last is from the arrival that led to
// what they sent. Both travel with every enquiry, viewing and reservation,
// and the channel is always worked out here, never taken from the browser.
package attribution

import (
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// The channels, in the order the sales desk lists them.
const (
	Search         = "Search"
	PaidSearch     = "Paid search"
	SocialMedia    = "Social media"
	PaidSocial     = "Paid social"
	PropertyPortal = "Property portal"
	Email          = "Email"
	WhatsApp       = "WhatsApp"
	OtherWebsite   = "Other website"
	Direct         = "Direct"
)

// Channels lists every channel a touch can be given.
var Channels = []string{Search, PaidSearch, SocialMedia, PaidSocial, PropertyPortal, Email, WhatsApp, OtherWebsite, Direct}

// Touch is one arrival at the site.
type Touch struct {
	Source   string    // utm_source, such as "instagram"
	Medium   string    // utm_medium, such as "paid-social"
	Campaign string    // utm_campaign, such as "riverside-launch"
	Referrer string    // the host of the site that sent the visitor
	Landing  string    // the path of the first page they saw
	At       time.Time // when they arrived
	Channel  string    // worked out by Classify
}

// Attribution is the first and the last touch behind something a visitor sent.
type Attribution struct {
	First Touch
	Last  Touch
}

// Clean trims a touch the browser sent to what is worth keeping: short
// lower-case campaign fields, the sending site's host without its address,
// and the landing page's path without its query. A referrer that is the site
// itself is dropped. The channel is then worked out afresh.
func Clean(t Touch, siteHost string, now time.Time) Touch {
	out := Touch{
		Source:   clip(strings.ToLower(strings.TrimSpace(t.Source)), 60),
		Medium:   clip(strings.ToLower(strings.TrimSpace(t.Medium)), 60),
		Campaign: clip(strings.ToLower(strings.TrimSpace(t.Campaign)), 80),
		Referrer: hostOf(t.Referrer),
		Landing:  pathOf(t.Landing),
		At:       t.At,
	}
	if out.Referrer != "" && strings.EqualFold(out.Referrer, siteHost) {
		out.Referrer = ""
	}
	// A time from the future, or from more than a year ago, is not believed.
	if out.At.IsZero() || out.At.After(now.Add(5*time.Minute)) || out.At.Before(now.AddDate(-1, 0, 0)) {
		out.At = now
	}
	out.Channel = Classify(out)
	return out
}

// Classify names the channel a touch came through.
func Classify(t Touch) string {
	src, med, ref := t.Source, t.Medium, t.Referrer
	switch {
	case in(med, "cpc", "ppc", "paid", "paidsearch", "paid-search", "paid_search", "sem"):
		if isSocial(src) {
			return PaidSocial
		}
		return PaidSearch
	case in(med, "paid-social", "paidsocial", "paid_social", "social-paid", "social_paid"):
		return PaidSocial
	case med == "email" || in(src, "email", "newsletter"):
		return Email
	case src == "whatsapp" || med == "whatsapp" || strings.Contains(ref, "whatsapp"):
		return WhatsApp
	case in(med, "portal", "listing", "classified", "classifieds"):
		return PropertyPortal
	case in(med, "social", "social-media", "social_media", "organic-social") || isSocial(src):
		return SocialMedia
	case src != "":
		if isSearch(src) {
			return Search
		}
		return OtherWebsite
	case isSearch(ref):
		return Search
	case isSocial(ref):
		return SocialMedia
	case ref != "":
		return OtherWebsite
	}
	return Direct
}

func in(v string, options ...string) bool {
	for _, o := range options {
		if v == o {
			return true
		}
	}
	return false
}

var socialNames = []string{"facebook", "fb", "instagram", "ig", "twitter", "x", "t.co", "linkedin", "lnkd.in", "tiktok", "youtube", "threads", "pinterest", "snapchat"}

// isSocial reports a social network, named in a campaign or as a host.
func isSocial(v string) bool {
	if v == "" {
		return false
	}
	for _, name := range socialNames {
		if v == name || v == name+".com" || strings.HasSuffix(v, "."+name+".com") || (strings.Contains(name, ".") && (v == name || strings.HasSuffix(v, "."+name))) {
			return true
		}
	}
	return false
}

var searchNames = []string{"google", "bing", "duckduckgo", "yahoo", "ecosia", "yandex", "baidu", "brave"}

// isSearch reports a search engine, named in a campaign or as a host such as
// www.google.co.ke or search.brave.com.
func isSearch(v string) bool {
	if v == "" {
		return false
	}
	for _, name := range searchNames {
		if v == name || strings.HasPrefix(v, name+".") || strings.Contains(v, "."+name+".") {
			return true
		}
	}
	return false
}

// hostOf is the host of a referring address, lower-case and without "www.".
func hostOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return clip(strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."), 100)
}

// pathOf is the path of a landing page, without its query or fragment.
func pathOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return ""
	}
	return clip(u.Path, 120)
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

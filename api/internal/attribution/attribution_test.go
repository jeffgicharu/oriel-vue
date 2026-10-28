package attribution

import (
	"testing"
	"time"
)

func TestChannels(t *testing.T) {
	for _, c := range []struct {
		name string
		t    Touch
		want string
	}{
		{"nothing at all", Touch{}, Direct},
		{"a Google search", Touch{Referrer: "www.google.co.ke"}, Search},
		{"a Bing search", Touch{Referrer: "bing.com"}, Search},
		{"a search ad", Touch{Source: "google", Medium: "cpc", Campaign: "riverside-launch"}, PaidSearch},
		{"an Instagram ad", Touch{Source: "instagram", Medium: "paid-social"}, PaidSocial},
		{"a Facebook ad sent as cpc", Touch{Source: "facebook", Medium: "cpc"}, PaidSocial},
		{"an Instagram post", Touch{Referrer: "l.instagram.com"}, SocialMedia},
		{"a link on X", Touch{Referrer: "t.co"}, SocialMedia},
		{"a social campaign", Touch{Source: "linkedin", Medium: "social"}, SocialMedia},
		{"the newsletter", Touch{Source: "newsletter", Medium: "email"}, Email},
		{"a WhatsApp share", Touch{Source: "whatsapp"}, WhatsApp},
		{"a portal listing", Touch{Source: "portal", Medium: "listing", Campaign: "0704"}, PropertyPortal},
		{"another website", Touch{Referrer: "nairobi-living.example.org"}, OtherWebsite},
		{"an unknown campaign", Touch{Source: "partner", Medium: "referral"}, OtherWebsite},
	} {
		if got := Classify(c.t); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCleanKeepsOnlyWhatIsWorthKeeping(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	got := Clean(Touch{
		Source:   "  Instagram ",
		Medium:   "Paid-Social",
		Referrer: "https://www.Instagram.com/p/abc?x=1",
		Landing:  "/residences/0704?utm_source=instagram#plan",
		At:       now.Add(-time.Hour),
	}, "oriel-vue.example.com", now)
	if got.Source != "instagram" || got.Medium != "paid-social" || got.Referrer != "instagram.com" || got.Landing != "/residences/0704" {
		t.Fatalf("cleaned to %+v", got)
	}
	if got.Channel != PaidSocial {
		t.Fatalf("channel %q", got.Channel)
	}

	own := Clean(Touch{Referrer: "https://oriel-vue.example.com/residences"}, "oriel-vue.example.com", now)
	if own.Referrer != "" || own.Channel != Direct {
		t.Fatalf("the site's own pages count as a referrer: %+v", own)
	}

	odd := Clean(Touch{Landing: "//evil.example.com/x", At: now.AddDate(2, 0, 0)}, "oriel-vue.example.com", now)
	if odd.Landing != "" {
		t.Fatalf("landing %q was kept", odd.Landing)
	}
	if !odd.At.Equal(now) {
		t.Fatalf("a time in the future was believed: %s", odd.At)
	}

	long := Clean(Touch{Campaign: string(make([]byte, 500))}, "", now)
	if len(long.Campaign) > 80 {
		t.Fatalf("campaign kept at %d characters", len(long.Campaign))
	}
}

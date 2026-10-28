package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// deskCookie holds a member of staff's session. It covers the whole site,
// because the desk's pages are rendered from /desk and read from /api/desk.
const deskCookie = "oriel_desk"

func (a *App) deskCookieFor(token string) *http.Cookie {
	return &http.Cookie{
		Name:     deskCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(store.SessionLength.Seconds()),
		HttpOnly: true,
		Secure:   strings.HasPrefix(a.cfg.SiteURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *App) expiredDeskCookie() *http.Cookie {
	c := a.deskCookieFor("")
	c.MaxAge = -1
	return c
}

// EnterSignature is the signature in a one-click sign-in link for a role.
func EnterSignature(deskKey []byte, role string) string {
	mac := hmac.New(sha256.New, deskKey)
	mac.Write([]byte("enter:" + role))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
}

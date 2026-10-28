package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/jeffgicharu/oriel-vue/api/internal/copies"
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

// SignInInput is the desk's sign-in form.
type SignInInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeskSession is who is signed in to the desk.
type DeskSession struct {
	Staff DeskStaff `json:"staff"`
	// Manager is true for the sales manager, who may change prices and
	// statuses, decide reservations and see every lead.
	Manager bool `json:"manager"`
	// Waiting is how many reservation requests wait for a decision, for the
	// badge on the desk's menu.
	Waiting int `json:"waiting"`
}

func (a *App) sessionOf(r *http.Request, db *store.DB, s store.Staff) (DeskSession, error) {
	out := DeskSession{Staff: deskStaffOf(s), Manager: s.Manager()}
	err := db.QueryRowContext(r.Context(), `SELECT count(*) FROM reservations WHERE status = 'requested'`).Scan(&out.Waiting)
	return out, err
}

// deskSignIn signs a member of the team in with their password, on the
// visitor's own copy, which is made if need be.
func (a *App) deskSignIn(w http.ResponseWriter, r *http.Request) {
	var in SignInInput
	if !a.readJSON(w, r, &in) {
		return
	}
	if !a.formAllowed(r) {
		w.Header().Set("Retry-After", "60")
		a.problem(w, http.StatusTooManyRequests, "rate-limited", "Too many attempts. Please wait a minute and try again.", nil)
		return
	}
	staff, err := a.db(r).CheckSignIn(r.Context(), in.Username, in.Password)
	if errors.Is(err, store.ErrBadSignIn) {
		a.problem(w, http.StatusUnauthorized, "bad-sign-in", "That username and password don’t match.", nil)
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	db := a.writableOrAnswer(w, r)
	if db == nil {
		return
	}
	token, err := db.StartSession(r.Context(), staff.ID, a.now())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	out, err := a.sessionOf(r, db, staff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	http.SetCookie(w, a.deskCookieFor(token))
	writeJSON(w, http.StatusOK, out)
}

func (a *App) deskSession(w http.ResponseWriter, r *http.Request) {
	out, err := a.sessionOf(r, a.db(r), staffFrom(r.Context()))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// deskSignOut ends the session, if there is one, and forgets the cookie.
func (a *App) deskSignOut(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(deskCookie); err == nil && !a.db(r).ReadOnly() {
		if err := a.db(r).EndSession(r.Context(), c.Value); err != nil {
			a.log.Error("signing out", "err", err)
		}
	}
	http.SetCookie(w, a.expiredDeskCookie())
	w.WriteHeader(http.StatusNoContent)
}

// deskEnter is a one-click sign-in: a fresh copy of the business, signed in
// as the manager or an agent, and then the desk.
func (a *App) deskEnter(w http.ResponseWriter, r *http.Request) {
	role, sig := r.PathValue("role"), r.PathValue("sig")
	if (role != "manager" && role != "agent") || !hmac.Equal([]byte(sig), []byte(EnterSignature(a.cfg.DeskKey, role))) {
		a.notFound(w, "There is nothing at "+r.URL.Path+".")
		return
	}
	check := a.isReleaseCheck(r)
	if !check && !a.copyLimiter.allow(clientIP(r), a.now()) {
		a.busy(w)
		return
	}
	prefix := ""
	if check {
		prefix = "chk"
	}
	h, err := a.copies.Create(prefix)
	if errors.Is(err, copies.ErrFull) {
		a.busy(w)
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	defer h.Release()
	staff, err := h.DB().FirstWithRole(r.Context(), role)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	token, err := h.DB().StartSession(r.Context(), staff.ID, a.now())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	http.SetCookie(w, a.copyCookieFor(h.ID()))
	http.SetCookie(w, a.deskCookieFor(token))
	// Without a body: the browser goes straight on to the desk.
	w.Header().Set("Location", "/desk")
	w.WriteHeader(http.StatusSeeOther)
}

// DeskTeam is the sales team, the manager first.
type DeskTeam struct {
	Team []DeskStaff `json:"team"`
}

func (a *App) deskTeam(w http.ResponseWriter, r *http.Request) {
	team, err := a.db(r).Team(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	out := DeskTeam{Team: []DeskStaff{}}
	for _, s := range team {
		out.Team = append(out.Team, deskStaffOf(s))
	}
	writeJSON(w, http.StatusOK, out)
}

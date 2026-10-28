package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"modernc.org/sqlite"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// fieldErrors maps a form field to what to say beside it.
type fieldErrors = map[string]string

// ContactInput is the name, email and phone every public form asks for, and
// the field people never see, which only robots fill in.
type ContactInput struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Website string `json:"website"`
}

// SourceInput is where the visitor came from, as their browser recorded it.
type SourceInput struct {
	First TouchInput `json:"first"`
	Last  TouchInput `json:"last"`
}

// TouchInput is one arrival, as the browser recorded it.
type TouchInput struct {
	Source   string `json:"source"`
	Medium   string `json:"medium"`
	Campaign string `json:"campaign"`
	Referrer string `json:"referrer"`
	Landing  string `json:"landing"`
	At       string `json:"at"`
}

var phoneDigits = regexp.MustCompile(`^\+?[0-9]{9,15}$`)

// contact checks the name, email and phone, noting what is wrong with each.
func contact(in ContactInput, errs fieldErrors) store.Contact {
	c := store.Contact{
		Name:  strings.Join(strings.Fields(in.Name), " "),
		Email: strings.TrimSpace(in.Email),
		Phone: strings.TrimSpace(in.Phone),
	}
	switch n := utf8.RuneCountInString(c.Name); {
	case n == 0:
		errs["name"] = "Please tell us your name."
	case n < 2:
		errs["name"] = "Please give your full name."
	case n > 80:
		errs["name"] = "That name is too long."
	}
	if c.Email == "" {
		errs["email"] = "Please give an email address."
	} else if a, err := mail.ParseAddress(c.Email); err != nil || a.Address != c.Email || !strings.Contains(c.Email[strings.LastIndex(c.Email, "@")+1:], ".") || len(c.Email) > 120 {
		errs["email"] = "That email address doesn’t look right."
	}
	phone, ok := normalisePhone(c.Phone)
	switch {
	case c.Phone == "":
		errs["phone"] = "Please give a phone number."
	case !ok:
		errs["phone"] = "Please give a phone number, such as 0712 345 678 or +254 712 345 678."
	default:
		c.Phone = phone
	}
	return c
}

// normalisePhone writes a Kenyan number as +254 7XX XXX XXX and leaves
// international numbers in their own form.
func normalisePhone(raw string) (string, bool) {
	s := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "").Replace(raw)
	if !phoneDigits.MatchString(s) {
		return "", false
	}
	switch {
	case strings.HasPrefix(s, "0") && len(s) == 10:
		s = "+254" + s[1:]
	case strings.HasPrefix(s, "254") && len(s) == 12:
		s = "+" + s
	case strings.HasPrefix(s, "7") && len(s) == 9, strings.HasPrefix(s, "1") && len(s) == 9:
		s = "+254" + s
	}
	if strings.HasPrefix(s, "+254") && len(s) == 13 {
		return s[:4] + " " + s[4:7] + " " + s[7:10] + " " + s[10:], true
	}
	if !strings.HasPrefix(s, "+") {
		return "", false
	}
	return s, true
}

// attributionOf cleans what the browser sent about where the visitor came
// from. A browser that sent nothing counts as a direct visit now.
func (a *App) attributionOf(in *SourceInput) *attribution.Attribution {
	now := a.now()
	host := a.siteHost()
	if in == nil {
		t := attribution.Clean(attribution.Touch{At: now}, host, now)
		return &attribution.Attribution{First: t, Last: t}
	}
	touch := func(t TouchInput) attribution.Touch {
		at, _ := time.Parse(time.RFC3339, t.At)
		return attribution.Clean(attribution.Touch{Source: t.Source, Medium: t.Medium, Campaign: t.Campaign, Referrer: t.Referrer, Landing: t.Landing, At: at}, host, now)
	}
	return &attribution.Attribution{First: touch(in.First), Last: touch(in.Last)}
}

func (a *App) siteHost() string {
	return strings.TrimPrefix(strings.TrimPrefix(a.cfg.SiteURL, "https://"), "http://")
}

// formAllowed applies the per-address limit on forms.
func (a *App) formAllowed(r *http.Request) bool {
	return a.isReleaseCheck(r) || a.formLimiter.allow(clientIP(r), a.now())
}

// writableOrAnswer is the database a form may change, making the visitor's
// copy if need be. When it cannot, it has answered and returns nil.
func (a *App) writableOrAnswer(w http.ResponseWriter, r *http.Request) *store.DB {
	db, err := a.writable(w, r)
	if errors.Is(err, errBusy) {
		a.busy(w)
		return nil
	}
	if err != nil {
		a.serverError(w, r, err)
		return nil
	}
	return db
}

// isInternal reports an error that is the server's fault, not the request's.
// The store writes its refusals for people to read; an error from the
// database itself, or a cancelled request, is internal.
func isInternal(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) || errors.Is(err, sql.ErrNoRows) || errors.Is(err, sql.ErrTxDone) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, store.ErrReadOnly)
}

func clipText(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

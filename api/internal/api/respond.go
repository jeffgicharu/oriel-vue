package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
)

// contentSecurityPolicy is for the API's own responses, which are never
// pages: nothing in them may run, load or be framed.
const contentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

// headers sets the security headers every response carries, and asks search
// engines not to list anything.
func (a *App) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		// Every answer depends on the visitor's own copy, so none is shared.
		h.Set("Cache-Control", "private, no-store")
		next.ServeHTTP(w, r)
	})
}

// recoverer turns a panic in a handler into a 500 and a log line.
func (a *App) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				a.log.Error("panic", "path", r.URL.Path, "value", v, "stack", string(debug.Stack()))
				a.serverError(w, r, nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// writeJSON sends a value as JSON with a status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// Problem is an error answer, as RFC 9457 describes. Errors names each form
// field that was wrong and what to say beside it.
type Problem struct {
	Type   string            `json:"type"`
	Title  string            `json:"title"`
	Status int               `json:"status"`
	Detail string            `json:"detail,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

var problemTitles = map[int]string{
	http.StatusBadRequest:            "The request could not be read",
	http.StatusUnauthorized:          "Please sign in",
	http.StatusForbidden:             "Not allowed",
	http.StatusNotFound:              "Not found",
	http.StatusMethodNotAllowed:      "Not allowed",
	http.StatusConflict:              "Already taken",
	http.StatusRequestEntityTooLarge: "Too much was sent",
	http.StatusUnsupportedMediaType:  "Send JSON",
	http.StatusUnprocessableEntity:   "Some answers need another look",
	http.StatusTooManyRequests:       "Too many requests",
	http.StatusInternalServerError:   "Something went wrong",
	http.StatusServiceUnavailable:    "Busy just now",
}

// problem sends an error answer. kind becomes the type, a short name the
// browser can act on, such as "validation" or "slot-taken".
func (a *App) problem(w http.ResponseWriter, status int, kind, detail string, errs map[string]string) {
	p := Problem{Type: "https://oriel-vue.jeffgicharu.com/problems/" + kind, Title: problemTitles[status], Status: status, Detail: detail, Errors: errs}
	if p.Title == "" {
		p.Title = http.StatusText(status)
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(p)
}

func (a *App) notFound(w http.ResponseWriter, what string) {
	a.problem(w, http.StatusNotFound, "not-found", what, nil)
}

func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		a.log.Error("request failed", "path", r.URL.Path, "err", err)
	}
	a.problem(w, http.StatusInternalServerError, "server", "Something went wrong on our side. Please try again in a moment.", nil)
}

// busy is the answer when no private copy can be made just now.
func (a *App) busy(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	a.problem(w, http.StatusServiceUnavailable, "busy", "Oriel is very busy just now. Please try again in a minute.", nil)
}

// tooMany is the answer when one address has sent too many forms.
func (a *App) tooMany(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	a.problem(w, http.StatusTooManyRequests, "rate-limited", "You have sent several forms in a short time. Please wait a minute and try again.", nil)
}

// maxBody is the largest request body any form needs.
const maxBody = 32 << 10

// readJSON decodes a request body into v. On failure it has already
// answered, and returns false.
func (a *App) readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		a.problem(w, http.StatusUnsupportedMediaType, "content-type", "Send the form as JSON.", nil)
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			a.problem(w, http.StatusRequestEntityTooLarge, "too-large", "That form is too long.", nil)
			return false
		}
		a.problem(w, http.StatusBadRequest, "bad-json", "The form could not be read.", nil)
		return false
	}
	if _, err := dec.Token(); err != io.EOF {
		a.problem(w, http.StatusBadRequest, "bad-json", "The form could not be read.", nil)
		return false
	}
	return true
}

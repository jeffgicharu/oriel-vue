package api

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// EnquiryInput is a question from the enquiry form or a home's page.
type EnquiryInput struct {
	ContactInput
	Message  string       `json:"message"`
	Interest string       `json:"interest"`
	Home     string       `json:"home"`
	Source   *SourceInput `json:"source"`
}

// EnquiryReceived thanks the visitor by name.
type EnquiryReceived struct {
	FirstName string `json:"firstName"`
	Home      *Home  `json:"home"`
}

func (a *App) enquire(w http.ResponseWriter, r *http.Request) {
	var in EnquiryInput
	if !a.readJSON(w, r, &in) {
		return
	}
	// A robot filled in the field people never see. It is thanked and
	// nothing is kept.
	if in.Website != "" {
		writeJSON(w, http.StatusCreated, EnquiryReceived{FirstName: "there"})
		return
	}
	errs := fieldErrors{}
	c := contact(in.ContactInput, errs)
	message := strings.TrimSpace(in.Message)
	switch {
	case message == "":
		errs["message"] = "Please write your question."
	case utf8.RuneCountInString(message) > 1500:
		errs["message"] = "Please keep your question under 1,500 characters."
	}
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Some answers need another look.", errs)
		return
	}
	if !a.formAllowed(r) {
		a.tooMany(w)
		return
	}
	db := a.writableOrAnswer(w, r)
	if db == nil {
		return
	}
	var home *store.Home
	if in.Home != "" {
		if cat, err := db.LoadCatalog(r.Context()); err == nil {
			if h, ok := cat.Home(in.Home); ok {
				home = &h
			}
		}
	}
	c.Interest = clipText(strings.TrimSpace(in.Interest), 80)
	unitID := ""
	if home != nil {
		unitID = home.ID
		if c.Interest == "" {
			c.Interest = "Home " + home.ID
		}
	}
	c.Attribution = a.attributionOf(in.Source)
	if _, err := db.Enquire(r.Context(), c, unitID, message, a.now()); err != nil {
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead", Text: c.Name + " sent a question"})
	out := EnquiryReceived{FirstName: strings.Fields(c.Name)[0]}
	if home != nil {
		h := homeOf(*home)
		out.Home = &h
	}
	writeJSON(w, http.StatusCreated, out)
}

// WhatsAppInput is a tap on a WhatsApp link: the home it was about, if any,
// and where the visitor came from.
type WhatsAppInput struct {
	Home   string       `json:"home"`
	Source *SourceInput `json:"source"`
}

// whatsapp records a tap on a WhatsApp link. The conversation itself happens
// in WhatsApp; this tells the sales team one may be coming, and from where.
// Like anything a visitor sends, it goes into their own copy.
func (a *App) whatsapp(w http.ResponseWriter, r *http.Request) {
	var in WhatsAppInput
	if !a.readJSON(w, r, &in) {
		return
	}
	if !a.formAllowed(r) {
		a.tooMany(w)
		return
	}
	db := a.writableOrAnswer(w, r)
	if db == nil {
		return
	}
	unitID := ""
	if in.Home != "" {
		if cat, err := db.LoadCatalog(r.Context()); err == nil {
			if h, ok := cat.Home(in.Home); ok {
				unitID = h.ID
			}
		}
	}
	if err := db.RecordWhatsApp(r.Context(), unitID, *a.attributionOf(in.Source), a.now()); err != nil {
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "activity", Text: "A visitor opened WhatsApp"})
	w.WriteHeader(http.StatusNoContent)
}

package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jeffgicharu/oriel-vue/api/internal/store"
	"github.com/jeffgicharu/oriel-vue/api/internal/viewings"
)

// Calendar is the days a visitor can book, each with its times.
type Calendar struct {
	Minutes int           `json:"minutes"`
	Days    []CalendarDay `json:"days"`
}

// CalendarDay is one day of the booking calendar.
type CalendarDay struct {
	Date   string `json:"date"`
	Open   bool   `json:"open"`
	Reason string `json:"reason"`
	// Blocked is a day closed for a holiday or by the sales team, rather
	// than the suite's weekly day off.
	Blocked bool           `json:"blocked"`
	Free    int            `json:"free"`
	Slots   []CalendarSlot `json:"slots"`
}

// CalendarSlot is a viewing time with the places left in it.
type CalendarSlot struct {
	Start string `json:"start"`
	Time  string `json:"time"`
	Left  int    `json:"left"`
}

// ViewingInput is the booking form.
type ViewingInput struct {
	ContactInput
	Slot   string       `json:"slot"`
	Party  int          `json:"party"`
	Notes  string       `json:"notes"`
	Home   string       `json:"home"`
	Source *SourceInput `json:"source"`
}

// Viewing is a booked visit, as its page shows it.
type Viewing struct {
	Ref     string `json:"ref"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Start   string `json:"start"`
	Minutes int    `json:"minutes"`
	Party   int    `json:"party"`
	Notes   string `json:"notes"`
	Status  string `json:"status"`
	Home    *Home  `json:"home"`
	// Agent is the consultant who will show the visitor round.
	Agent     string `json:"agent"`
	CreatedAt string `json:"createdAt"`
}

// slotLayout is how a viewing time is written in the API: Nairobi time,
// to the minute.
const slotLayout = "2006-01-02T15:04"

func (a *App) calendar(w http.ResponseWriter, r *http.Request) {
	days, err := a.db(r).Calendar(r.Context(), a.now())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	out := Calendar{Minutes: int(viewings.Length / time.Minute), Days: []CalendarDay{}}
	for _, d := range days {
		day := CalendarDay{Date: d.Date.Format(time.DateOnly), Open: d.Open, Reason: d.Reason, Blocked: d.Blocked, Free: d.Free(), Slots: []CalendarSlot{}}
		for _, s := range d.Slots {
			day.Slots = append(day.Slots, CalendarSlot{Start: s.Start.In(viewings.Nairobi).Format(slotLayout), Time: s.Start.In(viewings.Nairobi).Format("15:04"), Left: s.Left})
		}
		out.Days = append(out.Days, day)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) bookViewing(w http.ResponseWriter, r *http.Request) {
	var in ViewingInput
	if !a.readJSON(w, r, &in) {
		return
	}
	if in.Website != "" {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Some answers need another look.", fieldErrors{"slot": "Please choose a time."})
		return
	}
	errs := fieldErrors{}
	c := contact(in.ContactInput, errs)
	start, err := time.ParseInLocation(slotLayout, in.Slot, viewings.Nairobi)
	if err != nil {
		errs["slot"] = "Please choose a time."
	}
	if in.Party < 1 || in.Party > 6 {
		errs["party"] = "Viewings are for one to six people."
	}
	notes := strings.TrimSpace(in.Notes)
	if utf8.RuneCountInString(notes) > 600 {
		errs["notes"] = "Please keep this under 600 characters."
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
	homeID := ""
	if in.Home != "" {
		if cat, err := db.LoadCatalog(r.Context()); err == nil {
			if _, ok := cat.Home(in.Home); ok {
				homeID = in.Home
			}
		}
	}
	c.Interest = "Viewing"
	if homeID != "" {
		c.Interest = "Home " + homeID
	}
	c.Attribution = a.attributionOf(in.Source)
	v, err := db.BookViewing(r.Context(), store.ViewingRequest{Contact: c, Start: start, PartySize: in.Party, UnitID: homeID, Notes: notes}, a.now())
	switch {
	case errors.Is(err, store.ErrSlotGone):
		a.problem(w, http.StatusConflict, "slot-taken", "That time has just been taken.", fieldErrors{"slot": "That time has just been taken. Please choose another."})
		return
	case errors.Is(err, store.ErrSlotClosed):
		a.problem(w, http.StatusUnprocessableEntity, "validation", "The sales suite is not open then.", fieldErrors{"slot": "The sales suite is not open at that time. Please choose another."})
		return
	case err != nil:
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead", Text: c.Name + " booked a viewing"})
	writeJSON(w, http.StatusCreated, a.viewingOf(r, db, v))
}

func (a *App) viewing(w http.ResponseWriter, r *http.Request) {
	db := a.db(r)
	v, err := db.ViewingByRef(r.Context(), r.PathValue("ref"))
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "There is no viewing "+r.PathValue("ref")+" in this browser.")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a.viewingOf(r, db, v))
}

func (a *App) cancelViewing(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	db := a.db(r)
	if db.ReadOnly() {
		a.notFound(w, "There is no viewing "+ref+" in this browser.")
		return
	}
	err := db.CancelViewing(r.Context(), ref, a.now(), 0)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "There is no viewing "+ref+" in this browser.")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead", Text: "A viewing was cancelled"})
	v, err := db.ViewingByRef(r.Context(), ref)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a.viewingOf(r, db, v))
}

func (a *App) viewingOf(r *http.Request, db *store.DB, v store.Viewing) Viewing {
	out := Viewing{
		Ref: v.Ref, Name: v.Name, Email: v.Email, Phone: v.Phone,
		Start:   v.StartsAt.In(viewings.Nairobi).Format(slotLayout),
		Minutes: int(viewings.Length / time.Minute),
		Party:   v.PartySize, Notes: v.Notes, Status: v.Status, Agent: v.AgentName,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339),
	}
	if v.UnitID != "" {
		if cat, err := db.LoadCatalog(r.Context()); err == nil {
			if h, ok := cat.Home(v.UnitID); ok {
				home := homeOf(h)
				out.Home = &home
			}
		}
	}
	return out
}

// viewingCalendar is the booking as a calendar file.
func (a *App) viewingCalendar(w http.ResponseWriter, r *http.Request) {
	v, err := a.db(r).ViewingByRef(r.Context(), r.PathValue("ref"))
	if err != nil {
		a.notFound(w, "There is no viewing "+r.PathValue("ref")+" in this browser.")
		return
	}
	stamp := func(t time.Time) string { return t.UTC().Format("20060102T150405Z") }
	desc := "Your viewing at Oriel with the sales team. Reference " + v.Ref + "."
	if v.UnitID != "" {
		desc += " Home " + v.UnitID + "."
	}
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Oriel Residences//Viewings//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + v.Ref + "@" + a.siteHost(),
		"DTSTAMP:" + stamp(a.now()),
		"DTSTART:" + stamp(v.StartsAt),
		"DTEND:" + stamp(v.StartsAt.Add(viewings.Length)),
		"SUMMARY:Viewing at Oriel",
		"LOCATION:Oriel sales suite\\, Riverside\\, Nairobi",
		"DESCRIPTION:" + desc,
		"URL:" + a.cfg.SiteURL + "/viewings/" + v.Ref,
		"STATUS:CONFIRMED",
		"BEGIN:VALARM",
		"TRIGGER:-PT2H",
		"ACTION:DISPLAY",
		"DESCRIPTION:Viewing at Oriel in two hours",
		"END:VALARM",
		"END:VEVENT",
		"END:VCALENDAR",
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="oriel-viewing-%s.ics"`, strings.ToLower(v.Ref)))
	fmt.Fprint(w, strings.Join(lines, "\r\n")+"\r\n")
}

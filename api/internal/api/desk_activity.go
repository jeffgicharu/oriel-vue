package api

import (
	"net/http"
	"strconv"
)

// activityPage is how many lines of the log come at a time.
const activityPage = 60

// DeskActivity is a page of the desk's log: who changed what, and when.
type DeskActivity struct {
	// Kind is whose activity: "staff" (the team), "visitor" (the website),
	// "system" (the desk itself), or empty for everything.
	Kind    string              `json:"kind"`
	Offset  int                 `json:"offset"`
	Entries []DeskActivityEntry `json:"entries"`
	// More is true when there are older lines; Next is the offset that
	// reads them.
	More bool `json:"more"`
	Next int  `json:"next"`
}

// DeskActivityEntry is a line of the log. Action marks where the subject
// goes with {}; without a mark the subject comes last.
type DeskActivityEntry struct {
	ID      int64  `json:"id"`
	At      string `json:"at"`
	Actor   string `json:"actor"`
	Staff   string `json:"staff"`
	Colour  string `json:"colour"`
	Action  string `json:"action"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
	// Link is the desk's page about the subject, if it has one.
	Link string `json:"link"`
}

func (a *App) deskActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind != "staff" && kind != "visitor" && kind != "system" {
		kind = ""
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	entries, err := a.db(r).ActivityLog(r.Context(), kind, offset, activityPage+1)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	out := DeskActivity{Kind: kind, Offset: offset, Entries: []DeskActivityEntry{}}
	if len(entries) > activityPage {
		out.More, out.Next, entries = true, offset+activityPage, entries[:activityPage]
	}
	for _, e := range entries {
		out.Entries = append(out.Entries, DeskActivityEntry{
			ID: e.ID, At: localTime(e.At), Actor: e.Actor, Staff: e.StaffName, Colour: e.Colour,
			Action: e.Action, Subject: e.Subject, Detail: e.Detail, Link: e.Link,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

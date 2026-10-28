package api

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// DeskLeadFilter is the leads list's filters, as they were applied.
type DeskLeadFilter struct {
	Stage string `json:"stage"`
	// Agent is the member whose leads are shown, or 0 for everyone's. An
	// agent always sees only their own.
	Agent  int64  `json:"agent"`
	Q      string `json:"q"`
	Source string `json:"source"`
	Sort   string `json:"sort"`
	View   string `json:"view"`
}

// DeskLeads is the leads page, as a table or as the pipeline board.
type DeskLeads struct {
	Filter DeskLeadFilter `json:"filter"`
	// Leads is the table, up to 300; it is empty on the board.
	Leads []DeskLead `json:"leads"`
	// More is true when more leads match than the table holds.
	More bool `json:"more"`
	// Columns is the board, a column for each stage in pipeline order; it is
	// empty for the table.
	Columns []DeskLeadColumn `json:"columns"`
	// Counts is how many leads are at each stage, and Total how many there
	// are: the agent's own, or everyone's for the manager.
	Counts map[string]int `json:"counts"`
	Total  int            `json:"total"`
	// Sources is every source leads have come from, the commonest first.
	Sources []string `json:"sources"`
	// CSV is the same list as a spreadsheet.
	CSV string `json:"csv"`
}

// DeskLeadColumn is one stage of the board.
type DeskLeadColumn struct {
	Stage string     `json:"stage"`
	Title string     `json:"title"`
	Leads []DeskLead `json:"leads"`
	// More is how many leads at the stage the board leaves out: the won and
	// lost columns show only the most recent.
	More int `json:"more"`
}

// DeskLeadPage is one lead's page.
type DeskLeadPage struct {
	Lead DeskLead `json:"lead"`
	// Timeline is the lead's history, newest first.
	Timeline     []DeskTimelineEntry `json:"timeline"`
	Viewings     []DeskViewing       `json:"viewings"`
	Reservations []DeskReservation   `json:"reservations"`
	// Touches is what the lead sent from the website, newest first, each
	// with where they came from.
	Touches []DeskTouch `json:"touches"`
	// Notice says what a change just did; it is empty on a read.
	Notice string `json:"notice"`
}

// DeskTimelineEntry is a line in a lead's history.
type DeskTimelineEntry struct {
	At    string `json:"at"`
	Kind  string `json:"kind"`
	Body  string `json:"body"`
	Home  string `json:"home"`
	Staff string `json:"staff"`
}

// DeskTouch is something a lead sent from the website.
type DeskTouch struct {
	At string `json:"at"`
	// Action is "enquiry", "viewing" or "reservation".
	Action string `json:"action"`
	Home   string `json:"home"`
	// Last is the arrival that led to it; First is the visitor's first
	// arrival, which their browser keeps for 90 days.
	Last  DeskArrival `json:"last"`
	First DeskArrival `json:"first"`
}

// DeskArrival is one arrival at the site.
type DeskArrival struct {
	Channel  string `json:"channel"`
	Source   string `json:"source"`
	Medium   string `json:"medium"`
	Campaign string `json:"campaign"`
	// Referrer is the host of the site that sent the visitor.
	Referrer string `json:"referrer"`
	Landing  string `json:"landing"`
	At       string `json:"at"`
}

func arrivalOf(t attribution.Touch) DeskArrival {
	return DeskArrival{Channel: t.Channel, Source: t.Source, Medium: t.Medium, Campaign: t.Campaign, Referrer: t.Referrer, Landing: t.Landing, At: localTime(t.At)}
}

// DeskLeadInput is a lead added by hand: someone who called, walked in or
// was referred.
type DeskLeadInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	// Source is where they came from, such as Walk-in, Phone call or
	// Referral; Walk-in when left out.
	Source   string `json:"source"`
	Interest string `json:"interest"`
	// Buying is "home" or "investment".
	Buying string `json:"buying"`
	// Agent is who looks after the lead. Only the manager chooses; an
	// agent's new lead is their own, as is the manager's when left out.
	Agent int64  `json:"agent"`
	Note  string `json:"note"`
}

// DeskNoteInput is a note on a lead.
type DeskNoteInput struct {
	Body string `json:"body"`
}

// DeskStageInput moves a lead to a stage. A lost lead needs a reason.
type DeskStageInput struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
}

// DeskFollowUpInput sets the day to follow a lead up on, or clears it when
// empty.
type DeskFollowUpInput struct {
	Day string `json:"day"`
}

// DeskAssignInput gives a lead to a member of the team.
type DeskAssignInput struct {
	Agent int64 `json:"agent"`
}

// DeskMoveInput is a card dropped on the board: its stage, and the cards it
// now sits between in that column (0 at either end).
type DeskMoveInput struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
	After  int64  `json:"after"`
	Before int64  `json:"before"`
}

// boardLimit is how many leads the won and lost columns show.
const boardLimit = 24

// tableLimit is how many leads the table shows.
const tableLimit = 300

// leadFilter reads the list's filters; an agent always sees only their own.
func leadFilter(r *http.Request, staff store.Staff) store.LeadFilter {
	q := r.URL.Query()
	f := store.LeadFilter{
		Query:  clipText(strings.TrimSpace(q.Get("q")), 80),
		Source: clipText(strings.TrimSpace(q.Get("source")), 40),
		Sort:   "recent",
	}
	switch q.Get("sort") {
	case "name", "follow", "board":
		f.Sort = q.Get("sort")
	}
	if store.StageTitle(q.Get("stage")) != "" {
		f.Stage = q.Get("stage")
	}
	if staff.Manager() {
		if n, err := strconv.ParseInt(q.Get("agent"), 10, 64); err == nil && n > 0 {
			f.AgentID = n
		}
	} else {
		f.AgentID = staff.ID
	}
	return f
}

// csvLink is the address of the spreadsheet for the same filters.
func csvLink(f store.LeadFilter, staff store.Staff) string {
	v := url.Values{}
	if f.Query != "" {
		v.Set("q", f.Query)
	}
	if f.Stage != "" {
		v.Set("stage", f.Stage)
	}
	if f.AgentID > 0 && staff.Manager() {
		v.Set("agent", strconv.FormatInt(f.AgentID, 10))
	}
	if f.Source != "" {
		v.Set("source", f.Source)
	}
	if f.Sort != "recent" {
		v.Set("sort", f.Sort)
	}
	link := "/api/desk/leads.csv"
	if len(v) > 0 {
		link += "?" + v.Encode()
	}
	return link
}

func (a *App) deskLeads(w http.ResponseWriter, r *http.Request) {
	staff := staffFrom(r.Context())
	db := a.db(r)
	f := leadFilter(r, staff)
	view := "table"
	if r.URL.Query().Get("view") == "board" {
		view = "board"
	}
	out := DeskLeads{
		Filter:  DeskLeadFilter{Stage: f.Stage, Agent: f.AgentID, Q: f.Query, Source: f.Source, Sort: f.Sort, View: view},
		Leads:   []DeskLead{},
		Columns: []DeskLeadColumn{},
		Counts:  map[string]int{},
		Sources: []string{},
		CSV:     csvLink(f, staff),
	}
	counts, err := db.LeadCounts(r.Context(), agentScope(staff))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	for _, stage := range store.Stages {
		out.Counts[stage] = counts[stage]
		out.Total += counts[stage]
	}
	sources, err := db.LeadSources(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	out.Sources = append(out.Sources, sources...)

	if view == "board" {
		out.Filter.Stage, out.Filter.Sort = "", "board"
		for _, stage := range store.Stages {
			board := f
			board.Stage, board.Sort, board.Limit = stage, "board", 0
			leads, err := db.Leads(r.Context(), board)
			if err != nil {
				a.serverError(w, r, err)
				return
			}
			col := DeskLeadColumn{Stage: stage, Title: store.StageTitle(stage), Leads: a.deskLeadsOf(leads)}
			if (stage == "won" || stage == "lost") && len(leads) > boardLimit {
				// The most recently moved, rather than the board's order.
				board.Sort, board.Limit = "recent", boardLimit
				recent, err := db.Leads(r.Context(), board)
				if err != nil {
					a.serverError(w, r, err)
					return
				}
				col.Leads, col.More = a.deskLeadsOf(recent), len(leads)-boardLimit
			}
			out.Columns = append(out.Columns, col)
		}
		writeJSON(w, http.StatusOK, out)
		return
	}

	f.Limit = tableLimit + 1
	list, err := db.Leads(r.Context(), f)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if len(list) > tableLimit {
		out.More, list = true, list[:tableLimit]
	}
	out.Leads = a.deskLeadsOf(list)
	writeJSON(w, http.StatusOK, out)
}

// deskLeadsCSV is the same list as a spreadsheet, with where each lead first
// came from on the website.
func (a *App) deskLeadsCSV(w http.ResponseWriter, r *http.Request) {
	db := a.db(r)
	list, err := db.Leads(r.Context(), leadFilter(r, staffFrom(r.Context())))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	first, err := db.FirstTouches(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	nairobi := store.Nairobi()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="oriel-leads-`+dateOf(a.now())+`.csv"`)
	out := csv.NewWriter(w)
	out.Write([]string{"Name", "Email", "Phone", "Stage", "Agent", "Source", "Channel", "Campaign", "Interest", "Budget (KES)", "Buying", "Follow up on", "Lost because", "First contact", "Last activity"})
	for _, l := range list {
		budget := ""
		if l.Budget > 0 {
			budget = strconv.FormatInt(l.Budget, 10)
		}
		t := first[l.ID]
		out.Write([]string{
			csvSafe(l.Name), l.Email, l.Phone, store.StageTitle(l.Stage), l.AgentName, csvSafe(l.Source), t.Channel, csvSafe(t.Campaign),
			csvSafe(l.Interest), budget, l.BuyingFor, l.FollowUpOn, csvSafe(l.LostReason),
			l.CreatedAt.In(nairobi).Format("2006-01-02 15:04"), l.UpdatedAt.In(nairobi).Format("2006-01-02 15:04"),
		})
	}
	out.Flush()
}

// csvSafe keeps a spreadsheet from reading a cell as a formula.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func (a *App) deskAddLead(w http.ResponseWriter, r *http.Request) {
	var in DeskLeadInput
	if !a.readJSON(w, r, &in) {
		return
	}
	staff := staffFrom(r.Context())
	db := a.db(r)
	errs := fieldErrors{}
	c := contact(ContactInput{Name: in.Name, Email: in.Email, Phone: in.Phone}, errs)
	note := strings.TrimSpace(in.Note)
	tooLong(errs, "note", note, 600)
	var agent int64
	if staff.Manager() && in.Agent != 0 {
		member, err := teamMember(r, db, in.Agent)
		if err != nil {
			a.serverError(w, r, err)
			return
		}
		if member == nil {
			errs["agent"] = "Please choose an agent."
		}
		agent = in.Agent
	}
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Some answers need another look.", errs)
		return
	}
	c.Source = clipText(strings.TrimSpace(in.Source), 40)
	c.Interest = clipText(strings.TrimSpace(in.Interest), 80)
	c.Buying = "home"
	if in.Buying == "investment" {
		c.Buying = "investment"
	}
	id, err := db.AddLead(r.Context(), store.NewLead{Contact: c, Note: note, AgentID: agent}, staff, a.now())
	var exists store.LeadExistsError
	if errors.As(err, &exists) {
		text := "A lead with this email address is already on the desk."
		a.problem(w, http.StatusConflict, "lead-exists", text, fieldErrors{"email": text})
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead-changed", ID: strconv.FormatInt(id, 10)})
	l, err := db.LeadByID(r.Context(), id)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	page, err := a.leadPage(r, l, c.Name+" added.")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, page)
}

// teamMember finds a member of the team by id, or nil.
func teamMember(r *http.Request, db *store.DB, id int64) (*store.Staff, error) {
	team, err := db.Team(r.Context())
	if err != nil {
		return nil, err
	}
	for _, m := range team {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, nil
}

// leadFromPath is the lead the address names, if the member may see it.
func (a *App) leadFromPath(w http.ResponseWriter, r *http.Request) (store.Lead, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		a.notFound(w, "No such lead.")
		return store.Lead{}, false
	}
	l, err := a.db(r).LeadByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "No such lead.")
		return store.Lead{}, false
	}
	if err != nil {
		a.serverError(w, r, err)
		return store.Lead{}, false
	}
	staff := staffFrom(r.Context())
	if !staff.Manager() && l.AgentID != staff.ID {
		a.notYours(w)
		return store.Lead{}, false
	}
	return l, true
}

// leadPage is everything a lead's page shows.
func (a *App) leadPage(r *http.Request, l store.Lead, notice string) (DeskLeadPage, error) {
	db := a.db(r)
	out := DeskLeadPage{Lead: a.deskLeadOf(l), Timeline: []DeskTimelineEntry{}, Viewings: []DeskViewing{}, Reservations: []DeskReservation{}, Touches: []DeskTouch{}, Notice: notice}
	timeline, err := db.Timeline(r.Context(), l.ID)
	if err != nil {
		return out, err
	}
	for _, e := range timeline {
		out.Timeline = append(out.Timeline, DeskTimelineEntry{At: localTime(e.At), Kind: e.Kind, Body: e.Body, Home: e.UnitID, Staff: e.StaffName})
	}
	viewingList, err := db.LeadViewings(r.Context(), l.ID)
	if err != nil {
		return out, err
	}
	for _, v := range viewingList {
		out.Viewings = append(out.Viewings, a.deskViewingOf(v))
	}
	reservations, err := db.LeadReservations(r.Context(), l.ID)
	if err != nil {
		return out, err
	}
	for _, res := range reservations {
		out.Reservations = append(out.Reservations, a.deskReservationOf(res))
	}
	touches, err := db.LeadTouches(r.Context(), l.ID)
	if err != nil {
		return out, err
	}
	for _, t := range touches {
		out.Touches = append(out.Touches, DeskTouch{At: localTime(t.At), Action: t.Action, Home: t.UnitID, Last: arrivalOf(t.Last), First: arrivalOf(t.First)})
	}
	return out, nil
}

func (a *App) deskLead(w http.ResponseWriter, r *http.Request) {
	l, ok := a.leadFromPath(w, r)
	if !ok {
		return
	}
	page, err := a.leadPage(r, l, "")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// leadAction runs a change to a lead and answers with its page as it now
// stands. A refusal from the store is shown beside field.
func (a *App) leadAction(w http.ResponseWriter, r *http.Request, field string, run func(db *store.DB, l store.Lead, staff store.Staff) (string, error)) {
	l, ok := a.leadFromPath(w, r)
	if !ok {
		return
	}
	db := a.db(r)
	done, err := run(db, l, staffFrom(r.Context()))
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "No such lead.")
		return
	}
	if err != nil {
		a.answerError(w, r, err, field)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead-changed", ID: strconv.FormatInt(l.ID, 10)})
	if l, err = db.LeadByID(r.Context(), l.ID); err != nil {
		a.serverError(w, r, err)
		return
	}
	page, err := a.leadPage(r, l, done)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *App) deskLeadNote(w http.ResponseWriter, r *http.Request) {
	var in DeskNoteInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.leadAction(w, r, "body", func(db *store.DB, l store.Lead, staff store.Staff) (string, error) {
		body := strings.TrimSpace(in.Body)
		switch {
		case body == "":
			return "", refusal{"body", "The note is empty."}
		case len([]rune(body)) > 2000:
			return "", refusal{"body", "Please keep this under 2,000 characters."}
		}
		return "Note saved.", db.AddNote(r.Context(), l.ID, body, staff, a.now())
	})
}

// checkStage checks a stage a lead is being moved to, and the reason a lost
// lead needs.
func checkStage(stage, reason string) error {
	switch {
	case store.StageTitle(stage) == "":
		return refusal{"stage", sentence(fmt.Sprintf("unknown stage %q", stage))}
	case stage == "lost" && strings.TrimSpace(reason) == "":
		return refusal{"reason", "Please say why the lead was lost."}
	case len([]rune(strings.TrimSpace(reason))) > 200:
		return refusal{"reason", "Please keep this under 200 characters."}
	}
	return nil
}

func (a *App) deskLeadStage(w http.ResponseWriter, r *http.Request) {
	var in DeskStageInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.leadAction(w, r, "stage", func(db *store.DB, l store.Lead, staff store.Staff) (string, error) {
		if err := checkStage(in.Stage, in.Reason); err != nil {
			return "", err
		}
		if err := db.MoveLead(r.Context(), l.ID, in.Stage, in.Reason, 0, 0, staff, a.now()); err != nil {
			return "", err
		}
		return "Moved to " + store.StageTitle(in.Stage) + ".", nil
	})
}

func (a *App) deskLeadFollowUp(w http.ResponseWriter, r *http.Request) {
	var in DeskFollowUpInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.leadAction(w, r, "day", func(db *store.DB, l store.Lead, staff store.Staff) (string, error) {
		day := strings.TrimSpace(in.Day)
		if day != "" {
			if _, err := time.ParseInLocation(time.DateOnly, day, store.Nairobi()); err != nil {
				return "", refusal{"day", "That date doesn’t look right."}
			}
		}
		if err := db.SetFollowUp(r.Context(), l.ID, day, staff, a.now()); err != nil {
			return "", err
		}
		if day == "" {
			return "Follow-up cleared.", nil
		}
		return "Follow-up set for " + dayLabel(day) + ".", nil
	})
}

func (a *App) deskLeadAssign(w http.ResponseWriter, r *http.Request) {
	var in DeskAssignInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.leadAction(w, r, "agent", func(db *store.DB, l store.Lead, staff store.Staff) (string, error) {
		member, err := teamMember(r, db, in.Agent)
		if err != nil {
			return "", err
		}
		if member == nil {
			return "", refusal{"agent", "Please choose an agent."}
		}
		if err := db.AssignLead(r.Context(), l.ID, member.ID, staff, a.now()); err != nil {
			return "", err
		}
		return "Assigned to " + member.Name + ".", nil
	})
}

// deskLeadMove is a card dropped on the board.
func (a *App) deskLeadMove(w http.ResponseWriter, r *http.Request) {
	var in DeskMoveInput
	if !a.readJSON(w, r, &in) {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		a.notFound(w, "No such lead.")
		return
	}
	if err := checkStage(in.Stage, in.Reason); err != nil {
		a.answerError(w, r, err, "")
		return
	}
	err = a.db(r).MoveLead(r.Context(), id, in.Stage, in.Reason, in.After, in.Before, staffFrom(r.Context()), a.now())
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "No such lead.")
		return
	}
	if err != nil {
		a.answerError(w, r, err, "stage")
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead-changed", ID: strconv.FormatInt(id, 10)})
	w.WriteHeader(http.StatusNoContent)
}

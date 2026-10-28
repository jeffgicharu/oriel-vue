package api

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/store"
	"github.com/jeffgicharu/oriel-vue/api/internal/viewings"
)

// DeskAgenda is the sales suite's agenda over a run of days.
type DeskAgenda struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Days is each day from From to To.
	Days []DeskAgendaDay `json:"days"`
	// Viewings is every viewing in those days, earliest first; an agent sees
	// only their own leads'.
	Viewings []DeskViewing `json:"viewings"`
	Hours    DeskHours     `json:"hours"`
	// Blocks is every closed day from today on.
	Blocks []DeskBlock `json:"blocks"`
}

// DeskAgendaDay is a day on the agenda.
type DeskAgendaDay struct {
	Date string `json:"date"`
	// Open is a day the suite opens and nobody has closed.
	Open bool `json:"open"`
	// Blocked is why the day was closed, when it was.
	Blocked string `json:"blocked"`
	// Viewings is how many visits are booked that day, leaving out cancelled
	// ones.
	Viewings int `json:"viewings"`
}

// DeskHours is when the sales suite opens for viewings.
type DeskHours struct {
	// Days are the days it opens, 0 for Sunday to 6 for Saturday.
	Days  []int    `json:"days"`
	Times []string `json:"times"`
	// Capacity is how many parties a slot takes at once.
	Capacity int `json:"capacity"`
	// Minutes is how long a visit takes.
	Minutes int `json:"minutes"`
}

// DeskBlock is a day the suite is closed for viewings.
type DeskBlock struct {
	Day    string `json:"day"`
	Reason string `json:"reason"`
}

// DeskOutcomeInput says whether a visitor came.
type DeskOutcomeInput struct {
	// Outcome is "attended" or "no_show".
	Outcome string `json:"outcome"`
	Notes   string `json:"notes"`
}

// DeskHoursInput changes when the suite opens.
type DeskHoursInput struct {
	// Days are the days it opens, 0 for Sunday to 6 for Saturday.
	Days []int `json:"days"`
	// Times are the 24-hour start times of the visits, such as "10:00".
	Times    []string `json:"times"`
	Capacity int      `json:"capacity"`
}

// DeskBlockInput closes a day for viewings.
type DeskBlockInput struct {
	Day    string `json:"day"`
	Reason string `json:"reason"`
}

// DeskViewingResult is a viewing after a change, with what the change did.
type DeskViewingResult struct {
	Viewing DeskViewing `json:"viewing"`
	Notice  string      `json:"notice"`
}

// DeskHoursResult is the opening hours after a change.
type DeskHoursResult struct {
	Hours  DeskHours `json:"hours"`
	Notice string    `json:"notice"`
}

// DeskBlocksResult is the closed days after a change.
type DeskBlocksResult struct {
	Blocks []DeskBlock `json:"blocks"`
	Notice string      `json:"notice"`
}

// agendaSpan is the most days the agenda shows at once.
const agendaSpan = 62

func hoursOf(c viewings.Config) DeskHours {
	out := DeskHours{Days: []int{}, Times: append([]string{}, c.Times...), Capacity: c.Capacity, Minutes: int(viewings.Length / time.Minute)}
	for _, d := range c.Days {
		out.Days = append(out.Days, int(d))
	}
	sort.Ints(out.Days)
	return out
}

// blocksFrom is every closed day from a day on.
func (a *App) blocksFrom(r *http.Request, from time.Time) ([]DeskBlock, error) {
	blocks, err := a.db(r).Blocks(r.Context(), from.Format(time.DateOnly))
	out := []DeskBlock{}
	for _, b := range blocks {
		out = append(out, DeskBlock{Day: b.Day, Reason: b.Reason})
	}
	return out, err
}

func (a *App) deskViewings(w http.ResponseWriter, r *http.Request) {
	staff := staffFrom(r.Context())
	db := a.db(r)
	nairobi := store.Nairobi()
	day := today(a.now())
	// The week from Monday, unless asked otherwise.
	from := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	q := r.URL.Query()
	errs := fieldErrors{}
	if v := q.Get("from"); v != "" {
		t, err := time.ParseInLocation(time.DateOnly, v, nairobi)
		if err != nil {
			errs["from"] = "That date doesn’t look right."
		}
		from = t
	}
	to := from.AddDate(0, 0, 6)
	if v := q.Get("to"); v != "" {
		t, err := time.ParseInLocation(time.DateOnly, v, nairobi)
		if err != nil {
			errs["to"] = "That date doesn’t look right."
		}
		to = t
	}
	if len(errs) == 0 {
		switch {
		case to.Before(from):
			errs["to"] = "The last day comes before the first."
		case to.After(from.AddDate(0, 0, agendaSpan-1)):
			errs["to"] = "Ask for up to " + strconv.Itoa(agendaSpan) + " days at a time."
		}
	}
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Those dates need another look.", errs)
		return
	}

	cfg := db.ViewingConfig(r.Context())
	earliest := from
	if day.Before(earliest) {
		earliest = day
	}
	blocks, err := db.Blocks(r.Context(), earliest.Format(time.DateOnly))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	list, err := db.ViewingsBetween(r.Context(), from, to.AddDate(0, 0, 1), agentScope(staff))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	blocked := map[string]string{}
	out := DeskAgenda{From: from.Format(time.DateOnly), To: to.Format(time.DateOnly), Days: []DeskAgendaDay{}, Viewings: []DeskViewing{}, Hours: hoursOf(cfg), Blocks: []DeskBlock{}}
	for _, b := range blocks {
		blocked[b.Day] = b.Reason
		if b.Day >= day.Format(time.DateOnly) {
			out.Blocks = append(out.Blocks, DeskBlock{Day: b.Day, Reason: b.Reason})
		}
	}
	perDay := map[string]int{}
	for _, v := range list {
		if v.Status != "cancelled" {
			perDay[v.StartsAt.In(nairobi).Format(time.DateOnly)]++
		}
		out.Viewings = append(out.Viewings, a.deskViewingOf(v))
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(time.DateOnly)
		reason, isBlocked := blocked[key]
		out.Days = append(out.Days, DeskAgendaDay{Date: key, Open: cfg.Opens(d) && !isBlocked, Blocked: reason, Viewings: perDay[key]})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) deskViewingOutcome(w http.ResponseWriter, r *http.Request) {
	var in DeskOutcomeInput
	if !a.readJSON(w, r, &in) {
		return
	}
	ref := r.PathValue("ref")
	db := a.db(r)
	notes := strings.TrimSpace(in.Notes)
	var err error
	switch {
	case in.Outcome != "attended" && in.Outcome != "no_show":
		err = refusal{"outcome", "Unknown outcome."}
	case len([]rune(notes)) > 300:
		err = refusal{"notes", "Please keep this under 300 characters."}
	default:
		err = db.SetViewingOutcome(r.Context(), ref, in.Outcome, notes, staffFrom(r.Context()), a.now())
	}
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "No such viewing.")
		return
	}
	if err != nil {
		a.answerError(w, r, err, "outcome")
		return
	}
	v, err := db.ViewingByRef(r.Context(), ref)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "lead-changed", ID: strconv.FormatInt(v.LeadID, 10)})
	notice := "Marked as came."
	if in.Outcome == "no_show" {
		notice = "Marked as did not come."
	}
	writeJSON(w, http.StatusOK, DeskViewingResult{Viewing: a.deskViewingOf(v), Notice: notice})
}

func (a *App) deskViewingHours(w http.ResponseWriter, r *http.Request) {
	var in DeskHoursInput
	if !a.readJSON(w, r, &in) {
		return
	}
	var cfg viewings.Config
	seen := map[int]bool{}
	for _, n := range in.Days {
		if n >= 0 && n <= 6 && !seen[n] {
			seen[n] = true
			cfg.Days = append(cfg.Days, time.Weekday(n))
		}
	}
	errs := fieldErrors{}
	for _, t := range in.Times {
		t = strings.TrimSpace(t)
		h, m, err := viewings.ParseClock(t)
		if err != nil {
			errs["times"] = "“" + t + "” is not a time; write times like 10:00, 14:45."
			break
		}
		cfg.Times = append(cfg.Times, twoDigits(h)+":"+twoDigits(m))
	}
	cfg.Capacity = in.Capacity
	if len(cfg.Days) == 0 {
		errs["days"] = "Choose at least one day."
	}
	if len(in.Times) == 0 {
		errs["times"] = "Choose at least one time."
	}
	if cfg.Capacity < 1 || cfg.Capacity > 6 {
		errs["capacity"] = "A slot takes between 1 and 6 parties."
	}
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Some answers need another look.", errs)
		return
	}
	if err := a.db(r).SaveViewingConfig(r.Context(), cfg, staffFrom(r.Context()), a.now()); err != nil {
		a.answerError(w, r, err, "times")
		return
	}
	writeJSON(w, http.StatusOK, DeskHoursResult{Hours: hoursOf(a.db(r).ViewingConfig(r.Context())), Notice: "Opening hours saved. The booking page shows them at once."})
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func (a *App) deskBlockDay(w http.ResponseWriter, r *http.Request) {
	var in DeskBlockInput
	if !a.readJSON(w, r, &in) {
		return
	}
	day, reason := strings.TrimSpace(in.Day), strings.TrimSpace(in.Reason)
	errs := fieldErrors{}
	if _, err := time.ParseInLocation(time.DateOnly, day, store.Nairobi()); err != nil {
		errs["day"] = "That date doesn’t look right."
	}
	if reason == "" {
		errs["reason"] = "Please give a reason."
	}
	tooLong(errs, "reason", reason, 120)
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Some answers need another look.", errs)
		return
	}
	if err := a.db(r).BlockDay(r.Context(), day, reason, staffFrom(r.Context()), a.now()); err != nil {
		a.answerError(w, r, err, "day")
		return
	}
	a.answerBlocks(w, r, dayLabel(day)+" is closed for viewings.")
}

func (a *App) deskUnblockDay(w http.ResponseWriter, r *http.Request) {
	day := r.PathValue("day")
	if _, err := time.ParseInLocation(time.DateOnly, day, store.Nairobi()); err != nil {
		a.notFound(w, "There is no closed day "+day+".")
		return
	}
	if err := a.db(r).UnblockDay(r.Context(), day, staffFrom(r.Context()), a.now()); err != nil {
		a.answerError(w, r, err, "")
		return
	}
	a.answerBlocks(w, r, dayLabel(day)+" is open for viewings again.")
}

// answerBlocks answers a change to the closed days with every one from today
// on.
func (a *App) answerBlocks(w http.ResponseWriter, r *http.Request, notice string) {
	blocks, err := a.blocksFrom(r, today(a.now()))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, DeskBlocksResult{Blocks: blocks, Notice: notice})
}

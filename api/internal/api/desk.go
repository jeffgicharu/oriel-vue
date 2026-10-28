package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jeffgicharu/oriel-vue/api/internal/building"
	"github.com/jeffgicharu/oriel-vue/api/internal/plans"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
	"github.com/jeffgicharu/oriel-vue/api/internal/viewings"
)

// The sales desk lives under /api/desk. A member of the team signs in on a
// copy of the business, by password or by a one-click link, and works on that
// copy alone; the visitor-facing site reads the same copy, so a change on the
// desk shows on the website at once.

type staffKey struct{}

func staffFrom(ctx context.Context) store.Staff {
	s, _ := ctx.Value(staffKey{}).(store.Staff)
	return s
}

// desk lets only a signed-in member of the team through.
func (a *App) desk(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := a.db(r)
		c, err := r.Cookie(deskCookie)
		var staff store.Staff
		if err == nil {
			staff, err = db.SessionStaff(r.Context(), c.Value, a.now())
		}
		if err != nil {
			if !errors.Is(err, http.ErrNoCookie) && !errors.Is(err, sql.ErrNoRows) {
				a.log.Error("reading a desk session", "err", err)
			}
			a.problem(w, http.StatusUnauthorized, "signed-out", "Please sign in again.", nil)
			return
		}
		// Holds that have run out are released before anything is read, as
		// every page of the desk did.
		if err := db.LapseHolds(r.Context(), a.now()); err != nil {
			a.log.Error("lapsing holds", "err", err)
		}
		next(w, r.WithContext(context.WithValue(r.Context(), staffKey{}, staff)))
	}
}

// managers lets only the sales manager through.
func (a *App) managers(next http.HandlerFunc) http.HandlerFunc {
	return a.desk(func(w http.ResponseWriter, r *http.Request) {
		if !staffFrom(r.Context()).Manager() {
			a.managersOnly(w)
			return
		}
		next(w, r)
	})
}

func (a *App) managersOnly(w http.ResponseWriter) {
	a.problem(w, http.StatusForbidden, "managers-only", "Only the sales manager can do that.", nil)
}

func (a *App) notYours(w http.ResponseWriter) {
	a.problem(w, http.StatusForbidden, "not-yours", "This lead belongs to another agent.", nil)
}

// agentScope is the agent whose work a member sees: themselves, or nobody in
// particular for the manager, who sees everyone's.
func agentScope(s store.Staff) int64 {
	if s.Manager() {
		return 0
	}
	return s.ID
}

// refusal is a change the desk turns down before it reaches the store, with
// the field it is about.
type refusal struct{ field, text string }

func (e refusal) Error() string { return e.text }

// answerError answers a change that did not go through: a refusal beside its
// field, a rule of the business the store would not break, or a fault of our
// own. The store writes its refusals for people to read, so they are shown as
// they are, beside field when one is given. A missing record is for the
// caller to answer before this, since only it knows what was missing.
func (a *App) answerError(w http.ResponseWriter, r *http.Request, err error, field string) {
	var rf refusal
	switch {
	case errors.As(err, &rf):
		a.problem(w, http.StatusUnprocessableEntity, "validation", rf.text, fieldErrors{rf.field: rf.text})
	case errors.Is(err, store.ErrManagersOnly):
		a.managersOnly(w)
	case errors.Is(err, store.ErrNotYours):
		a.notYours(w)
	case isInternal(err):
		a.serverError(w, r, err)
	default:
		text := sentence(err.Error())
		var errs fieldErrors
		if field != "" {
			errs = fieldErrors{field: text}
		}
		a.problem(w, http.StatusUnprocessableEntity, "validation", text, errs)
	}
}

// sentence turns a refusal from the store into a sentence: a capital first
// letter and a full stop.
func sentence(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	if last := r[len(r)-1]; last != '.' && last != '?' && last != '!' {
		r = append(r, '.')
	}
	return string(r)
}

// tooLong checks a field's length, noting it when it is over.
func tooLong(errs fieldErrors, field, value string, max int) {
	if len([]rune(value)) > max {
		errs[field] = "Please keep this under " + thousands(int64(max)) + " characters."
	}
}

// kes writes an amount as the desk shows it: "KES 18,480,000".
func kes(n int64) string { return "KES " + thousands(n) }

func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// dayLabel writes a calendar date as the desk says it: "Tue 21 Oct".
func dayLabel(day string) string {
	t, err := time.ParseInLocation(time.DateOnly, day, store.Nairobi())
	if err != nil {
		return day
	}
	return t.Format("Mon 2 Jan")
}

// today is midnight in Nairobi on the day of now.
func today(now time.Time) time.Time {
	local := now.In(store.Nairobi())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, store.Nairobi())
}

// ---------- Shapes the desk's pages share ----------

// DeskStaff is a member of the sales team.
type DeskStaff struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	// Role is "manager" or "agent".
	Role   string `json:"role"`
	Title  string `json:"title"`
	Colour string `json:"colour"`
	// Initials are shown on a lead card: "GW".
	Initials string `json:"initials"`
}

func deskStaffOf(s store.Staff) DeskStaff {
	return DeskStaff{ID: s.ID, Username: s.Username, Name: s.Name, Role: s.Role, Title: s.Title, Colour: s.Colour, Initials: s.Initials()}
}

// DeskHome is a home as the desk sees it: always with its price, whatever its
// status.
type DeskHome struct {
	ID         string `json:"id"`
	Level      int    `json:"level"`
	Number     int    `json:"number"`
	Type       string `json:"type"`
	TypeName   string `json:"typeName"`
	Title      string `json:"title"`
	Facings    string `json:"facings"`
	Internal   int    `json:"internal"`
	RiverView  bool   `json:"riverView"`
	Status     string `json:"status"`
	StatusName string `json:"statusName"`
	Price      int64  `json:"price"`
	ListPrice  int64  `json:"listPrice"`
	// Overridden is true when the home has its own price rather than the
	// price list's.
	Overridden bool `json:"overridden"`
	// Wide is a penthouse, which takes half its floor on the stacking plan.
	Wide bool `json:"wide"`
}

func deskHomeOf(h store.Home) DeskHome {
	return DeskHome{
		ID: h.ID, Level: h.Level, Number: h.Position.Number, Type: h.Type.Code, TypeName: h.Type.Name,
		Title: homeTitle(h), Facings: h.Facings(), Internal: h.Type.Internal, RiverView: h.Position.RiverView,
		Status: h.Status, StatusName: store.StatusName(h.Status),
		Price: h.Price, ListPrice: h.ListPrice, Overridden: h.Overridden, Wide: h.Level == building.PenthouseLevel,
	}
}

// DeskReservation is a reservation as the desk sees it.
type DeskReservation struct {
	Ref        string `json:"ref"`
	Status     string `json:"status"`
	StatusName string `json:"statusName"`
	// Group is the tab of the reservations page it sits in.
	Group    string `json:"group"`
	LeadID   int64  `json:"leadId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Home     string `json:"home"`
	Plan     string `json:"plan"`
	PlanName string `json:"planName"`
	Price    int64  `json:"price"`
	Note     string `json:"note"`
	// Agent looks after the buyer.
	Agent           string `json:"agent"`
	Requested       string `json:"requested"`
	HoldEnds        string `json:"holdEnds"`
	HoldSecondsLeft int64  `json:"holdSecondsLeft"`
	Decided         string `json:"decided"`
	DecidedBy       string `json:"decidedBy"`
	DeclineReason   string `json:"declineReason"`
	OfferNumber     string `json:"offerNumber"`
	// Offer is the letter of offer, once one has been issued.
	Offer         string `json:"offer"`
	Deposit       string `json:"deposit"`
	DepositAmount int64  `json:"depositAmount"`
	// SuggestedDeposit is a tenth of the price, which the deposit form starts
	// from.
	SuggestedDeposit int64 `json:"suggestedDeposit"`
}

// reservationGroup is the tab a reservation with a status sits in.
func reservationGroup(status string) string {
	for group, statuses := range store.ReservationGroups {
		for _, s := range statuses {
			if s == status {
				return group
			}
		}
	}
	return "closed"
}

func planName(code string) string {
	if p, ok := plans.ByCode(code); ok {
		return p.Name
	}
	return code
}

func (a *App) deskReservationOf(r store.Reservation) DeskReservation {
	out := DeskReservation{
		Ref: r.Ref, Status: r.Status, StatusName: store.ReservationStatusName(r.Status), Group: reservationGroup(r.Status),
		LeadID: r.LeadID, Name: r.Name, Email: r.Email, Phone: r.Phone, Home: r.UnitID,
		Plan: r.Plan, PlanName: planName(r.Plan), Price: r.Price, Note: r.BuyerNote, Agent: r.AgentName,
		Requested: localTime(r.RequestedAt), HoldEnds: localTime(r.HoldExpiresAt),
		Decided: localTime(r.DecidedAt), DecidedBy: r.DecidedBy, DeclineReason: r.DeclineReason, OfferNumber: r.OfferNumber,
		Deposit: localTime(r.DepositAt), DepositAmount: r.DepositAmount, SuggestedDeposit: r.Price / 10,
	}
	if left := r.HoldExpiresAt.Sub(a.now()); r.Status == "requested" && left > 0 {
		out.HoldSecondsLeft = int64(left.Seconds())
	}
	if r.OfferNumber != "" {
		out.Offer = "/api/reservations/" + r.Ref + "/offer.pdf"
	}
	return out
}

// DeskViewing is a viewing on the desk's agenda.
type DeskViewing struct {
	Ref    string `json:"ref"`
	LeadID int64  `json:"leadId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	// Home is the home the visitor asked to see, if any.
	Home    string `json:"home"`
	Start   string `json:"start"`
	Minutes int    `json:"minutes"`
	Party   int    `json:"party"`
	Status  string `json:"status"`
	// Notes are the visitor's own; Outcome is what the agent wrote after.
	Notes   string `json:"notes"`
	Outcome string `json:"outcome"`
	Agent   string `json:"agent"`
	Booked  string `json:"booked"`
	// Due is a booked viewing whose time has come, waiting for the agent to
	// say whether the visitor came.
	Due bool `json:"due"`
}

func (a *App) deskViewingOf(v store.Viewing) DeskViewing {
	return DeskViewing{
		Ref: v.Ref, LeadID: v.LeadID, Name: v.Name, Email: v.Email, Phone: v.Phone, Home: v.UnitID,
		Start: localTime(v.StartsAt), Minutes: int(viewings.Length / time.Minute), Party: v.PartySize,
		Status: v.Status, Notes: v.Notes, Outcome: v.Outcome, Agent: v.AgentName, Booked: localTime(v.CreatedAt),
		Due: v.Status == "booked" && v.StartsAt.Before(a.now()),
	}
}

// DeskLead is a person the sales team is talking to.
type DeskLead struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	// Source is where they came from: the website's channel for their first
	// arrival, or what the team wrote down, such as Referral.
	Source      string `json:"source"`
	Stage       string `json:"stage"`
	StageTitle  string `json:"stageTitle"`
	AgentID     int64  `json:"agentId"`
	Agent       string `json:"agent"`
	AgentColour string `json:"agentColour"`
	Interest    string `json:"interest"`
	Budget      int64  `json:"budget"`
	// Buying is "home" to live in, "investment" to let, or empty.
	Buying     string `json:"buying"`
	FollowUpOn string `json:"followUpOn"`
	// Overdue is an open lead whose follow-up day has come.
	Overdue      bool   `json:"overdue"`
	LostReason   string `json:"lostReason"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
	StageChanged string `json:"stageChanged"`
}

func (a *App) deskLeadOf(l store.Lead) DeskLead {
	return DeskLead{
		ID: l.ID, Name: l.Name, Email: l.Email, Phone: l.Phone, Source: l.Source,
		Stage: l.Stage, StageTitle: store.StageTitle(l.Stage),
		AgentID: l.AgentID, Agent: l.AgentName, AgentColour: l.AgentColour,
		Interest: l.Interest, Budget: l.Budget, Buying: l.BuyingFor, FollowUpOn: l.FollowUpOn, Overdue: l.Overdue(a.now()),
		LostReason: l.LostReason, Created: localTime(l.CreatedAt), Updated: localTime(l.UpdatedAt), StageChanged: localTime(l.StageChangedAt),
	}
}

func (a *App) deskLeadsOf(list []store.Lead) []DeskLead {
	out := make([]DeskLead, 0, len(list))
	for _, l := range list {
		out = append(out, a.deskLeadOf(l))
	}
	return out
}

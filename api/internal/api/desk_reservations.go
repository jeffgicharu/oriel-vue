package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// DeskReservations is one tab of the reservations page.
type DeskReservations struct {
	// Group is the tab: waiting, confirmed (awaiting the deposit), completed
	// (sold) or closed.
	Group string `json:"group"`
	// Counts is how many reservations each tab holds.
	Counts map[string]int `json:"counts"`
	// Reservations is the tab's list: the waiting by when their hold lapses,
	// the rest newest first. An agent sees only their own leads'.
	Reservations []DeskReservation `json:"reservations"`
}

// DeskReservationResult is a reservation after a decision, with its home as
// it now stands and what the decision did.
type DeskReservationResult struct {
	Reservation DeskReservation `json:"reservation"`
	Home        DeskHome        `json:"home"`
	Notice      string          `json:"notice"`
}

// DeskDeclineInput turns a request down.
type DeskDeclineInput struct {
	// Reason is shown to the buyer.
	Reason string `json:"reason"`
}

// DeskDepositInput records the deposit, which sells the home.
type DeskDepositInput struct {
	// Amount is in KES.
	Amount int64 `json:"amount"`
}

func (a *App) deskReservations(w http.ResponseWriter, r *http.Request) {
	staff := staffFrom(r.Context())
	db := a.db(r)
	group := r.URL.Query().Get("group")
	if _, ok := store.ReservationGroups[group]; !ok {
		group = "waiting"
	}
	list, err := db.ReservationsIn(r.Context(), group, agentScope(staff))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	counts, err := db.ReservationGroupCounts(r.Context(), agentScope(staff))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	out := DeskReservations{Group: group, Counts: counts, Reservations: []DeskReservation{}}
	for _, res := range list {
		out.Reservations = append(out.Reservations, a.deskReservationOf(res))
	}
	writeJSON(w, http.StatusOK, out)
}

// reservationAction runs the manager's decision on a reservation and answers
// with the reservation and its home as they now stand.
func (a *App) reservationAction(w http.ResponseWriter, r *http.Request, run func(db *store.DB, res store.Reservation, staff store.Staff) (string, error)) {
	ref := r.PathValue("ref")
	db := a.db(r)
	before, err := db.ReservationByRef(r.Context(), ref)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "No such reservation.")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	done, err := run(db, before, staffFrom(r.Context()))
	if err != nil {
		a.answerError(w, r, err, "")
		return
	}
	after, err := db.ReservationByRef(r.Context(), ref)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	h, found := c.Home(after.UnitID)
	if !found {
		a.notFound(w, "No such home.")
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID,
		event{Type: "unit", ID: h.ID, Status: h.Status},
		event{Type: "reservation", ID: ref, Status: after.Status},
	)
	writeJSON(w, http.StatusOK, DeskReservationResult{Reservation: a.deskReservationOf(after), Home: deskHomeOf(h), Notice: done})
}

func (a *App) deskConfirm(w http.ResponseWriter, r *http.Request) {
	a.reservationAction(w, r, func(db *store.DB, res store.Reservation, staff store.Staff) (string, error) {
		if err := db.ConfirmReservation(r.Context(), res.Ref, staff.ID, a.now()); err != nil {
			return "", err
		}
		after, err := db.ReservationByRef(r.Context(), res.Ref)
		if err != nil {
			return "", err
		}
		return "Reservation " + res.Ref + " confirmed. Letter of offer " + after.OfferNumber + " issued, and home " + res.UnitID + " is reserved.", nil
	})
}

func (a *App) deskDecline(w http.ResponseWriter, r *http.Request) {
	var in DeskDeclineInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.reservationAction(w, r, func(db *store.DB, res store.Reservation, staff store.Staff) (string, error) {
		reason := strings.TrimSpace(in.Reason)
		switch n := len([]rune(reason)); {
		case n < 3:
			return "", refusal{"reason", "Please give the buyer a reason."}
		case n > 200:
			return "", refusal{"reason", "Please keep this under 200 characters."}
		}
		if err := db.DeclineReservation(r.Context(), res.Ref, reason, staff.ID, a.now()); err != nil {
			return "", err
		}
		return "Reservation " + res.Ref + " declined. Home " + res.UnitID + " is for sale again.", nil
	})
}

func (a *App) deskDeposit(w http.ResponseWriter, r *http.Request) {
	var in DeskDepositInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.reservationAction(w, r, func(db *store.DB, res store.Reservation, staff store.Staff) (string, error) {
		switch {
		case in.Amount <= 0:
			return "", refusal{"amount", "Please give the deposit in shillings, such as 1,848,000."}
		case in.Amount > res.Price:
			return "", refusal{"amount", "The deposit is more than the price of the home."}
		}
		if err := db.RecordDeposit(r.Context(), res.Ref, in.Amount, staff.ID, a.now()); err != nil {
			return "", err
		}
		return "Deposit of " + kes(in.Amount) + " recorded. Home " + res.UnitID + " is sold.", nil
	})
}

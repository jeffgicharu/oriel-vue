package api

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/jeffgicharu/oriel-vue/api/internal/building"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// DeskStack is the stacking plan: every home, floor by floor, the top floor
// first.
type DeskStack struct {
	Floors []DeskFloor `json:"floors"`
	// Counts is how many homes have each status.
	Counts map[string]int `json:"counts"`
	Total  int            `json:"total"`
}

// DeskFloor is one floor of the stacking plan, its homes in number order.
type DeskFloor struct {
	Level int `json:"level"`
	// Name is the level as the plan labels it: "L07".
	Name  string     `json:"name"`
	Homes []DeskHome `json:"homes"`
}

// DeskHomeDetail is a home's drawer on the stacking plan.
type DeskHomeDetail struct {
	Home DeskHome `json:"home"`
	// History is the home's changes, newest first.
	History []DeskHomeEvent `json:"history"`
	// Reservation is the one that holds, reserves or sold the home, if any.
	Reservation *DeskReservation `json:"reservation"`
	// CanChange is true for the sales manager, the only one who may change a
	// home's status or price.
	CanChange bool `json:"canChange"`
	// Notice says what a change just did; it is empty on a read.
	Notice string `json:"notice"`
}

// DeskHomeEvent is a change to a home. A status change goes from one status
// to another; a price change from one price, or "list price", to another.
type DeskHomeEvent struct {
	At     string `json:"at"`
	Kind   string `json:"kind"`
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
	Staff  string `json:"staff"`
}

// DeskStatusInput withholds a home from sale or releases it.
type DeskStatusInput struct {
	// Status is "withheld" or "available".
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// DeskPriceInput gives a home its own price.
type DeskPriceInput struct {
	// Price is in KES, to the nearest 10,000; 0 puts the home back on the
	// list price.
	Price  int64  `json:"price"`
	Reason string `json:"reason"`
}

func (a *App) deskStack(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	byLevel := map[int][]DeskHome{}
	for _, h := range c.Homes {
		byLevel[h.Level] = append(byLevel[h.Level], deskHomeOf(h))
	}
	out := DeskStack{Floors: []DeskFloor{}, Counts: c.Counts(), Total: len(c.Homes)}
	for _, level := range building.HomeLevels() {
		homes := append([]DeskHome{}, byLevel[level]...)
		sort.Slice(homes, func(i, j int) bool { return homes[i].ID < homes[j].ID })
		out.Floors = append(out.Floors, DeskFloor{Level: level, Name: building.LevelName(level), Homes: homes})
	}
	writeJSON(w, http.StatusOK, out)
}

// homeDetail is a home's drawer, with what a change just did.
func (a *App) homeDetail(r *http.Request, h store.Home, notice string) (DeskHomeDetail, error) {
	db := a.db(r)
	out := DeskHomeDetail{Home: deskHomeOf(h), History: []DeskHomeEvent{}, CanChange: staffFrom(r.Context()).Manager(), Notice: notice}
	history, err := db.UnitHistory(r.Context(), h.ID)
	if err != nil {
		return out, err
	}
	for _, e := range history {
		out.History = append(out.History, DeskHomeEvent{At: localTime(e.At), Kind: e.Kind, From: e.From, To: e.To, Reason: e.Reason, Staff: e.StaffName})
	}
	res, err := db.UnitReservation(r.Context(), h.ID)
	if err != nil {
		return out, err
	}
	if res != nil {
		d := a.deskReservationOf(*res)
		out.Reservation = &d
	}
	return out, nil
}

func (a *App) deskHome(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	h, found := c.Home(r.PathValue("id"))
	if !found {
		a.notFound(w, "No such home.")
		return
	}
	out, err := a.homeDetail(r, h, "")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// homeChange runs a change to a home and answers with its drawer as it now
// stands. A refusal from the store is shown beside field.
func (a *App) homeChange(w http.ResponseWriter, r *http.Request, field string, change func(db *store.DB, staff store.Staff, id string) (string, error)) {
	id := r.PathValue("id")
	done, err := change(a.db(r), staffFrom(r.Context()), id)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "No such home.")
		return
	}
	if err != nil {
		a.answerError(w, r, err, field)
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	h, found := c.Home(id)
	if !found {
		a.notFound(w, "No such home.")
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "unit", ID: h.ID, Status: h.Status})
	out, err := a.homeDetail(r, h, done)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) deskHomeStatus(w http.ResponseWriter, r *http.Request) {
	var in DeskStatusInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.homeChange(w, r, "status", func(db *store.DB, staff store.Staff, id string) (string, error) {
		reason := strings.TrimSpace(in.Reason)
		switch {
		case in.Status != store.Available && in.Status != store.Withheld:
			return "", refusal{"status", "A home can be made available or withheld here; holds, reservations and sales go through a reservation."}
		case reason == "":
			return "", refusal{"reason", "Please give a reason."}
		case len([]rune(reason)) > 200:
			return "", refusal{"reason", "Please keep this under 200 characters."}
		}
		if err := db.SetUnitStatus(r.Context(), id, in.Status, reason, staff, a.now()); err != nil {
			return "", err
		}
		if in.Status == store.Withheld {
			return "Home " + id + " is withheld from sale.", nil
		}
		return "Home " + id + " is for sale again.", nil
	})
}

func (a *App) deskHomePrice(w http.ResponseWriter, r *http.Request) {
	var in DeskPriceInput
	if !a.readJSON(w, r, &in) {
		return
	}
	a.homeChange(w, r, "price", func(db *store.DB, staff store.Staff, id string) (string, error) {
		reason := strings.TrimSpace(in.Reason)
		switch {
		case in.Price < 0:
			return "", refusal{"price", "Please give the price in shillings, such as 18,480,000."}
		case reason == "":
			return "", refusal{"reason", "Please give a reason."}
		case len([]rune(reason)) > 200:
			return "", refusal{"reason", "Please keep this under 200 characters."}
		}
		if err := db.SetPriceOverride(r.Context(), id, in.Price, reason, staff, a.now()); err != nil {
			return "", err
		}
		if in.Price == 0 {
			return "Home " + id + " is back on the list price.", nil
		}
		return "Home " + id + " now has its own price of " + kes(in.Price) + ".", nil
	})
}

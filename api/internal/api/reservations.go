package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jeffgicharu/oriel-vue/api/internal/pdf"
	"github.com/jeffgicharu/oriel-vue/api/internal/plans"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
	"github.com/jeffgicharu/oriel-vue/api/internal/viewings"
)

// ReservationInput is the reservation form.
type ReservationInput struct {
	ContactInput
	Home string `json:"home"`
	Plan string `json:"plan"`
	// Buying is "home" for a home to live in, or "investment".
	Buying string       `json:"buying"`
	Note   string       `json:"note"`
	Agree  bool         `json:"agree"`
	Source *SourceInput `json:"source"`
}

// Reservation is a request to reserve a home, as its page shows it.
type Reservation struct {
	Ref       string `json:"ref"`
	Status    string `json:"status"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Home      Home   `json:"home"`
	Price     int64  `json:"price"`
	Note      string `json:"note"`
	Agent     string `json:"agent"`
	Requested string `json:"requested"`
	HoldEnds  string `json:"holdEnds"`
	// HoldSecondsLeft is how long the hold has to run as the answer is sent.
	HoldSecondsLeft int64    `json:"holdSecondsLeft"`
	Decided         string   `json:"decided"`
	DeclineReason   string   `json:"declineReason"`
	OfferNumber     string   `json:"offerNumber"`
	Deposit         string   `json:"deposit"`
	DepositAmount   int64    `json:"depositAmount"`
	Schedule        Schedule `json:"schedule"`
}

func localTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(viewings.Nairobi).Format(slotLayout)
}

func (a *App) reservationOf(c *store.Catalog, r store.Reservation) Reservation {
	out := Reservation{
		Ref: r.Ref, Status: r.Status, Name: r.Name, Email: r.Email, Phone: r.Phone,
		Price: r.Price, Note: r.BuyerNote, Agent: r.AgentName,
		Requested: localTime(r.RequestedAt), HoldEnds: localTime(r.HoldExpiresAt),
		Decided: localTime(r.DecidedAt), DeclineReason: r.DeclineReason, OfferNumber: r.OfferNumber,
		Deposit: localTime(r.DepositAt), DepositAmount: r.DepositAmount,
	}
	if left := r.HoldExpiresAt.Sub(a.now()); r.Status == "requested" && left > 0 {
		out.HoldSecondsLeft = int64(left.Seconds())
	}
	if h, ok := c.Home(r.UnitID); ok {
		out.Home = homeOf(h)
	}
	plan, _ := plans.ByCode(r.Plan)
	s, err := plans.Build(plan, r.Price, a.planOptions(r.RequestedAt))
	if err != nil {
		s, _ = plans.Build(plan, r.Price, a.planOptions(a.now()))
	}
	out.Schedule = scheduleOf(s)
	return out
}

func (a *App) reserve(w http.ResponseWriter, r *http.Request) {
	var in ReservationInput
	if !a.readJSON(w, r, &in) {
		return
	}
	if in.Website != "" {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Some answers need another look.", fieldErrors{"agree": "Please confirm you understand how a reservation works."})
		return
	}
	errs := fieldErrors{}
	c := contact(in.ContactInput, errs)
	if _, ok := plans.ByCode(in.Plan); !ok {
		errs["plan"] = "Please choose how you will pay."
	}
	if !in.Agree {
		errs["agree"] = "Please confirm you understand how a reservation works."
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > 600 {
		errs["note"] = "Please keep this under 600 characters."
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
	cat, err := db.LoadCatalog(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	h, found := cat.Home(in.Home)
	if !found {
		a.notFound(w, "There is no home "+in.Home+" at Oriel.")
		return
	}
	c.Buying = "home"
	if in.Buying == "investment" {
		c.Buying = "investment"
	}
	c.Interest = h.Type.Short + ", home " + h.ID
	c.Attribution = a.attributionOf(in.Source)
	res, err := db.RequestReservation(r.Context(), store.ReservationRequest{Contact: c, UnitID: h.ID, Plan: in.Plan, Note: note}, h.Price, a.now())
	if errors.Is(err, store.ErrNotAvailable) {
		a.problem(w, http.StatusConflict, "not-available", "Home "+h.ID+" was reserved by someone else a moment ago.", fieldErrors{"home": "This home is no longer available."})
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID,
		event{Type: "unit", ID: h.ID, Status: store.Held},
		event{Type: "reservation", ID: res.Ref, Status: res.Status, Text: c.Name + " requested " + h.ID},
	)
	if cat, err = db.LoadCatalog(r.Context()); err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, a.reservationOf(cat, res))
}

func (a *App) reservation(w http.ResponseWriter, r *http.Request) {
	db := a.db(r)
	if !db.ReadOnly() {
		if err := db.LapseHolds(r.Context(), a.now()); err != nil {
			a.log.Error("lapsing holds", "err", err)
		}
	}
	res, err := db.ReservationByRef(r.Context(), r.PathValue("ref"))
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "There is no reservation "+r.PathValue("ref")+" in this browser.")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.reservationOf(c, res))
}

// offerPDF is a confirmed reservation's letter of offer.
func (a *App) offerPDF(w http.ResponseWriter, r *http.Request) {
	db := a.db(r)
	res, err := db.ReservationByRef(r.Context(), r.PathValue("ref"))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && res.OfferNumber == "") {
		a.notFound(w, "There is no letter of offer for "+r.PathValue("ref")+".")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	h, found := c.Home(res.UnitID)
	if !found {
		a.notFound(w, "There is no home "+res.UnitID+".")
		return
	}
	plan, _ := plans.ByCode(res.Plan)
	from := res.DecidedAt
	if from.IsZero() {
		from = res.RequestedAt
	}
	s, err := plans.Build(plan, res.Price, a.planOptions(from))
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	manager, err := db.FirstWithRole(r.Context(), "manager")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	body, err := pdf.OfferLetter(pdf.Offer{
		Reservation: res, Home: h, HomeTitle: homeTitle(h), Schedule: s, Plan: planFor(h.Unit),
		Manager: manager.Name, Parking: h.Type.Parking, Issued: from, Nairobi: store.Nairobi(), Site: a.siteHost(),
	})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writePDF(w, "oriel-letter-of-offer-"+strings.ToLower(res.OfferNumber)+".pdf", body)
}

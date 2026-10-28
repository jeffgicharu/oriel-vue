package api

import "net/http"

func (a *App) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/healthz", a.healthz)
	mux.HandleFunc("GET /api/events", a.events)

	// The building and its homes.
	mux.HandleFunc("GET /api/overview", a.overview)
	mux.HandleFunc("GET /api/building", a.buildingModel)
	mux.HandleFunc("GET /api/progress", a.progress)
	mux.HandleFunc("GET /api/homes", a.homes)
	mux.HandleFunc("GET /api/homes/{id}", a.home)
	mux.HandleFunc("GET /api/homes/{id}/schedule", a.schedule)
	mux.HandleFunc("GET /api/types/{code}", a.homeType)
	mux.HandleFunc("GET /api/plans", a.typePlans)
	mux.HandleFunc("GET /api/payment-plans", a.paymentPlans)
	mux.HandleFunc("GET /api/compare", a.compare)
	mux.HandleFunc("GET /api/price-list.pdf", a.priceListPDF)

	// What visitors send.
	mux.HandleFunc("POST /api/enquiries", a.enquire)
	mux.HandleFunc("POST /api/whatsapp", a.whatsapp)
	mux.HandleFunc("GET /api/viewings/calendar", a.calendar)
	mux.HandleFunc("POST /api/viewings", a.bookViewing)
	mux.HandleFunc("GET /api/viewings/{ref}", a.viewing)
	mux.HandleFunc("POST /api/viewings/{ref}/cancel", a.cancelViewing)
	mux.HandleFunc("GET /api/viewings/{ref}/calendar.ics", a.viewingCalendar)
	mux.HandleFunc("POST /api/reservations", a.reserve)
	mux.HandleFunc("GET /api/reservations/{ref}", a.reservation)
	mux.HandleFunc("GET /api/reservations/{ref}/offer.pdf", a.offerPDF)
	mux.HandleFunc("GET /api/privacy", a.privacy)
	mux.HandleFunc("POST /api/privacy/delete", a.deleteMine)

	// The sales desk. a.desk lets in any member of the team; a.managers only
	// the sales manager.
	mux.HandleFunc("POST /api/desk/session", a.deskSignIn)
	mux.HandleFunc("GET /api/desk/session", a.desk(a.deskSession))
	mux.HandleFunc("DELETE /api/desk/session", a.deskSignOut)
	mux.HandleFunc("GET /api/desk/enter/{role}/{sig}", a.deskEnter)
	mux.HandleFunc("GET /api/desk/events", a.desk(a.events))
	mux.HandleFunc("GET /api/desk/team", a.desk(a.deskTeam))
	mux.HandleFunc("GET /api/desk/dashboard", a.desk(a.deskDashboard))
	mux.HandleFunc("GET /api/desk/stack", a.desk(a.deskStack))
	mux.HandleFunc("GET /api/desk/homes/{id}", a.desk(a.deskHome))
	mux.HandleFunc("POST /api/desk/homes/{id}/status", a.managers(a.deskHomeStatus))
	mux.HandleFunc("POST /api/desk/homes/{id}/price", a.managers(a.deskHomePrice))
	mux.HandleFunc("GET /api/desk/leads", a.desk(a.deskLeads))
	mux.HandleFunc("GET /api/desk/leads.csv", a.desk(a.deskLeadsCSV))
	mux.HandleFunc("POST /api/desk/leads", a.desk(a.deskAddLead))
	mux.HandleFunc("GET /api/desk/leads/{id}", a.desk(a.deskLead))
	mux.HandleFunc("POST /api/desk/leads/{id}/notes", a.desk(a.deskLeadNote))
	mux.HandleFunc("POST /api/desk/leads/{id}/stage", a.desk(a.deskLeadStage))
	mux.HandleFunc("POST /api/desk/leads/{id}/follow-up", a.desk(a.deskLeadFollowUp))
	mux.HandleFunc("POST /api/desk/leads/{id}/assign", a.managers(a.deskLeadAssign))
	mux.HandleFunc("POST /api/desk/leads/{id}/move", a.desk(a.deskLeadMove))
	mux.HandleFunc("GET /api/desk/viewings", a.desk(a.deskViewings))
	mux.HandleFunc("POST /api/desk/viewings/{ref}/outcome", a.desk(a.deskViewingOutcome))
	mux.HandleFunc("PUT /api/desk/viewings/hours", a.managers(a.deskViewingHours))
	mux.HandleFunc("POST /api/desk/viewings/blocks", a.managers(a.deskBlockDay))
	mux.HandleFunc("DELETE /api/desk/viewings/blocks/{day}", a.managers(a.deskUnblockDay))
	mux.HandleFunc("GET /api/desk/reservations", a.desk(a.deskReservations))
	mux.HandleFunc("POST /api/desk/reservations/{ref}/confirm", a.managers(a.deskConfirm))
	mux.HandleFunc("POST /api/desk/reservations/{ref}/decline", a.managers(a.deskDecline))
	mux.HandleFunc("POST /api/desk/reservations/{ref}/deposit", a.managers(a.deskDeposit))
	mux.HandleFunc("GET /api/desk/pricing", a.managers(a.deskPricing))
	mux.HandleFunc("POST /api/desk/pricing/preview", a.managers(a.deskPricingPreview))
	mux.HandleFunc("POST /api/desk/pricing", a.managers(a.deskPublishPrices))
	mux.HandleFunc("GET /api/desk/activity", a.desk(a.deskActivity))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		a.notFound(w, "There is nothing at "+r.URL.Path+".")
	})
}

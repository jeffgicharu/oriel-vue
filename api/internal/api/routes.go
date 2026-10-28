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

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		a.notFound(w, "There is nothing at "+r.URL.Path+".")
	})
}

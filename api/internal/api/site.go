package api

import (
	"math"
	"net/http"

	"github.com/jeffgicharu/oriel-vue/api/internal/building"
	"github.com/jeffgicharu/oriel-vue/api/internal/pdf"
	"github.com/jeffgicharu/oriel-vue/api/internal/programme"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// healthz answers the release scripts: which release this is, and that the
// shared sample can be read.
func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Release", a.release)
	if _, err := a.copies.Sample().LoadCatalog(r.Context()); err != nil {
		a.log.Error("health check", "err", err)
		w.Header().Del("X-Release")
		a.problem(w, http.StatusServiceUnavailable, "unhealthy", "The sample cannot be read.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"release": a.release})
}

// BuildingModel is the tower's geometry, with every home's status, for the
// drawings and the 3D view.
type BuildingModel struct {
	building.Model
	Status map[string]string `json:"status"`
}

func (a *App) buildingModel(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	m := BuildingModel{Model: building.ModelData(), Status: map[string]string{}}
	for _, h := range c.Homes {
		m.Status[h.ID] = h.Status
	}
	writeJSON(w, http.StatusOK, m)
}

// Progress is the construction programme as it stands today.
type Progress struct {
	AsOf       string      `json:"asOf"`
	Percent    int         `json:"percent"`
	Frame      float64     `json:"frame"`
	Current    string      `json:"current"`
	Complete   bool        `json:"complete"`
	Completion string      `json:"completion"`
	ShowFlats  string      `json:"showFlats"`
	Storeys    int         `json:"storeys"`
	Milestones []Milestone `json:"milestones"`
	Photos     []Photo     `json:"photos"`
}

// Milestone is a step in the programme.
type Milestone struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Title  string `json:"title"`
	Note   string `json:"note"`
	When   string `json:"when"`
	Status string `json:"status"`
}

// Photo is a dated picture of the site.
type Photo struct {
	Image   string `json:"image"`
	Taken   string `json:"taken"`
	Caption string `json:"caption"`
}

func (a *App) progress(w http.ResponseWriter, r *http.Request) {
	now := a.now()
	g := programme.On(now)
	out := Progress{
		AsOf: dateOf(now), Percent: g.Percent, Frame: math.Round(g.Frame*100) / 100, Current: g.Current,
		Complete: g.Complete, Completion: dateOf(g.Completion), ShowFlats: dateOf(programme.ShowFlatsOpen(now)),
		Storeys: programme.Storeys, Milestones: []Milestone{}, Photos: []Photo{},
	}
	for _, m := range g.Milestones {
		out.Milestones = append(out.Milestones, Milestone{Start: dateOf(m.Start), End: dateOf(m.End), Title: m.Title, Note: m.Note, When: m.When(), Status: m.Status(now)})
	}
	for _, p := range g.Photos {
		out.Photos = append(out.Photos, Photo{Image: p.Image, Taken: dateOf(p.Taken), Caption: p.Caption})
	}
	writeJSON(w, http.StatusOK, out)
}

// priceListPDF is every home for sale with its price.
func (a *App) priceListPDF(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	var listID int64
	a.db(r).QueryRowContext(r.Context(), `SELECT COALESCE(MAX(id), 1) FROM price_lists`).Scan(&listID)
	body, err := pdf.PriceList(pdf.PriceListDoc{Catalog: c, ListID: listID, AsAt: a.now(), Nairobi: store.Nairobi(), Site: a.siteHost()})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writePDF(w, "oriel-price-list.pdf", body)
}

func writePDF(w http.ResponseWriter, name string, body []byte) {
	h := w.Header()
	h.Set("Content-Type", "application/pdf")
	h.Set("Content-Disposition", `inline; filename="`+name+`"`)
	_, _ = w.Write(body)
}

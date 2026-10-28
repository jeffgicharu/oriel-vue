package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jeffgicharu/oriel-vue/api/internal/building"
	"github.com/jeffgicharu/oriel-vue/api/internal/pricing"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// PriceListInput is a price list: a base price for each type, raised for
// each floor above level 2 and for a river view.
type PriceListInput struct {
	// Base is the price in KES of each type, A to P, on level 2 without a
	// river view, to the nearest 10,000.
	Base map[string]int64 `json:"base"`
	// FloorBP is the premium for each floor above level 2, in basis points:
	// 100 is 1%.
	FloorBP int64 `json:"floorBP"`
	// RiverBP is the premium for a river view, in basis points.
	RiverBP int64 `json:"riverBP"`
}

// DeskPricingInput is a new price list, with what changed.
type DeskPricingInput struct {
	List PriceListInput `json:"list"`
	Note string         `json:"note"`
}

// DeskPriceList is a published price list.
type DeskPriceList struct {
	ID        int64            `json:"id"`
	Published string           `json:"published"`
	By        string           `json:"by"`
	Note      string           `json:"note"`
	Base      map[string]int64 `json:"base"`
	FloorBP   int64            `json:"floorBP"`
	RiverBP   int64            `json:"riverBP"`
}

// DeskPricing is the pricing page: the list in force and every list before.
type DeskPricing struct {
	Current DeskPriceList `json:"current"`
	// History is every published list, newest first, the current one
	// included.
	History []DeskPriceList `json:"history"`
	// Notice says what publishing just did; it is empty on a read.
	Notice string `json:"notice"`
}

// DeskPricePreview is what publishing a list would do to every home not yet
// reserved or sold.
type DeskPricePreview struct {
	List PriceListInput `json:"list"`
	Note string         `json:"note"`
	// Rows is each of those homes, the ones whose price changes first.
	Rows []DeskPriceRow `json:"rows"`
	// OldTotal and NewTotal are the price of all of them now and after.
	OldTotal int64 `json:"oldTotal"`
	NewTotal int64 `json:"newTotal"`
	// Changed is how many would be repriced; KeptOwn how many keep a price
	// of their own.
	Changed int `json:"changed"`
	KeptOwn int `json:"keptOwn"`
}

// DeskPriceRow is one home in a preview.
type DeskPriceRow struct {
	Home   string `json:"home"`
	Type   string `json:"type"`
	Level  int    `json:"level"`
	Status string `json:"status"`
	Old    int64  `json:"old"`
	New    int64  `json:"new"`
	Change int64  `json:"change"`
	// KeepsOwn is a home with its own price, which a new list leaves alone.
	KeepsOwn bool `json:"keepsOwn"`
}

func priceListOf(e store.PriceListEntry) DeskPriceList {
	base := map[string]int64{}
	for _, code := range building.TypeOrder {
		base[code] = e.List.Base[code]
	}
	return DeskPriceList{ID: e.ID, Published: localTime(e.PublishedAt), By: e.By, Note: e.Note, Base: base, FloorBP: e.List.FloorBP, RiverBP: e.List.RiverBP}
}

// pricingPage is the list in force and every one before it.
func (a *App) pricingPage(r *http.Request, notice string) (DeskPricing, error) {
	lists, err := a.db(r).PriceLists(r.Context())
	out := DeskPricing{History: []DeskPriceList{}, Notice: notice}
	if err != nil {
		return out, err
	}
	for i, l := range lists {
		if i == 0 {
			out.Current = priceListOf(l)
		}
		out.History = append(out.History, priceListOf(l))
	}
	return out, nil
}

func (a *App) deskPricing(w http.ResponseWriter, r *http.Request) {
	out, err := a.pricingPage(r, "")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// readPriceList checks a price list, noting what is wrong with it.
func readPriceList(in PriceListInput, errs fieldErrors) pricing.List {
	l := pricing.List{Base: map[string]int64{}, FloorBP: in.FloorBP, RiverBP: in.RiverBP}
	for _, code := range building.TypeOrder {
		p := in.Base[code]
		switch {
		case p <= 0:
			errs["base."+code] = "Type " + code + ": please give the base price in shillings."
		case p%pricing.Rounding != 0:
			errs["base."+code] = "Type " + code + ": prices are set to the nearest KES 10,000."
		}
		l.Base[code] = p
	}
	if l.FloorBP < 0 {
		errs["floorBP"] = "The floor premium should be a percentage, such as 1 or 0.8."
	}
	if l.RiverBP < 0 {
		errs["riverBP"] = "The river premium should be a percentage, such as 4."
	}
	if len(errs) == 0 {
		if err := l.Validate(); err != nil {
			errs[priceListField(err.Error())] = sentence(err.Error())
		}
	}
	return l
}

// priceListField is the field a price list's own check is about: "type B
// at KES 500 is outside..." is about type B's base price.
func priceListField(msg string) string {
	switch {
	case strings.HasPrefix(msg, "type "):
		if code, _, ok := strings.Cut(strings.TrimPrefix(msg, "type "), " "); ok {
			return "base." + code
		}
	case strings.Contains(msg, "river"):
		return "riverBP"
	}
	return "floorBP"
}

func (a *App) deskPricingPreview(w http.ResponseWriter, r *http.Request) {
	var in DeskPricingInput
	if !a.readJSON(w, r, &in) {
		return
	}
	errs := fieldErrors{}
	list := readPriceList(in.List, errs)
	note := strings.TrimSpace(in.Note)
	tooLong(errs, "note", note, 120)
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "The price list needs another look.", errs)
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	out := DeskPricePreview{List: PriceListInput{Base: list.Base, FloorBP: list.FloorBP, RiverBP: list.RiverBP}, Note: note, Rows: []DeskPriceRow{}}
	for _, h := range c.Homes {
		if h.Status == store.Reserved || h.Status == store.Sold {
			continue
		}
		row := DeskPriceRow{Home: h.ID, Type: h.Type.Code, Level: h.Level, Status: h.Status, Old: h.Price, New: list.Price(h.Unit), KeepsOwn: h.Overridden}
		if row.KeepsOwn {
			row.New = h.Price
			out.KeptOwn++
		} else if row.New != row.Old {
			out.Changed++
		}
		row.Change = row.New - row.Old
		out.OldTotal += row.Old
		out.NewTotal += row.New
		out.Rows = append(out.Rows, row)
	}
	// The homes that change come first.
	sort.SliceStable(out.Rows, func(i, j int) bool {
		ci, cj := out.Rows[i].Change != 0, out.Rows[j].Change != 0
		if ci != cj {
			return ci
		}
		if out.Rows[i].Type != out.Rows[j].Type {
			return strings.Index("ABCDP", out.Rows[i].Type) < strings.Index("ABCDP", out.Rows[j].Type)
		}
		return out.Rows[i].Home < out.Rows[j].Home
	})
	writeJSON(w, http.StatusOK, out)
}

func (a *App) deskPublishPrices(w http.ResponseWriter, r *http.Request) {
	var in DeskPricingInput
	if !a.readJSON(w, r, &in) {
		return
	}
	errs := fieldErrors{}
	list := readPriceList(in.List, errs)
	note := strings.TrimSpace(in.Note)
	tooLong(errs, "note", note, 120)
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "The price list needs another look.", errs)
		return
	}
	id, err := a.db(r).PublishPriceList(r.Context(), list, note, staffFrom(r.Context()), a.now())
	if err != nil {
		a.answerError(w, r, err, "floorBP")
		return
	}
	a.broker.publish(visitorFrom(r.Context()).copyID, event{Type: "prices", ID: strconv.FormatInt(id, 10)})
	out, err := a.pricingPage(r, "Price list "+strconv.FormatInt(id, 10)+" is in force. The website shows the new prices.")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

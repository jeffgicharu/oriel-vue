package api

import (
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/building"
	"github.com/jeffgicharu/oriel-vue/api/internal/floorplan"
	"github.com/jeffgicharu/oriel-vue/api/internal/light"
	"github.com/jeffgicharu/oriel-vue/api/internal/plans"
	"github.com/jeffgicharu/oriel-vue/api/internal/programme"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// ---------- What is sent ----------

// Home is one home as a list shows it.
type Home struct {
	ID          string   `json:"id"`
	Level       int      `json:"level"`
	Number      int      `json:"number"`
	Type        string   `json:"type"`
	TypeName    string   `json:"typeName"`
	TypeShort   string   `json:"typeShort"`
	Title       string   `json:"title"`
	Bedrooms    int      `json:"bedrooms"`
	Bathrooms   int      `json:"bathrooms"`
	Study       bool     `json:"study"`
	DSQ         bool     `json:"dsq"`
	Internal    int      `json:"internal"`
	Outdoor     int      `json:"outdoor"`
	OutdoorName string   `json:"outdoorName"`
	Facing      []string `json:"facing"`
	Facings     string   `json:"facings"`
	Aspect      string   `json:"aspect"`
	RiverView   bool     `json:"riverView"`
	Corner      bool     `json:"corner"`
	// Sun is the hours of direct sun on an average day, from its light study.
	Sun    float64 `json:"sun"`
	Status string  `json:"status"`
	// Price is null when the home is reserved, sold or withheld: the site
	// shows prices only for homes a buyer can still have.
	Price *int64 `json:"price"`
}

// TypeSummary is one kind of home, with what is left of it.
type TypeSummary struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Short       string   `json:"short"`
	Bedrooms    int      `json:"bedrooms"`
	Bathrooms   int      `json:"bathrooms"`
	Study       bool     `json:"study"`
	DSQ         bool     `json:"dsq"`
	Internal    int      `json:"internal"`
	Outdoor     int      `json:"outdoor"`
	OutdoorName string   `json:"outdoorName"`
	Parking     int      `json:"parking"`
	Summary     string   `json:"summary"`
	Features    []string `json:"features"`
	Total       int      `json:"total"`
	Available   int      `json:"available"`
	// Withheld is how many are held back for a later release.
	Withheld int `json:"withheld"`
	// From is the lowest price among the homes still available; null when
	// none is.
	From *int64 `json:"from"`
}

// Overview is what the home page and the site's frame say about the building
// as it stands.
type Overview struct {
	AsOf       string         `json:"asOf"`
	Available  int            `json:"available"`
	Total      int            `json:"total"`
	From       int64          `json:"from"`
	Counts     map[string]int `json:"counts"`
	Types      []TypeSummary  `json:"types"`
	Completion string         `json:"completion"`
	Progress   ProgressBrief  `json:"progress"`
	PriceList  string         `json:"priceList"`
}

// ProgressBrief is where the construction stands, in a line.
type ProgressBrief struct {
	Percent int     `json:"percent"`
	Current string  `json:"current"`
	Frame   float64 `json:"frame"`
}

// Filter is the residences list's filters, normalised: anything unknown in
// the request is dropped.
type Filter struct {
	Beds   int    `json:"beds"`
	Max    int64  `json:"max"`
	Floors string `json:"floors"`
	Side   string `json:"side"`
	All    bool   `json:"all"`
	Sort   string `json:"sort"`
}

// HomeList is the residences list.
type HomeList struct {
	Homes     []Home        `json:"homes"`
	Filter    Filter        `json:"filter"`
	Available int           `json:"available"`
	Total     int           `json:"total"`
	From      int64         `json:"from"`
	Types     []TypeSummary `json:"types"`
}

// HomeDetail is everything a home's page shows.
type HomeDetail struct {
	Home Home        `json:"home"`
	Type TypeSummary `json:"type"`
	// Plan is drawn as the home is built: mirrored where it is handed, with
	// north at North degrees clockwise from the top of the drawing.
	Plan      Plan       `json:"plan"`
	Handed    bool       `json:"handed"`
	North     int        `json:"north"`
	Light     Light      `json:"light"`
	Floor     Floor      `json:"floor"`
	Schedules []Schedule `json:"schedules"`
	Similar   []Home     `json:"similar"`
}

// Floor is the level a home is on, from above, for its key plan. Rectangles
// are [x0, z0, x1, z1] in metres, north towards smaller z.
type Floor struct {
	Level int         `json:"level"`
	Plate [4]float64  `json:"plate"`
	Core  *[4]float64 `json:"core"`
	Homes []FloorHome `json:"homes"`
}

// FloorHome is one home's place on its floor.
type FloorHome struct {
	Number int        `json:"number"`
	Foot   [4]float64 `json:"foot"`
	Bay    [4]float64 `json:"bay"`
}

// Light is a home's light study: the hours of direct sun on a typical day in
// each month.
type Light struct {
	Months       []LightMonth `json:"months"`
	Best         int          `json:"best"`
	Worst        int          `json:"worst"`
	AverageDay   float64      `json:"averageDay"`
	MorningShare float64      `json:"morningShare"`
}

// LightMonth is one month of a light study.
type LightMonth struct {
	Month     int     `json:"month"`
	Morning   float64 `json:"morning"`
	Afternoon float64 `json:"afternoon"`
}

// Schedule is what a buyer pays on one plan, and when.
type Schedule struct {
	Plan        PaymentPlan  `json:"plan"`
	ListPrice   int64        `json:"listPrice"`
	Discount    int64        `json:"discount"`
	Price       int64        `json:"price"`
	Completion  string       `json:"completion"`
	Instalments []Instalment `json:"instalments"`
	Mortgage    *Mortgage    `json:"mortgage"`
}

// PaymentPlan is one way of paying.
type PaymentPlan struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	DiscountBP int64  `json:"discountBP"`
}

// Instalment is one payment.
type Instalment struct {
	Label     string `json:"label"`
	Due       string `json:"due"`
	Amount    int64  `json:"amount"`
	PercentBP int64  `json:"percentBP"`
	Note      string `json:"note"`
}

// Mortgage is the bank's share on the mortgage plan, with a monthly estimate.
type Mortgage struct {
	Loan       int64  `json:"loan"`
	RateBP     int64  `json:"rateBP"`
	Years      int    `json:"years"`
	Monthly    int64  `json:"monthly"`
	TotalPaid  int64  `json:"totalPaid"`
	FirstMonth string `json:"firstMonth"`
}

// TypeDetail is a kind of home with every home of it.
type TypeDetail struct {
	Type  TypeSummary `json:"type"`
	Plan  Plan        `json:"plan"`
	Homes []Home      `json:"homes"`
}

// ---------- Handlers ----------

func (a *App) catalog(w http.ResponseWriter, r *http.Request) (*store.Catalog, bool) {
	c, err := a.db(r).LoadCatalog(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return nil, false
	}
	return c, true
}

func (a *App) overview(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	now := a.now()
	g := programme.On(now)
	writeJSON(w, http.StatusOK, Overview{
		AsOf:       dateOf(now),
		Available:  c.Counts()[store.Available],
		Total:      len(c.Homes),
		From:       c.FromPrice(),
		Counts:     c.Counts(),
		Types:      typeSummaries(c),
		Completion: dateOf(g.Completion),
		Progress:   ProgressBrief{Percent: g.Percent, Current: g.Current, Frame: math.Round(g.Frame*100) / 100},
		PriceList:  dateOf(c.PriceList.PublishedAt),
	})
}

func (a *App) homes(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	f := parseFilter(r.URL.Query())
	homes := []Home{}
	var picked []store.Home
	if raw := r.URL.Query().Get("ids"); raw != "" {
		// Named homes, such as a visitor's shortlist, in the order named and
		// whatever their status.
		ids := strings.Split(raw, ",")
		if len(ids) > 12 {
			a.problem(w, http.StatusUnprocessableEntity, "validation", "Ask for up to twelve homes at a time.", map[string]string{"ids": "Ask for up to twelve homes."})
			return
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if h, ok := c.Home(strings.TrimSpace(id)); ok && !seen[h.ID] {
				seen[h.ID] = true
				picked = append(picked, h)
			}
		}
		f = Filter{Sort: "price", All: true}
	} else {
		for _, h := range c.Homes {
			if f.matches(h) {
				picked = append(picked, h)
			}
		}
		sortHomes(picked, f.Sort)
	}
	for _, h := range picked {
		homes = append(homes, homeOf(h))
	}
	writeJSON(w, http.StatusOK, HomeList{
		Homes:     homes,
		Filter:    f,
		Available: c.Counts()[store.Available],
		Total:     len(c.Homes),
		From:      c.FromPrice(),
		Types:     typeSummaries(c),
	})
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	h, found := c.Home(r.PathValue("id"))
	if !found {
		a.notFound(w, "There is no home "+r.PathValue("id")+" at Oriel.")
		return
	}
	d := detailOf(c, h)
	if h.ShowsPrice() {
		for _, p := range plans.Plans {
			if s, err := plans.Build(p, h.Price, a.planOptions(a.now())); err == nil {
				d.Schedules = append(d.Schedules, scheduleOf(s))
			}
		}
	}
	for _, s := range similarHomes(c, h, 3) {
		d.Similar = append(d.Similar, homeOf(s))
	}
	writeJSON(w, http.StatusOK, d)
}

// schedule is one plan's schedule for a home, with the buyer's own mortgage
// rate and term when they change them.
func (a *App) schedule(w http.ResponseWriter, r *http.Request) {
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	h, found := c.Home(r.PathValue("id"))
	if !found || !h.ShowsPrice() {
		a.notFound(w, "Home "+r.PathValue("id")+" has no price to work from.")
		return
	}
	q := r.URL.Query()
	plan, ok := plans.ByCode(q.Get("plan"))
	if !ok {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Choose one of the payment plans.", map[string]string{"plan": "Choose staged, cash or mortgage."})
		return
	}
	opt := a.planOptions(a.now())
	errs := map[string]string{}
	if v := q.Get("rate"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < plans.MinRateBP || n > plans.MaxRateBP {
			errs["rate"] = "Choose a rate between " + pct(plans.MinRateBP) + " and " + pct(plans.MaxRateBP) + "."
		}
		opt.RateBP = n
	}
	if v := q.Get("years"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < plans.MinYears || n > plans.MaxYears {
			errs["years"] = "Choose a term between " + strconv.Itoa(plans.MinYears) + " and " + strconv.Itoa(plans.MaxYears) + " years."
		}
		opt.Years = n
	}
	if len(errs) > 0 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "The mortgage estimate needs another look.", errs)
		return
	}
	s, err := plans.Build(plan, h.Price, opt)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, scheduleOf(s))
}

func (a *App) homeType(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(r.PathValue("code"))
	if _, ok := building.Types[code]; !ok {
		a.notFound(w, "There is no type "+r.PathValue("code")+" at Oriel.")
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	var picked []store.Home
	for _, h := range c.Homes {
		if h.Type.Code == code {
			picked = append(picked, h)
		}
	}
	sortHomes(picked, "floor")
	d := TypeDetail{Type: typeSummaryOf(c, code), Homes: []Home{}}
	p, _ := floorplan.For(code)
	d.Plan = planOf(p)
	for _, h := range picked {
		d.Homes = append(d.Homes, homeOf(h))
	}
	writeJSON(w, http.StatusOK, d)
}

// paymentPlans is each way to pay worked out on one price, for the buying
// page, which shows the plans before a home is chosen.
func (a *App) paymentPlans(w http.ResponseWriter, r *http.Request) {
	price := int64(18_000_000)
	if v := r.URL.Query().Get("price"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1_000_000 || n > 500_000_000 {
			a.problem(w, http.StatusUnprocessableEntity, "validation", "Choose a price between KES 1,000,000 and KES 500,000,000.", map[string]string{"price": "Choose a price between KES 1,000,000 and KES 500,000,000."})
			return
		}
		price = n
	}
	out := []Schedule{}
	for _, p := range plans.Plans {
		s, err := plans.Build(p, price, a.planOptions(a.now()))
		if err != nil {
			a.serverError(w, r, err)
			return
		}
		out = append(out, scheduleOf(s))
	}
	writeJSON(w, http.StatusOK, map[string][]Schedule{"schedules": out})
}

// typePlans is every type's plan, for the drawings on the residences list.
func (a *App) typePlans(w http.ResponseWriter, r *http.Request) {
	out := map[string]Plan{}
	for _, code := range building.TypeOrder {
		p, _ := floorplan.For(code)
		out[code] = planOf(p)
	}
	writeJSON(w, http.StatusOK, out)
}

// compare is up to three homes side by side, each with its plan.
func (a *App) compare(w http.ResponseWriter, r *http.Request) {
	ids := strings.Split(r.URL.Query().Get("homes"), ",")
	if len(ids) > 3 {
		a.problem(w, http.StatusUnprocessableEntity, "validation", "Compare up to three homes at a time.", map[string]string{"homes": "Choose up to three homes."})
		return
	}
	c, ok := a.catalog(w, r)
	if !ok {
		return
	}
	out := []HomeDetail{}
	seen := map[string]bool{}
	for _, id := range ids {
		h, found := c.Home(strings.TrimSpace(id))
		if !found || seen[h.ID] {
			continue
		}
		seen[h.ID] = true
		out = append(out, detailOf(c, h))
	}
	writeJSON(w, http.StatusOK, map[string][]HomeDetail{"homes": out})
}

// detailOf is a home with its plan, light study and floor; the schedules and
// similar homes are left for the caller to add.
func detailOf(c *store.Catalog, h store.Home) HomeDetail {
	study, _ := light.For(h.ID)
	return HomeDetail{
		Home:      homeOf(h),
		Type:      typeSummaryOf(c, h.Type.Code),
		Plan:      planOf(planFor(h.Unit)),
		Handed:    h.Position.Handed,
		North:     h.Position.PlanRotation,
		Light:     lightOf(study),
		Floor:     floorOf(h.Unit),
		Schedules: []Schedule{},
		Similar:   []Home{},
	}
}

// ---------- The filters ----------

func parseFilter(q url.Values) Filter {
	f := Filter{Sort: "price"}
	if n, err := strconv.Atoi(q.Get("beds")); err == nil && n >= 1 && n <= 4 {
		f.Beds = n
	}
	if n, err := strconv.ParseInt(q.Get("max"), 10, 64); err == nil && n > 0 {
		f.Max = n
	}
	switch q.Get("floors") {
	case "lower", "middle", "upper":
		f.Floors = q.Get("floors")
	}
	switch q.Get("side") {
	case "river", "garden":
		f.Side = q.Get("side")
	}
	f.All = q.Get("show") == "all"
	switch q.Get("sort") {
	case "price-desc", "floor", "size":
		f.Sort = q.Get("sort")
	}
	return f
}

func (f Filter) matches(h store.Home) bool {
	if !f.All && h.Status != store.Available {
		return false
	}
	if f.Beds > 0 && h.Type.Bedrooms != f.Beds {
		return false
	}
	if f.Max > 0 && h.Price > f.Max {
		return false
	}
	switch f.Floors {
	case "lower":
		if h.Level > 6 {
			return false
		}
	case "middle":
		if h.Level < 7 || h.Level > 11 {
			return false
		}
	case "upper":
		if h.Level < 12 {
			return false
		}
	}
	switch f.Side {
	case "river":
		if !h.Position.RiverView {
			return false
		}
	case "garden":
		if h.Position.RiverView {
			return false
		}
	}
	return true
}

func sortHomes(homes []store.Home, by string) {
	sort.SliceStable(homes, func(i, j int) bool {
		a, b := homes[i], homes[j]
		switch by {
		case "price-desc":
			return a.Price > b.Price
		case "floor":
			if a.Level != b.Level {
				return a.Level > b.Level
			}
			return a.Position.Number < b.Position.Number
		case "size":
			if a.Type.Internal != b.Type.Internal {
				return a.Type.Internal > b.Type.Internal
			}
			return a.Price < b.Price
		default:
			if a.Price != b.Price {
				return a.Price < b.Price
			}
			return a.ID < b.ID
		}
	})
}

// similarHomes are homes of the same type still for sale, nearest floors
// first, and among those the ones facing the same way.
func similarHomes(c *store.Catalog, h store.Home, n int) []store.Home {
	var out []store.Home
	for _, other := range c.Homes {
		if other.ID != h.ID && other.Type.Code == h.Type.Code && other.OnSale() {
			out = append(out, other)
		}
	}
	abs := func(n int) int {
		if n < 0 {
			return -n
		}
		return n
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := abs(out[i].Level-h.Level), abs(out[j].Level-h.Level)
		if di != dj {
			return di < dj
		}
		return out[i].Facings() == h.Facings() && out[j].Facings() != h.Facings()
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// ---------- Turning the domain into what is sent ----------

// planOptions are the options for a schedule from a reservation made at
// from, against the completion date as the programme stands today.
func (a *App) planOptions(from time.Time) plans.Options {
	return plans.Options{From: from, Completion: programme.Completion(a.now())}
}

// planFor is the drawing of a home as it really sits: mirrored where it is
// handed.
func planFor(u building.Unit) floorplan.Plan {
	p, _ := floorplan.For(u.Type.Code)
	if u.Position.Handed {
		p = p.Mirrored()
	}
	return p
}

func homeTitle(h store.Home) string {
	if h.Type.Code == "P" {
		return "Penthouse on the top floor"
	}
	return h.Type.Name + " on level " + strconv.Itoa(h.Level)
}

func homeOf(h store.Home) Home {
	out := Home{
		ID:          h.ID,
		Level:       h.Level,
		Number:      h.Position.Number,
		Type:        h.Type.Code,
		TypeName:    h.Type.Name,
		TypeShort:   h.Type.Short,
		Title:       homeTitle(h),
		Bedrooms:    h.Type.Bedrooms,
		Bathrooms:   h.Type.Bathrooms,
		Study:       h.Type.Study,
		DSQ:         h.Type.DSQ,
		Internal:    h.Type.Internal,
		Outdoor:     h.Type.Outdoor,
		OutdoorName: h.Type.OutdoorName,
		Facing:      []string{},
		Facings:     h.Facings(),
		Aspect:      h.Aspect(),
		RiverView:   h.Position.RiverView,
		Corner:      h.Position.Bay.Corner,
		Status:      h.Status,
	}
	if s, ok := light.For(h.ID); ok {
		out.Sun = math.Round(s.AverageDay*10) / 10
	}
	for _, f := range h.Position.Facing {
		out.Facing = append(out.Facing, string(f))
	}
	if h.ShowsPrice() {
		price := h.Price
		out.Price = &price
	}
	return out
}

func typeSummaries(c *store.Catalog) []TypeSummary {
	out := []TypeSummary{}
	for _, t := range c.Types() {
		out = append(out, typeSummary(t))
	}
	return out
}

func typeSummaryOf(c *store.Catalog, code string) TypeSummary {
	for _, t := range c.Types() {
		if t.Code == code {
			return typeSummary(t)
		}
	}
	return TypeSummary{}
}

func typeSummary(t store.TypeSummary) TypeSummary {
	out := TypeSummary{
		Code: t.Code, Name: t.Name, Short: t.Short,
		Bedrooms: t.Bedrooms, Bathrooms: t.Bathrooms, Study: t.Study, DSQ: t.DSQ,
		Internal: t.Internal, Outdoor: t.Outdoor, OutdoorName: t.OutdoorName, Parking: t.Parking,
		Summary: t.Summary, Features: append([]string{}, t.Features...),
		Total: t.Total, Available: t.Available, Withheld: t.Withheld,
	}
	if t.From > 0 {
		from := t.From
		out.From = &from
	}
	return out
}

func floorOf(u building.Unit) Floor {
	rect := func(r building.Rect) [4]float64 {
		round := func(v float64) float64 { return math.Round(v*100) / 100 }
		return [4]float64{round(r.X0), round(r.Z0), round(r.X1), round(r.Z1)}
	}
	f := Floor{
		Level: u.Level,
		Plate: [4]float64{building.XMin, building.ZMin, building.XMax, building.ZMax},
		Homes: []FloorHome{},
	}
	positions := building.Positions
	if u.Level == building.PenthouseLevel {
		positions = building.PenthousePositions
	} else {
		core := rect(building.Core)
		f.Core = &core
	}
	for _, p := range positions {
		f.Homes = append(f.Homes, FloorHome{Number: p.Number, Foot: rect(p.Footprint), Bay: rect(p.Bay.Plan)})
	}
	return f
}

func lightOf(s light.Study) Light {
	round := func(v float64) float64 { return math.Round(v*100) / 100 }
	out := Light{Best: int(s.Best), Worst: int(s.Worst), AverageDay: round(s.AverageDay), MorningShare: round(s.MorningShare), Months: []LightMonth{}}
	for _, m := range s.Months {
		out.Months = append(out.Months, LightMonth{Month: int(m.Month), Morning: round(m.Morning), Afternoon: round(m.Afternoon)})
	}
	return out
}

func scheduleOf(s plans.Schedule) Schedule {
	out := Schedule{
		Plan:        PaymentPlan{Code: s.Plan.Code, Name: s.Plan.Name, Summary: s.Plan.Summary, DiscountBP: s.Plan.DiscountBP},
		ListPrice:   s.ListPrice,
		Discount:    s.Discount,
		Price:       s.Price,
		Completion:  dateOf(s.Completion),
		Instalments: []Instalment{},
	}
	for _, i := range s.Instalments {
		out.Instalments = append(out.Instalments, Instalment{Label: i.Label, Due: dateOf(i.Due), Amount: i.Amount, PercentBP: i.PercentBP, Note: i.Note})
	}
	if m := s.Mortgage; m != nil {
		out.Mortgage = &Mortgage{Loan: m.Loan, RateBP: m.RateBP, Years: m.Years, Monthly: m.Monthly, TotalPaid: m.TotalPaid, FirstMonth: dateOf(m.FirstMonth)}
	}
	return out
}

// dateOf writes a day as it falls in Nairobi: "2028-12-15".
func dateOf(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(store.Nairobi()).Format(time.DateOnly)
}

func pct(bp int64) string {
	whole, frac := bp/100, bp%100
	if frac == 0 {
		return strconv.FormatInt(whole, 10) + "%"
	}
	return strconv.FormatFloat(float64(bp)/100, 'f', -1, 64) + "%"
}

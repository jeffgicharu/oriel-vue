package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"

	"github.com/jeffgicharu/oriel-vue/api/internal/attribution"
	"github.com/jeffgicharu/oriel-vue/api/internal/config"
	"github.com/jeffgicharu/oriel-vue/api/internal/sample"
	"github.com/jeffgicharu/oriel-vue/api/internal/store"
)

// contract is api/openapi.yaml, loaded for checking answers against.
type contract struct {
	doc      map[string]any
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
	ops      []operation
	seen     map[string]bool
}

type operation struct {
	method, template string
	re               *regexp.Regexp
	responses        map[string]any
}

func loadContract(t *testing.T) *contract {
	t.Helper()
	raw, err := os.ReadFile("../../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var y any
	if err := yaml.Unmarshal(raw, &y); err != nil {
		t.Fatal(err)
	}
	// Through JSON, so the schema library sees the types it expects.
	j, err := json.Marshal(y)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(j))
	if err != nil {
		t.Fatal(err)
	}
	c := &contract{doc: doc.(map[string]any), compiler: jsonschema.NewCompiler(), schemas: map[string]*jsonschema.Schema{}, seen: map[string]bool{}}
	c.compiler.DefaultDraft(jsonschema.Draft2020)
	if err := c.compiler.AddResource("openapi.json", doc); err != nil {
		t.Fatal(err)
	}
	paths := c.doc["paths"].(map[string]any)
	// A path parameter such as {id} matches one segment.
	param := regexp.MustCompile(`\\\{[^\\]+\\\}`)
	for template, item := range paths {
		re := regexp.MustCompile("^" + param.ReplaceAllString(regexp.QuoteMeta(template), `[^/]+`) + "$")
		for method, op := range item.(map[string]any) {
			o := op.(map[string]any)
			c.ops = append(c.ops, operation{method: strings.ToUpper(method), template: template, re: re, responses: o["responses"].(map[string]any)})
		}
	}
	return c
}

// pointer escapes a path for a JSON pointer.
func pointer(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString("/")
		b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(p))
	}
	return b.String()
}

// check validates one answer: its status must be one the contract lists
// for the operation, its type one listed for that status, and its body must
// match the schema.
func (c *contract) check(t *testing.T, method, path string, res *http.Response, body []byte) {
	t.Helper()
	route := strings.SplitN(path, "?", 2)[0]
	var op *operation
	for i := range c.ops {
		if c.ops[i].method == method && c.ops[i].re.MatchString(route) {
			if op == nil || len(c.ops[i].template) > len(op.template) {
				op = &c.ops[i]
			}
		}
	}
	if op == nil {
		t.Errorf("%s %s is not in the contract", method, route)
		return
	}
	c.seen[method+" "+op.template] = true
	status := strconv.Itoa(res.StatusCode)
	resp, ok := op.responses[status].(map[string]any)
	if !ok {
		t.Errorf("%s %s answered %s, which the contract does not list", method, path, status)
		return
	}
	if ref, ok := resp["$ref"].(string); ok {
		resp = c.resolve(ref)
	}
	content, _ := resp["content"].(map[string]any)
	if len(content) == 0 {
		if len(body) > 0 {
			t.Errorf("%s %s answered %s with a body the contract does not describe", method, path, status)
		}
		return
	}
	ct, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if _, ok := content[ct]; !ok {
		t.Errorf("%s %s answered %s as %q, which the contract does not list", method, path, status, ct)
		return
	}
	if ct != "application/json" && ct != "application/problem+json" {
		return
	}
	loc := "openapi.json#" + pointer("paths", op.template, strings.ToLower(method), "responses", status, "content", ct, "schema")
	if ref, ok := op.responses[status].(map[string]any)["$ref"].(string); ok {
		loc = "openapi.json" + ref + pointer("content", ct, "schema")
	}
	sch, ok := c.schemas[loc]
	if !ok {
		var err error
		sch, err = c.compiler.Compile(loc)
		if err != nil {
			t.Fatalf("compiling %s: %v", loc, err)
		}
		c.schemas[loc] = sch
	}
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		t.Errorf("%s %s answered with JSON that does not parse: %v", method, path, err)
		return
	}
	if err := sch.Validate(v); err != nil {
		t.Errorf("%s %s answered %s with a body the contract does not allow:\n%v\nbody: %.600s", method, path, status, err, body)
	}
}

func (c *contract) resolve(ref string) map[string]any {
	var node any = c.doc
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		node = node.(map[string]any)[part]
	}
	return node.(map[string]any)
}

// testApp runs the API on a fresh data folder, as a visitor's browser sees it.
func testApp(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	cfg := config.Config{
		Addr:    "127.0.0.1:0",
		DataDir: t.TempDir(),
		SiteURL: "http://oriel.test",
		Secret:  []byte("0123456789abcdef0123456789abcdef"),
		DeskKey: []byte("0123456789abcdef"),
	}
	a, err := New(cfg, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(func() {
		a.Close()
		srv.Close()
		a.Shutdown()
	})
	jar, _ := cookiejar.New(nil)
	return srv, &http.Client{Jar: jar, Timeout: 10 * time.Second}
}

type caller struct {
	t      *testing.T
	srv    *httptest.Server
	client *http.Client
	c      *contract
}

func (k caller) call(method, path string, body any) (*http.Response, []byte) {
	k.t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			k.t.Fatal(err)
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, k.srv.URL+path, r)
	if err != nil {
		k.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := k.client.Do(req)
	if err != nil {
		k.t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		k.t.Fatal(err)
	}
	k.c.check(k.t, method, path, res, data)
	return res, data
}

var buyer = map[string]any{"name": "Amani Wanjiru", "email": "amani@example.com", "phone": "0712 345 678"}

func with(base map[string]any, more ...any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for i := 0; i+1 < len(more); i += 2 {
		out[more[i].(string)] = more[i+1]
	}
	return out
}

func firstFreeSlot(t *testing.T, body []byte) string {
	t.Helper()
	var cal Calendar
	if err := json.Unmarshal(body, &cal); err != nil {
		t.Fatal(err)
	}
	for _, d := range cal.Days {
		for _, s := range d.Slots {
			if d.Open && s.Left > 0 {
				return s.Start
			}
		}
	}
	t.Fatal("no free viewing time")
	return ""
}

// TestEveryAnswerKeepsToTheContract calls every operation, with good and bad
// input, and checks each answer against api/openapi.yaml. The TypeScript
// client is generated from the same file, so what the site expects and what
// the API sends cannot drift apart.
func TestEveryAnswerKeepsToTheContract(t *testing.T) {
	c := loadContract(t)
	srv, client := testApp(t)
	k := caller{t, srv, client, c}

	src := map[string]any{
		"first": map[string]any{"source": "instagram", "medium": "paid-social", "campaign": "riverside-launch", "referrer": "", "landing": "/residences/0704", "at": time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)},
		"last":  map[string]any{"source": "", "medium": "", "campaign": "", "referrer": "https://www.google.com/", "landing": "/", "at": time.Now().UTC().Format(time.RFC3339)},
	}

	// The building and its homes.
	k.call("GET", "/api/healthz", nil)
	k.call("GET", "/api/overview", nil)
	k.call("GET", "/api/building", nil)
	k.call("GET", "/api/progress", nil)
	k.call("GET", "/api/homes", nil)
	k.call("GET", "/api/homes?beds=2&floors=upper&side=river&show=all&sort=floor&max=40000000", nil)
	k.call("GET", "/api/homes?beds=9&sort=nonsense", nil)
	k.call("GET", "/api/homes?ids=0806,0704,0806,9999", nil)
	k.call("GET", "/api/homes?ids=0201,0202,0203,0204,0205,0206,0301,0302,0303,0304,0305,0306,0401", nil)
	k.call("GET", "/api/homes/0704", nil)
	k.call("GET", "/api/homes/1601", nil)
	k.call("GET", "/api/homes/9999", nil)
	k.call("GET", "/api/homes/0704/schedule?plan=mortgage&rate=1200&years=15", nil)
	k.call("GET", "/api/homes/0704/schedule?plan=staged", nil)
	k.call("GET", "/api/homes/0704/schedule?plan=mortgage&rate=99999", nil)
	k.call("GET", "/api/homes/0704/schedule?plan=layaway", nil)
	k.call("GET", "/api/homes/9999/schedule?plan=cash", nil)
	k.call("GET", "/api/types/B", nil)
	k.call("GET", "/api/types/p", nil)
	k.call("GET", "/api/types/Z", nil)
	k.call("GET", "/api/plans", nil)
	k.call("GET", "/api/payment-plans", nil)
	k.call("GET", "/api/payment-plans?price=25000000", nil)
	k.call("GET", "/api/payment-plans?price=12", nil)
	k.call("GET", "/api/compare?homes=0704,1601,0806", nil)
	k.call("GET", "/api/compare?homes=0704,1601,0806,0201", nil)
	k.call("GET", "/api/price-list.pdf", nil)

	// What visitors send.
	res, _ := k.call("POST", "/api/enquiries", map[string]any{"name": "", "email": "nope", "phone": "1", "message": ""})
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("a form with mistakes answered %d", res.StatusCode)
	}
	k.call("POST", "/api/enquiries", map[string]any{"name": "A", "unexpected": true})
	res, body := k.call("POST", "/api/enquiries", with(buyer, "message", "Is there parking for two cars?", "home", "0704", "source", src))
	if res.StatusCode != http.StatusCreated || !strings.Contains(string(body), `"firstName":"Amani"`) {
		t.Errorf("an enquiry answered %d: %s", res.StatusCode, body)
	}
	k.call("POST", "/api/whatsapp", map[string]any{"home": "0704", "source": src})

	_, body = k.call("GET", "/api/viewings/calendar", nil)
	slot := firstFreeSlot(t, body)
	k.call("POST", "/api/viewings", with(buyer, "slot", "", "party", 9))
	res, body = k.call("POST", "/api/viewings", with(buyer, "slot", slot, "party", 2, "home", "0704", "source", src))
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("booking answered %d: %s", res.StatusCode, body)
	}
	var v Viewing
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatal(err)
	}
	k.call("GET", "/api/viewings/"+v.Ref, nil)
	k.call("GET", "/api/viewings/"+v.Ref+"/calendar.ics", nil)
	k.call("GET", "/api/viewings/V-NONE0", nil)
	k.call("POST", "/api/viewings/"+v.Ref+"/cancel", nil)
	k.call("POST", "/api/viewings/V-NONE0/cancel", nil)
	// A reservation on a home still for sale, then one it can no longer take.
	_, body = k.call("GET", "/api/homes", nil)
	var list HomeList
	if err := json.Unmarshal(body, &list); err != nil || len(list.Homes) == 0 {
		t.Fatalf("no home to reserve: %s", body)
	}
	free := list.Homes[0].ID
	k.call("POST", "/api/reservations", with(buyer, "home", free, "plan", "layaway", "agree", false))
	res, body = k.call("POST", "/api/reservations", with(buyer, "home", free, "plan", "cash", "buying", "investment", "agree", true, "source", src))
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("a reservation answered %d: %s", res.StatusCode, body)
	}
	var made Reservation
	if err := json.Unmarshal(body, &made); err != nil {
		t.Fatal(err)
	}
	if made.Status != "requested" || made.Home.Status != "held" || made.HoldSecondsLeft <= 0 {
		t.Errorf("the reservation came back as %s", body)
	}
	k.call("POST", "/api/reservations", with(buyer, "home", free, "plan", "cash", "agree", true))
	k.call("POST", "/api/reservations", with(buyer, "home", "9999", "plan", "cash", "agree", true))
	k.call("GET", "/api/reservations/"+made.Ref, nil)
	k.call("GET", "/api/reservations/R-NONE0", nil)
	k.call("GET", "/api/reservations/"+made.Ref+"/offer.pdf", nil)

	res, body = k.call("GET", "/api/privacy", nil)
	if !strings.Contains(string(body), `"hasCopy":true`) {
		t.Errorf("after sending a form the browser has no copy: %s", body)
	}
	k.call("POST", "/api/privacy/delete", nil)
	_, body = k.call("GET", "/api/privacy", nil)
	if !strings.Contains(string(body), `"hasCopy":false`) {
		t.Errorf("after deleting, the browser still has a copy: %s", body)
	}
	k.call("GET", "/api/viewings/"+v.Ref, nil)

	callTheDesk(t, srv, c)

	// Every operation in the contract was called at least once, except the
	// stream, which the next test reads. The desk's stream is called signed
	// out; TestTheDeskStreamIsForTheTeam reads it signed in.
	var missed []string
	for _, op := range c.ops {
		key := op.method + " " + op.template
		if !c.seen[key] && op.template != "/api/events" {
			missed = append(missed, key)
		}
	}
	sort.Strings(missed)
	if len(missed) > 0 {
		t.Errorf("never called: %s", strings.Join(missed, ", "))
	}
}

// TestTheEventStreamOpensAndIsCapped reads the stream's first line, and checks
// that a visitor beyond the cap is told to come back rather than held.
func TestTheEventStreamOpensAndIsCapped(t *testing.T) {
	srv, client := testApp(t)
	res, err := client.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16)
	n, _ := io.ReadAtLeast(res.Body, buf, 6)
	res.Body.Close()
	if !strings.HasPrefix(string(buf[:n]), "retry:") {
		t.Fatalf("the stream opens with %q", buf[:n])
	}
}

func TestStreamsBeyondTheCapAreTurnedAway(t *testing.T) {
	a := &App{streamSlots: make(chan struct{}, 1), broker: newBroker(), log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	a.streamSlots <- struct{}{}
	rec := httptest.NewRecorder()
	a.events(rec, httptest.NewRequest("GET", "/api/events", nil))
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("a stream beyond the cap answered %d with Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}

// TestEveryAnswerCarriesTheSecurityHeaders checks a read, an error and a
// refusal alike.
func TestEveryAnswerCarriesTheSecurityHeaders(t *testing.T) {
	srv, client := testApp(t)
	for _, path := range []string{"/api/overview", "/api/homes/9999", "/api/nothing-here"} {
		res, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		for header, want := range map[string]string{
			"X-Robots-Tag":              "noindex, nofollow",
			"X-Content-Type-Options":    "nosniff",
			"Referrer-Policy":           "strict-origin-when-cross-origin",
			"X-Frame-Options":           "DENY",
			"Strict-Transport-Security": "max-age=31536000",
			"Cache-Control":             "private, no-store",
		} {
			if got := res.Header.Get(header); got != want {
				t.Errorf("%s sends %s %q, want %q", path, header, got, want)
			}
		}
	}
}

// TestFormsFromOtherSitesAreRefused posts as another site would.
func TestFormsFromOtherSitesAreRefused(t *testing.T) {
	srv, client := testApp(t)
	req, _ := http.NewRequest("POST", srv.URL+"/api/enquiries", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://elsewhere.example.com")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden || res.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("a cross-site post answered %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

// TestTheRealAddressIsBelievedOnlyFromThisMachine.
func TestTheRealAddressIsBelievedOnlyFromThisMachine(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "198.51.100.7")
	r.RemoteAddr = "127.0.0.1:41234"
	if got := clientIP(r); got != "198.51.100.7" {
		t.Errorf("from this machine the address is %q", got)
	}
	r.RemoteAddr = "203.0.113.9:5555"
	if got := clientIP(r); got != "203.0.113.9" {
		t.Errorf("from elsewhere the header was believed: %q", got)
	}
}

// deskKey is the test app's desk key, which signs the one-click links.
var deskKey = []byte("0123456789abcdef")

// deskBrowser is a browser of its own that does not follow redirects, so the
// one-click link's answer is checked as it is sent.
func deskBrowser() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:           jar,
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type deskCall struct {
	method, path string
	body         any
}

// callTheDesk calls every operation of the sales desk: signed out, signed in
// by password, and by link as the manager and as an agent, each in a browser
// and a copy of their own.
func callTheDesk(t *testing.T, srv *httptest.Server, c *contract) {
	answers := func(res *http.Response, body []byte, status int) bool {
		t.Helper()
		if res.StatusCode != status {
			t.Errorf("%s %s answered %d, want %d: %.400s", res.Request.Method, res.Request.URL.RequestURI(), res.StatusCode, status, body)
			return false
		}
		return true
	}
	decode := func(body []byte, v any) {
		t.Helper()
		if err := json.Unmarshal(body, v); err != nil {
			t.Fatalf("%v: %.300s", err, body)
		}
	}
	day := func(offset int) string {
		return time.Now().In(store.Nairobi()).AddDate(0, 0, offset).Format(time.DateOnly)
	}
	someList := map[string]any{"base": map[string]int64{"A": 9_800_000, "B": 15_400_000, "C": 21_600_000, "D": 25_900_000, "P": 52_000_000}, "floorBP": 100, "riverBP": 400}
	hours := map[string]any{"days": []int{0, 2, 3, 4, 5, 6}, "times": []string{"10:00", "11:30", "14:00", "15:30"}, "capacity": 2}

	// Signed out, everything but signing in is refused.
	nobody := caller{t, srv, deskBrowser(), c}
	for _, call := range []deskCall{
		{"GET", "/api/desk/session", nil},
		{"GET", "/api/desk/events", nil},
		{"GET", "/api/desk/team", nil},
		{"GET", "/api/desk/dashboard", nil},
		{"GET", "/api/desk/stack", nil},
		{"GET", "/api/desk/homes/0704", nil},
		{"POST", "/api/desk/homes/0704/status", map[string]any{"status": "withheld", "reason": "Show flat"}},
		{"POST", "/api/desk/homes/0704/price", map[string]any{"price": 0, "reason": "Back on the list"}},
		{"GET", "/api/desk/leads", nil},
		{"GET", "/api/desk/leads.csv", nil},
		{"POST", "/api/desk/leads", map[string]any{"name": "Amina Wekesa"}},
		{"GET", "/api/desk/leads/1", nil},
		{"POST", "/api/desk/leads/1/notes", map[string]any{"body": "Called"}},
		{"POST", "/api/desk/leads/1/stage", map[string]any{"stage": "contacted"}},
		{"POST", "/api/desk/leads/1/follow-up", map[string]any{"day": ""}},
		{"POST", "/api/desk/leads/1/assign", map[string]any{"agent": 2}},
		{"POST", "/api/desk/leads/1/move", map[string]any{"stage": "contacted"}},
		{"GET", "/api/desk/viewings", nil},
		{"POST", "/api/desk/viewings/V-NONE0/outcome", map[string]any{"outcome": "attended"}},
		{"PUT", "/api/desk/viewings/hours", hours},
		{"POST", "/api/desk/viewings/blocks", map[string]any{"day": day(9), "reason": "Staff training"}},
		{"DELETE", "/api/desk/viewings/blocks/" + day(9), nil},
		{"GET", "/api/desk/reservations", nil},
		{"POST", "/api/desk/reservations/R-NONE0/confirm", nil},
		{"POST", "/api/desk/reservations/R-NONE0/decline", map[string]any{"reason": "The buyer withdrew."}},
		{"POST", "/api/desk/reservations/R-NONE0/deposit", map[string]any{"amount": 1_000_000}},
		{"GET", "/api/desk/pricing", nil},
		{"POST", "/api/desk/pricing/preview", map[string]any{"list": someList}},
		{"POST", "/api/desk/pricing", map[string]any{"list": someList}},
		{"GET", "/api/desk/activity", nil},
	} {
		res, body := nobody.call(call.method, call.path, call.body)
		answers(res, body, http.StatusUnauthorized)
	}
	res, body := nobody.call("DELETE", "/api/desk/session", nil)
	answers(res, body, http.StatusNoContent)
	res, body = nobody.call("POST", "/api/desk/session", map[string]any{"username": "grace.wambui", "password": "not the password"})
	answers(res, body, http.StatusUnauthorized)
	res, body = nobody.call("POST", "/api/desk/session", map[string]any{"username": "grace.wambui", "pin": "1234"})
	answers(res, body, http.StatusBadRequest)
	res, body = nobody.call("GET", "/api/desk/enter/manager/not-the-signature", nil)
	answers(res, body, http.StatusNotFound)
	res, body = nobody.call("GET", "/api/desk/enter/owner/"+EnterSignature(deskKey, "owner"), nil)
	answers(res, body, http.StatusNotFound)

	// Signed in with a password, on the browser's own copy.
	byPassword := caller{t, srv, deskBrowser(), c}
	res, body = byPassword.call("POST", "/api/desk/session", map[string]any{"username": sample.TeamMembers[0].Username, "password": sample.Password})
	if answers(res, body, http.StatusOK) {
		var s DeskSession
		decode(body, &s)
		if !s.Manager || s.Staff.Name != "Grace Wambui" || s.Waiting < 1 {
			t.Errorf("signing in answered %s", body)
		}
	}
	res, body = byPassword.call("GET", "/api/desk/session", nil)
	answers(res, body, http.StatusOK)
	res, body = byPassword.call("DELETE", "/api/desk/session", nil)
	answers(res, body, http.StatusNoContent)
	res, body = byPassword.call("GET", "/api/desk/session", nil)
	answers(res, body, http.StatusUnauthorized)

	// The manager, by the one-click link.
	manager := caller{t, srv, deskBrowser(), c}
	res, body = manager.call("GET", "/api/desk/enter/manager/"+EnterSignature(deskKey, "manager"), nil)
	if answers(res, body, http.StatusSeeOther) && res.Header.Get("Location") != "/desk" {
		t.Errorf("the link goes on to %q", res.Header.Get("Location"))
	}
	res, body = manager.call("GET", "/api/desk/session", nil)
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"manager":true`) {
		t.Errorf("the manager's link signed in %s", body)
	}
	_, body = manager.call("GET", "/api/desk/team", nil)
	var team DeskTeam
	decode(body, &team)
	var agentID int64
	agentName := ""
	for _, m := range team.Team {
		if m.Role == "agent" {
			agentID, agentName = m.ID, m.Name
			break
		}
	}
	if agentID == 0 {
		t.Fatalf("the team has no agent: %s", body)
	}

	res, body = manager.call("GET", "/api/desk/dashboard", nil)
	if answers(res, body, http.StatusOK) {
		var dash DeskDashboard
		decode(body, &dash)
		web, taps := 0, 0
		for _, ch := range dash.Channels {
			web += ch.Leads
		}
		for _, ch := range dash.WhatsApp {
			taps += ch.Taps
		}
		if dash.Total != 86 || len(dash.Channels) != len(attribution.Channels) || web == 0 || taps == 0 || dash.WaitingTotal == 0 || len(dash.Months) != 8 {
			t.Errorf("the dashboard shows %d homes, %d web leads, %d taps, %d waiting: %.400s", dash.Total, web, taps, dash.WaitingTotal, body)
		}
	}

	// The stacking plan and a home's drawer.
	_, body = manager.call("GET", "/api/desk/stack", nil)
	var stack DeskStack
	decode(body, &stack)
	homes := 0
	var free DeskHome
	for _, f := range stack.Floors {
		for _, h := range f.Homes {
			homes++
			if free.ID == "" && h.Status == store.Available {
				free = h
			}
		}
	}
	if homes != 86 || free.ID == "" {
		t.Fatalf("the stacking plan has %d homes and none for sale: %.300s", homes, body)
	}
	home := "/api/desk/homes/" + free.ID
	res, body = manager.call("GET", home, nil)
	answers(res, body, http.StatusOK)
	res, body = manager.call("GET", "/api/desk/homes/9999", nil)
	answers(res, body, http.StatusNotFound)
	res, body = manager.call("POST", home+"/status", map[string]any{"status": "withheld", "reason": " "})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", home+"/status", map[string]any{"status": "sold", "reason": "Sold by hand"})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", home+"/status", map[string]any{"state": "withheld"})
	answers(res, body, http.StatusBadRequest)
	res, body = manager.call("POST", home+"/status", map[string]any{"status": "withheld", "reason": "Kept for a show flat"})
	if answers(res, body, http.StatusOK) {
		var d DeskHomeDetail
		decode(body, &d)
		if d.Home.Status != store.Withheld || d.Notice != "Home "+free.ID+" is withheld from sale." || len(d.History) == 0 {
			t.Errorf("withholding answered %s", body)
		}
	}
	res, body = manager.call("POST", home+"/status", map[string]any{"status": "available", "reason": "The show flat moved"})
	answers(res, body, http.StatusOK)
	res, body = manager.call("POST", "/api/desk/homes/9999/status", map[string]any{"status": "withheld", "reason": "Show flat"})
	answers(res, body, http.StatusNotFound)
	res, body = manager.call("POST", home+"/price", map[string]any{"price": free.ListPrice + 5_000, "reason": "Corner premium"})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", home+"/price", map[string]any{"price": free.ListPrice + 250_000, "reason": "Corner premium"})
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"overridden":true`) {
		t.Errorf("a price of its own answered %s", body)
	}
	res, body = manager.call("POST", home+"/price", map[string]any{"price": 0, "reason": "Back on the list"})
	answers(res, body, http.StatusOK)

	// Leads: the table, the board, the spreadsheet.
	res, body = manager.call("GET", "/api/desk/leads", nil)
	var leads DeskLeads
	decode(body, &leads)
	if !answers(res, body, http.StatusOK) || len(leads.Leads) == 0 || leads.Total < 200 || len(leads.Sources) == 0 {
		t.Fatalf("the leads table has %d of %d", len(leads.Leads), leads.Total)
	}
	var otherLead DeskLead
	for _, l := range leads.Leads {
		if l.AgentID != agentID && l.AgentID != 0 {
			otherLead = l
			break
		}
	}
	res, body = manager.call("GET", "/api/desk/leads?view=board", nil)
	if answers(res, body, http.StatusOK) {
		var board DeskLeads
		decode(body, &board)
		if len(board.Columns) != len(store.Stages) || len(board.Leads) != 0 {
			t.Errorf("the board has %d columns", len(board.Columns))
		}
	}
	res, body = manager.call("GET", fmt.Sprintf("/api/desk/leads?stage=viewing&agent=%d&q=a&source=Search&sort=follow", agentID), nil)
	answers(res, body, http.StatusOK)
	res, body = manager.call("GET", "/api/desk/leads?stage=nonsense&sort=nonsense", nil)
	answers(res, body, http.StatusOK)
	res, body = manager.call("GET", "/api/desk/leads.csv?source=Search", nil)
	if answers(res, body, http.StatusOK) && !strings.HasPrefix(string(body), "Name,Email,Phone,Stage,Agent,Source,Channel,Campaign,") {
		t.Errorf("the spreadsheet starts %.120q", body)
	}
	_, body = manager.call("GET", "/api/desk/leads?source=Search", nil)
	var fromSearch DeskLeads
	decode(body, &fromSearch)
	if len(fromSearch.Leads) == 0 {
		t.Errorf("no lead came from search")
	} else {
		res, body = manager.call("GET", fmt.Sprintf("/api/desk/leads/%d", fromSearch.Leads[0].ID), nil)
		var page DeskLeadPage
		decode(body, &page)
		if n := len(page.Touches); !answers(res, body, http.StatusOK) || n == 0 || page.Touches[n-1].First.Channel != attribution.Search {
			t.Errorf("a lead from search has its visits as %.400s", body)
		}
	}

	// A lead added by hand, and changed.
	res, body = manager.call("POST", "/api/desk/leads", map[string]any{"name": "", "email": "nope", "phone": "1"})
	answers(res, body, http.StatusUnprocessableEntity)
	walkIn := map[string]any{"name": "Amina Wekesa", "email": "amina.walk-in@example.com", "phone": "0722 111 222", "source": "Walk-in", "interest": "2 bed, high floor", "buying": "home", "agent": agentID, "note": "Walked in after seeing the hoarding."}
	res, body = manager.call("POST", "/api/desk/leads", walkIn)
	var added DeskLeadPage
	decode(body, &added)
	if !answers(res, body, http.StatusCreated) || added.Lead.AgentID != agentID || added.Notice != "Amina Wekesa added." {
		t.Fatalf("adding a lead answered %s", body)
	}
	res, body = manager.call("POST", "/api/desk/leads", walkIn)
	answers(res, body, http.StatusConflict)
	res, body = manager.call("POST", "/api/desk/leads", with(walkIn, "email", "amina.other@example.com", "agent", 999))
	answers(res, body, http.StatusUnprocessableEntity)

	lead := fmt.Sprintf("/api/desk/leads/%d", added.Lead.ID)
	for _, step := range []struct {
		path   string
		body   any
		status int
	}{
		{lead + "/notes", map[string]any{"body": "  "}, http.StatusUnprocessableEntity},
		{lead + "/notes", map[string]any{"text": "Called"}, http.StatusBadRequest},
		{lead + "/notes", map[string]any{"body": "Called; coming on Saturday with her sister."}, http.StatusOK},
		{"/api/desk/leads/999999/notes", map[string]any{"body": "Called"}, http.StatusNotFound},
		{lead + "/stage", map[string]any{"stage": "lost"}, http.StatusUnprocessableEntity},
		{lead + "/stage", map[string]any{"stage": "contacted"}, http.StatusOK},
		{lead + "/follow-up", map[string]any{"day": "soon"}, http.StatusUnprocessableEntity},
		{lead + "/follow-up", map[string]any{"day": day(3)}, http.StatusOK},
		{lead + "/follow-up", map[string]any{"day": ""}, http.StatusOK},
		{lead + "/assign", map[string]any{"agent": 999}, http.StatusUnprocessableEntity},
		{lead + "/assign", map[string]any{"agent": agentID}, http.StatusOK},
		{lead + "/move", map[string]any{"stage": "viewing", "after": 0, "before": 0}, http.StatusNoContent},
		{lead + "/move", map[string]any{"stage": "lost"}, http.StatusUnprocessableEntity},
		{"/api/desk/leads/999999/move", map[string]any{"stage": "contacted"}, http.StatusNotFound},
	} {
		res, body = manager.call("POST", step.path, step.body)
		answers(res, body, step.status)
	}
	res, body = manager.call("GET", lead, nil)
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"stage":"viewing"`) {
		t.Errorf("the lead after its changes is %.300s", body)
	}
	res, body = manager.call("GET", "/api/desk/leads/999999", nil)
	answers(res, body, http.StatusNotFound)
	res, body = manager.call("GET", "/api/desk/leads/abc", nil)
	answers(res, body, http.StatusNotFound)

	// Viewings: the agenda, an outcome, the hours and a closed day.
	res, body = manager.call("GET", "/api/desk/viewings", nil)
	if answers(res, body, http.StatusOK) && strings.Count(string(body), `"date":`) != 7 {
		t.Errorf("the week's agenda is %.300s", body)
	}
	res, body = manager.call("GET", "/api/desk/viewings?from=2026-02-30", nil)
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("GET", "/api/desk/viewings?from="+day(0)+"&to="+day(-1), nil)
	answers(res, body, http.StatusUnprocessableEntity)
	_, body = manager.call("GET", "/api/desk/viewings?from="+day(-20)+"&to="+day(14), nil)
	var agenda DeskAgenda
	decode(body, &agenda)
	var seen, ahead, othersSeen DeskViewing
	for _, v := range agenda.Viewings {
		if v.Status == "attended" && seen.Ref == "" {
			seen = v
		}
		if v.Status == "attended" && v.Agent != agentName && othersSeen.Ref == "" {
			othersSeen = v
		}
		if v.Status == "booked" && ahead.Ref == "" {
			ahead = v
		}
	}
	if seen.Ref == "" || ahead.Ref == "" || othersSeen.Ref == "" {
		t.Fatalf("the agenda has no past or coming viewing: %.300s", body)
	}
	res, body = manager.call("POST", "/api/desk/viewings/"+seen.Ref+"/outcome", map[string]any{"outcome": "attended", "notes": "Came with their family."})
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"notice":"Marked as came."`) {
		t.Errorf("an outcome answered %s", body)
	}
	res, body = manager.call("POST", "/api/desk/viewings/"+ahead.Ref+"/outcome", map[string]any{"outcome": "no_show"})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/viewings/"+seen.Ref+"/outcome", map[string]any{"outcome": "maybe"})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/viewings/V-NONE0/outcome", map[string]any{"outcome": "attended"})
	answers(res, body, http.StatusNotFound)
	res, body = manager.call("PUT", "/api/desk/viewings/hours", map[string]any{"days": []int{}, "times": []string{"25:00"}, "capacity": 9})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("PUT", "/api/desk/viewings/hours", hours)
	answers(res, body, http.StatusOK)
	res, body = manager.call("POST", "/api/desk/viewings/blocks", map[string]any{"day": "someday", "reason": ""})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/viewings/blocks", map[string]any{"day": day(-3), "reason": "Staff training"})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/viewings/blocks", map[string]any{"day": day(20), "reason": "Staff training"})
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"reason":"Staff training"`) {
		t.Errorf("closing a day answered %s", body)
	}
	res, body = manager.call("DELETE", "/api/desk/viewings/blocks/"+day(20), nil)
	answers(res, body, http.StatusOK)
	res, body = manager.call("DELETE", "/api/desk/viewings/blocks/someday", nil)
	answers(res, body, http.StatusNotFound)

	// Reservations: confirm one, take its deposit, decline another.
	_, body = manager.call("GET", "/api/desk/reservations", nil)
	var waiting DeskReservations
	decode(body, &waiting)
	if len(waiting.Reservations) < 2 {
		t.Fatalf("too few reservations are waiting: %.300s", body)
	}
	res, body = manager.call("GET", "/api/desk/reservations?group=closed", nil)
	answers(res, body, http.StatusOK)
	first, second := waiting.Reservations[0], waiting.Reservations[1]
	res, body = manager.call("POST", "/api/desk/reservations/"+first.Ref+"/confirm", nil)
	if answers(res, body, http.StatusOK) {
		var d DeskReservationResult
		decode(body, &d)
		if !strings.HasPrefix(d.Reservation.OfferNumber, "OR-") || d.Home.Status != store.Reserved || d.Reservation.Offer == "" {
			t.Errorf("confirming answered %s", body)
		}
	}
	res, body = manager.call("POST", "/api/desk/reservations/"+first.Ref+"/confirm", nil)
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/reservations/R-NONE0/confirm", nil)
	answers(res, body, http.StatusNotFound)
	res, body = manager.call("GET", "/api/reservations/"+first.Ref+"/offer.pdf", nil)
	if answers(res, body, http.StatusOK) && !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Errorf("the letter of offer starts %.20q", body)
	}
	res, body = manager.call("POST", "/api/desk/reservations/"+first.Ref+"/deposit", map[string]any{"amount": 0})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/reservations/"+first.Ref+"/deposit", map[string]any{"amount": first.SuggestedDeposit})
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"status":"completed"`) {
		t.Errorf("the deposit answered %s", body)
	}
	res, body = manager.call("POST", "/api/desk/reservations/"+second.Ref+"/decline", map[string]any{"reason": "no"})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/reservations/"+second.Ref+"/decline", map[string]any{"reason": "The buyer withdrew."})
	answers(res, body, http.StatusOK)
	res, body = manager.call("POST", "/api/desk/reservations/"+second.Ref+"/deposit", map[string]any{"amount": 1_000_000})
	answers(res, body, http.StatusUnprocessableEntity)

	// Pricing: preview a list, then publish it.
	_, body = manager.call("GET", "/api/desk/pricing", nil)
	var prices DeskPricing
	decode(body, &prices)
	list := map[string]any{"base": prices.Current.Base, "floorBP": prices.Current.FloorBP + 20, "riverBP": prices.Current.RiverBP}
	bad := map[string]any{"base": map[string]int64{"A": 5}, "floorBP": -1, "riverBP": 400}
	res, body = manager.call("POST", "/api/desk/pricing/preview", map[string]any{"list": list, "note": "Floor premium up"})
	if answers(res, body, http.StatusOK) {
		var p DeskPricePreview
		decode(body, &p)
		if p.Changed == 0 || p.NewTotal <= p.OldTotal {
			t.Errorf("the preview reprices %d homes", p.Changed)
		}
	}
	res, body = manager.call("POST", "/api/desk/pricing/preview", map[string]any{"list": bad})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/pricing", map[string]any{"list": bad})
	answers(res, body, http.StatusUnprocessableEntity)
	res, body = manager.call("POST", "/api/desk/pricing", map[string]any{"list": list, "note": "Floor premium up"})
	if answers(res, body, http.StatusCreated) {
		var after DeskPricing
		decode(body, &after)
		if after.Current.ID != prices.Current.ID+1 || after.Notice == "" {
			t.Errorf("publishing answered %.300s", body)
		}
	}

	// The activity log, a page at a time.
	res, body = manager.call("GET", "/api/desk/activity", nil)
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"more":true`) {
		t.Errorf("the activity log has one page: %.200s", body)
	}
	res, body = manager.call("GET", "/api/desk/activity?kind=staff&offset=60", nil)
	answers(res, body, http.StatusOK)

	// An agent, by the one-click link: their own leads, and nothing of the
	// manager's.
	agent := caller{t, srv, deskBrowser(), c}
	res, body = agent.call("GET", "/api/desk/enter/agent/"+EnterSignature(deskKey, "agent"), nil)
	answers(res, body, http.StatusSeeOther)
	res, body = agent.call("GET", "/api/desk/session", nil)
	if answers(res, body, http.StatusOK) && !strings.Contains(string(body), `"manager":false`) {
		t.Errorf("the agent's link signed in %s", body)
	}
	res, body = agent.call("GET", "/api/desk/dashboard", nil)
	answers(res, body, http.StatusOK)
	_, body = agent.call("GET", fmt.Sprintf("/api/desk/leads?agent=%d", otherLead.AgentID), nil)
	var own DeskLeads
	decode(body, &own)
	for _, l := range own.Leads {
		if l.AgentID != agentID {
			t.Errorf("an agent sees lead %d of agent %d", l.ID, l.AgentID)
			break
		}
	}
	if len(own.Leads) > 0 {
		res, body = agent.call("GET", fmt.Sprintf("/api/desk/leads/%d", own.Leads[0].ID), nil)
		answers(res, body, http.StatusOK)
	}
	other := fmt.Sprintf("/api/desk/leads/%d", otherLead.ID)
	res, body = agent.call("GET", other, nil)
	answers(res, body, http.StatusForbidden)
	res, body = agent.call("POST", other+"/notes", map[string]any{"body": "Called"})
	answers(res, body, http.StatusForbidden)
	res, body = agent.call("POST", other+"/move", map[string]any{"stage": "contacted"})
	answers(res, body, http.StatusForbidden)
	res, body = agent.call("POST", "/api/desk/viewings/"+othersSeen.Ref+"/outcome", map[string]any{"outcome": "attended"})
	answers(res, body, http.StatusForbidden)
	for _, call := range []deskCall{
		{"POST", "/api/desk/homes/0704/status", map[string]any{"status": "withheld", "reason": "Show flat"}},
		{"POST", "/api/desk/homes/0704/price", map[string]any{"price": 0, "reason": "Back on the list"}},
		{"POST", other + "/assign", map[string]any{"agent": agentID}},
		{"PUT", "/api/desk/viewings/hours", hours},
		{"POST", "/api/desk/viewings/blocks", map[string]any{"day": day(9), "reason": "Staff training"}},
		{"DELETE", "/api/desk/viewings/blocks/" + day(9), nil},
		{"POST", "/api/desk/reservations/" + first.Ref + "/confirm", nil},
		{"POST", "/api/desk/reservations/" + first.Ref + "/decline", map[string]any{"reason": "The buyer withdrew."}},
		{"POST", "/api/desk/reservations/" + first.Ref + "/deposit", map[string]any{"amount": 1_000_000}},
		{"GET", "/api/desk/pricing", nil},
		{"POST", "/api/desk/pricing/preview", map[string]any{"list": list}},
		{"POST", "/api/desk/pricing", map[string]any{"list": list}},
	} {
		res, body := agent.call(call.method, call.path, call.body)
		if answers(res, body, http.StatusForbidden) && !strings.Contains(string(body), "Only the sales manager can do that.") {
			t.Errorf("%s %s refused an agent with %s", call.method, call.path, body)
		}
	}

	res, body = manager.call("DELETE", "/api/desk/session", nil)
	answers(res, body, http.StatusNoContent)
	res, body = manager.call("GET", "/api/desk/dashboard", nil)
	answers(res, body, http.StatusUnauthorized)
}

// TestTheDeskStreamIsForTheTeam opens the desk's live updates signed in, and
// is turned away signed out.
func TestTheDeskStreamIsForTheTeam(t *testing.T) {
	srv, client := testApp(t)
	res, err := client.Get(srv.URL + "/api/desk/events")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("signed out, the desk's stream answered %d", res.StatusCode)
	}
	desk := deskBrowser()
	res, err = desk.Get(srv.URL + "/api/desk/enter/manager/" + EnterSignature(deskKey, "manager"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	res, err = desk.Get(srv.URL + "/api/desk/events")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16)
	n, _ := io.ReadAtLeast(res.Body, buf, 6)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(string(buf[:n]), "retry:") {
		t.Fatalf("signed in, the desk's stream answered %d and opens with %q", res.StatusCode, buf[:n])
	}
}

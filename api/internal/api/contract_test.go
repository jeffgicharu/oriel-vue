package api

import (
	"bytes"
	"encoding/json"
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

	"github.com/jeffgicharu/oriel-vue/api/internal/config"
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

	// Every operation in the contract was called at least once, except the
	// stream, which the next test reads.
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

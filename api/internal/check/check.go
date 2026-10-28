// Package check is the release gate. A new release runs it against itself,
// in the idle slot beside the live one, before any visitor is sent to it. It
// reads every page the web server renders and walks the API the way a
// visitor's browser would, and fails on the first thing that does not work.
//
// No browser runs on the server. The browser journeys run in CI with
// Playwright before a release is uploaded; this checks what the server
// itself sends.
package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// PublicPaths are the pages and reads checked again through nginx once a
// release is live.
func PublicPaths() []string {
	out := []string{}
	for _, p := range pages {
		if !strings.Contains(p.path, "?") {
			out = append(out, p.path)
		}
	}
	return append(out,
		"/robots.txt",
		"/sitemap.xml",
		"/site.webmanifest",
		"/favicon.svg",
		"/api/overview",
		"/api/homes",
		"/api/homes/0704",
		"/api/viewings/calendar",
		"/api/progress",
	)
}

// pages are the server-rendered pages, each with its title and a phrase from
// its content, so a page rendered empty or wrong cannot pass.
var pages = []struct {
	path   string
	title  string
	phrase string
}{
	{"/", "Oriel · 86 homes in Riverside, Nairobi", "Homes designed around the light."},
	{"/explore", "Explore the building in 3D · Oriel, Riverside", "Explore the building"},
	{"/explore?home=0704&beds=2", "Explore the building in 3D · Oriel, Riverside", "Choose a floor to open it up"},
	{"/residences", "Residences · Oriel, Riverside", "Find your home."},
	{"/residences?beds=2&floors=upper&show=all", "Residences · Oriel, Riverside", "Two bedrooms"},
	{"/residences/0704", "Home 0704, Two bedrooms · Oriel, Riverside", "Two bedrooms on level"},
	{"/residences/1601", "Home 1601, Penthouse · Oriel, Riverside", "Penthouse on the top floor"},
	{"/residences/types/A", "Type A: One bedroom · Oriel, Riverside", "One bedroom."},
	{"/residences/types/B", "Type B: Two bedrooms · Oriel, Riverside", "Two bedrooms."},
	{"/residences/types/P", "Type P: Penthouse · Oriel, Riverside", "Penthouse."},
	{"/compare?homes=0704,0806,1601", "Compare homes 0704, 0806, 1601 · Oriel, Riverside", "Side by side."},
	{"/shortlist", "Your shortlist · Oriel, Riverside", "Your shortlist."},
	{"/building", "The building · Oriel, Riverside", "Built around the light."},
	{"/location", "Location · Oriel, Riverside", "Between the forest and the city."},
	{"/progress", "Construction progress · Oriel, Riverside", "The programme"},
	{"/buying", "Buying off-plan · Oriel, Riverside", "Three ways to pay."},
	{"/viewings/new", "Book a viewing · Oriel, Riverside", "Book a viewing."},
	{"/enquire", "Ask a question · Oriel, Riverside", "Ask us anything."},
	{"/reserve/0704", "Reserve home 0704 · Oriel, Riverside", "Request a reservation."},
	{"/privacy", "Privacy · Oriel, Riverside", "Your privacy."},
	{"/terms", "Terms of use · Oriel, Riverside", "Terms of use."},
}

// Checker walks a release: its web server at Web and its API at API.
type Checker struct {
	Web     string
	API     string
	Release string
	// Token marks the checker's requests as the release gate's, so the copy
	// it writes to is thrown away and its forms skip the per-address limits.
	Token string
	// EnterManager and EnterAgent are the signatures of the desk's one-click
	// sign-in links.
	EnterManager, EnterAgent string

	client *http.Client
	out    io.Writer
	// viewing is the one the checker booked, and reserved the reservation it
	// requested, for the steps after.
	viewing  string
	reserved string
}

// New makes a checker with its own cookie jar, as a fresh visitor would have.
func New(web, api, release, token string, out io.Writer) *Checker {
	return &Checker{
		Web:     strings.TrimRight(web, "/"),
		API:     strings.TrimRight(api, "/"),
		Release: release,
		Token:   token,
		out:     out,
		client:  newClient(),
	}
}

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:     plainJar{jar},
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// plainJar keeps cookies marked Secure even though the gate talks to the
// release over plain HTTP on the loopback address. Cookies do not depend on
// the port, so the API's cookies reach the web server too, as they do in a
// browser where both share one address.
type plainJar struct{ http.CookieJar }

func (j plainJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, c := range cookies {
		c.Secure = false
	}
	j.CookieJar.SetCookies(u, cookies)
}

// Run performs every check in order and stops at the first failure.
func (c *Checker) Run(ctx context.Context) error {
	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"both halves answer as this release", c.health},
		{"every page renders on the server with its title, description, canonical link, share tags and structured data", c.pagesRender},
		{"every page asks search engines not to list it and is never cached", c.pagesPrivate},
		{"the files every page asks for load", c.assetsLoad},
		{"every link on those pages leads to a page", c.linksLead},
		{"robots.txt, the sitemap and the web manifest load", c.metadataFiles},
		{"an unknown address and an unknown home get a real 404", c.notFound},
		{"the API's reads answer with JSON, noindex and no-store", c.apiReads},
		{"the live-update stream opens", c.stream},
		{"a form with mistakes comes back with each field's problem", c.formMistakes},
		{"a visitor asks a question and taps WhatsApp", c.question},
		{"a visitor books a viewing and gets its calendar file", c.bookViewing},
		{"a booked viewing renders for the visitor who made it and nobody else", c.privateViewing},
		{"two visitors' pages rendered at once never see each other's", c.concurrentRenders},
		{"a visitor cancels the viewing", c.cancelViewing},
		{"a visitor requests a reservation, the home is held, and its page is theirs alone", c.reservation},
		{"the manager signs in by link, reads the dashboard and the stacking plan, confirms the visitor's reservation and gets its letter of offer", c.deskManager},
		{"an agent signs in by link and cannot open pricing", c.deskAgent},
		{"a visitor deletes what they sent", c.deleteMine},
	}
	for _, step := range steps {
		start := time.Now()
		if err := step.run(ctx); err != nil {
			fmt.Fprintf(c.out, "FAIL  %s: %v\n", step.name, err)
			return fmt.Errorf("%s: %w", step.name, err)
		}
		fmt.Fprintf(c.out, "ok    %s (%d ms)\n", step.name, time.Since(start).Milliseconds())
	}
	return nil
}

// ---------- Requests ----------

func (c *Checker) mark(req *http.Request) {
	req.Header.Set("User-Agent", "oriel-release-check")
	if c.Token != "" {
		req.Header.Set("X-Oriel-Check", c.Token)
	}
}

func (c *Checker) do(client *http.Client, req *http.Request) (*http.Response, string, error) {
	c.mark(req)
	res, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	return res, string(body), nil
}

func (c *Checker) get(ctx context.Context, client *http.Client, base, path string) (*http.Response, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return nil, "", err
	}
	return c.do(client, req)
}

// page fetches a page from the web server as this checker's visitor.
func (c *Checker) page(ctx context.Context, path string) (*http.Response, string, error) {
	return c.get(ctx, c.client, c.Web, path)
}

// read fetches from the API as this checker's visitor.
func (c *Checker) read(ctx context.Context, path string) (*http.Response, string, error) {
	return c.get(ctx, c.client, c.API, path)
}

// send posts JSON to the API as this checker's visitor.
func (c *Checker) send(ctx context.Context, path string, body any) (*http.Response, string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.API+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(c.client, req)
}

func want(res *http.Response, status int) error {
	if res.StatusCode != status {
		return fmt.Errorf("status %d, want %d", res.StatusCode, status)
	}
	return nil
}

// ---------- The release ----------

func (c *Checker) health(ctx context.Context) error {
	for _, target := range []struct{ base, path string }{{c.API, "/api/healthz"}, {c.Web, "/healthz"}} {
		res, _, err := c.get(ctx, c.client, target.base, target.path)
		if err != nil {
			return err
		}
		if err := want(res, http.StatusOK); err != nil {
			return fmt.Errorf("%s: %w", target.path, err)
		}
		if got := res.Header.Get("X-Release"); got != c.Release {
			return fmt.Errorf("%s reports release %q, want %q", target.path, got, c.Release)
		}
	}
	return nil
}

var (
	titleRe       = regexp.MustCompile(`<title>([^<]+)</title>`)
	descriptionRe = regexp.MustCompile(`<meta name="description" content="([^"]{40,})"`)
	canonicalRe   = regexp.MustCompile(`<link rel="canonical" href="(https?://[^"]+)"`)
	ogImageRe     = regexp.MustCompile(`<meta property="og:image" content="https?://[^"]+/assets/[^"]+"`)
	ogTitleRe     = regexp.MustCompile(`<meta property="og:title" content="[^"]+"`)
	jsonLDRe      = regexp.MustCompile(`(?s)<script type="application/ld\+json"[^>]*>(.*?)</script>`)
	robotsMetaRe  = regexp.MustCompile(`<meta name="robots" content="noindex, nofollow"`)
	assetRe       = regexp.MustCompile(`(?:href|src)="(/assets/[^"]+)"`)
	srcsetRe      = regexp.MustCompile(`srcset="([^"]+)"`)
	linkRe        = regexp.MustCompile(`<a [^>]*href="(/[^"#]*)`)
	appRe         = regexp.MustCompile(`<div id="app"[^>]*>\s*<`)
)

func htmlUnescape(s string) string {
	return strings.NewReplacer("&amp;", "&", "&#39;", "'", "&quot;", `"`, "&lt;", "<", "&gt;", ">").Replace(s)
}

func (c *Checker) pagesRender(ctx context.Context) error {
	for _, p := range pages {
		res, body, err := c.page(ctx, p.path)
		if err != nil {
			return fmt.Errorf("%s: %w", p.path, err)
		}
		if err := want(res, http.StatusOK); err != nil {
			return fmt.Errorf("%s: %w", p.path, err)
		}
		if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			return fmt.Errorf("%s is %q, not HTML", p.path, ct)
		}
		if !appRe.MatchString(body) {
			return fmt.Errorf("%s was not rendered on the server: the app's element is empty", p.path)
		}
		m := titleRe.FindStringSubmatch(body)
		if m == nil || htmlUnescape(m[1]) != p.title {
			got := ""
			if m != nil {
				got = htmlUnescape(m[1])
			}
			return fmt.Errorf("%s has the title %q, want %q", p.path, got, p.title)
		}
		if !strings.Contains(htmlUnescape(body), p.phrase) {
			return fmt.Errorf("%s does not say %q", p.path, p.phrase)
		}
		if !descriptionRe.MatchString(body) {
			return fmt.Errorf("%s has no description", p.path)
		}
		cm := canonicalRe.FindStringSubmatch(body)
		if cm == nil {
			return fmt.Errorf("%s has no canonical link", p.path)
		}
		if path := strings.SplitN(p.path, "?", 2)[0]; !strings.HasSuffix(cm[1], path) && !(path == "/" && strings.HasSuffix(cm[1], "/")) {
			return fmt.Errorf("%s names %s as its canonical address", p.path, cm[1])
		}
		if !ogTitleRe.MatchString(body) || !ogImageRe.MatchString(body) {
			return fmt.Errorf("%s is missing its share title or picture", p.path)
		}
		for _, ld := range jsonLDRe.FindAllStringSubmatch(body, -1) {
			var v any
			if err := json.Unmarshal([]byte(ld[1]), &v); err != nil {
				return fmt.Errorf("%s has structured data that does not parse: %v", p.path, err)
			}
		}
		if !robotsMetaRe.MatchString(body) {
			return fmt.Errorf("%s does not ask search engines not to list it", p.path)
		}
	}
	return nil
}

func (c *Checker) pagesPrivate(ctx context.Context) error {
	for _, p := range pages {
		res, _, err := c.page(ctx, p.path)
		if err != nil {
			return err
		}
		if got := res.Header.Get("X-Robots-Tag"); got != "noindex, nofollow" {
			return fmt.Errorf("%s sends X-Robots-Tag %q", p.path, got)
		}
		if got := res.Header.Get("Cache-Control"); got != "private, no-store" {
			return fmt.Errorf("%s sends Cache-Control %q", p.path, got)
		}
		for _, h := range []string{"X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Strict-Transport-Security", "Content-Security-Policy"} {
			if res.Header.Get(h) == "" {
				return fmt.Errorf("%s sends no %s", p.path, h)
			}
		}
	}
	return nil
}

func (c *Checker) assetsLoad(ctx context.Context) error {
	seen := map[string]bool{}
	for _, p := range pages {
		_, body, err := c.page(ctx, p.path)
		if err != nil {
			return err
		}
		var refs []string
		for _, m := range assetRe.FindAllStringSubmatch(body, -1) {
			refs = append(refs, m[1])
		}
		for _, m := range srcsetRe.FindAllStringSubmatch(body, -1) {
			for _, part := range strings.Split(m[1], ",") {
				if f := strings.Fields(strings.TrimSpace(part)); len(f) > 0 && strings.HasPrefix(f[0], "/assets/") {
					refs = append(refs, f[0])
				}
			}
		}
		for _, ref := range refs {
			if seen[ref] {
				continue
			}
			seen[ref] = true
			res, _, err := c.page(ctx, htmlUnescape(ref))
			if err != nil {
				return fmt.Errorf("%s: %w", ref, err)
			}
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("%s, asked for by %s, answers %d", ref, p.path, res.StatusCode)
			}
		}
	}
	if len(seen) < 3 {
		return fmt.Errorf("the pages ask for only %d files", len(seen))
	}
	return nil
}

func (c *Checker) linksLead(ctx context.Context) error {
	seen := map[string]bool{}
	for _, p := range pages {
		_, body, err := c.page(ctx, p.path)
		if err != nil {
			return err
		}
		for _, m := range linkRe.FindAllStringSubmatch(body, -1) {
			link := htmlUnescape(m[1])
			if seen[link] || strings.HasPrefix(link, "/api/") || strings.HasPrefix(link, "/assets/") || strings.HasPrefix(link, "/desk") {
				continue
			}
			seen[link] = true
			res, _, err := c.page(ctx, link)
			if err != nil {
				return fmt.Errorf("%s, linked from %s: %w", link, p.path, err)
			}
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("%s, linked from %s, answers %d", link, p.path, res.StatusCode)
			}
		}
	}
	return nil
}

func (c *Checker) metadataFiles(ctx context.Context) error {
	for _, f := range []struct{ path, contains string }{
		{"/robots.txt", "Sitemap:"},
		{"/sitemap.xml", "/residences/0704"},
		{"/site.webmanifest", `"name"`},
		{"/favicon.svg", "<svg"},
	} {
		res, body, err := c.page(ctx, f.path)
		if err != nil {
			return err
		}
		if err := want(res, http.StatusOK); err != nil {
			return fmt.Errorf("%s: %w", f.path, err)
		}
		if !strings.Contains(body, f.contains) {
			return fmt.Errorf("%s does not contain %q", f.path, f.contains)
		}
	}
	return nil
}

func (c *Checker) notFound(ctx context.Context) error {
	for _, path := range []string{"/no-such-page", "/residences/9999", "/residences/types/Z"} {
		res, body, err := c.page(ctx, path)
		if err != nil {
			return err
		}
		if err := want(res, http.StatusNotFound); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !strings.Contains(body, "Page not found") || !robotsMetaRe.MatchString(body) {
			return fmt.Errorf("%s is not Oriel's not-found page", path)
		}
	}
	return nil
}

func (c *Checker) apiReads(ctx context.Context) error {
	for _, path := range []string{"/api/overview", "/api/homes?beds=2", "/api/homes/0704", "/api/homes/0704/schedule?plan=mortgage&rate=1200&years=15", "/api/types/B", "/api/compare?homes=0704,1601", "/api/building", "/api/progress", "/api/viewings/calendar"} {
		res, body, err := c.read(ctx, path)
		if err != nil {
			return err
		}
		if err := want(res, http.StatusOK); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if ct := res.Header.Get("Content-Type"); ct != "application/json" {
			return fmt.Errorf("%s is %q", path, ct)
		}
		if res.Header.Get("X-Robots-Tag") != "noindex, nofollow" || res.Header.Get("Cache-Control") != "private, no-store" {
			return fmt.Errorf("%s is missing noindex or no-store", path)
		}
		if !json.Valid([]byte(body)) {
			return fmt.Errorf("%s is not valid JSON", path)
		}
	}
	res, body, err := c.read(ctx, "/api/homes/9999")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != "application/problem+json" || !strings.Contains(body, `"status":404`) {
		return fmt.Errorf("an unknown home answers %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	return nil
}

func (c *Checker) stream(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.API+"/api/events", nil)
	if err != nil {
		return err
	}
	c.mark(req)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.Header.Get("Content-Type") != "text/event-stream; charset=utf-8" {
		return fmt.Errorf("the stream is %q", res.Header.Get("Content-Type"))
	}
	buf := make([]byte, 64)
	n, err := res.Body.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if !strings.HasPrefix(string(buf[:n]), "retry:") {
		return fmt.Errorf("the stream opens with %q", string(buf[:n]))
	}
	return nil
}

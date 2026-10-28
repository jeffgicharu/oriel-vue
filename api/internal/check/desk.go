package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jeffgicharu/oriel-vue/api/internal/sample"
)

// The sales desk, walked as the manager and as an agent. Each one-click link
// makes a throwaway copy of its own, deleted afterwards. A copy made by a
// link starts from the sample, so to decide the reservation the checker
// requested, the manager also signs in by password on the checker's own copy.

// request sends a request to the API from one of the checker's browsers, with
// a body as JSON when there is one.
func (c *Checker) request(ctx context.Context, client *http.Client, method, path string, body any) (*http.Response, string, error) {
	var payload *strings.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		payload = strings.NewReader(string(data))
	} else {
		payload = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.API+path, payload)
	if err != nil {
		return nil, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(client, req)
}

// enterDesk follows a one-click link in a browser of its own, and checks it
// signs in as the role and sends the browser on to the desk.
func (c *Checker) enterDesk(ctx context.Context, role, sig string) (*http.Client, error) {
	if sig == "" {
		return nil, errors.New("no desk key to sign in with")
	}
	desk := newClient()
	res, _, err := c.request(ctx, desk, http.MethodGet, "/api/desk/dashboard", nil)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusUnauthorized {
		return nil, fmt.Errorf("the desk opened without signing in (status %d)", res.StatusCode)
	}
	res, body, err := c.request(ctx, desk, http.MethodGet, "/api/desk/enter/"+role+"/"+sig, nil)
	if err != nil {
		return nil, err
	}
	if err := want(res, http.StatusSeeOther); err != nil {
		return nil, fmt.Errorf("the %s's link: %w: %s", role, err, body)
	}
	if got := res.Header.Get("Location"); got != "/desk" {
		return nil, fmt.Errorf("the %s's link goes on to %q", role, got)
	}
	res, body, err = c.request(ctx, desk, http.MethodGet, "/api/desk/session", nil)
	if err != nil {
		return nil, err
	}
	var s struct {
		Staff struct {
			Role string `json:"role"`
		} `json:"staff"`
	}
	if err := want(res, http.StatusOK); err != nil {
		return nil, fmt.Errorf("after the %s's link: %w", role, err)
	}
	if err := json.Unmarshal([]byte(body), &s); err != nil || s.Staff.Role != role {
		return nil, fmt.Errorf("the %s's link signed in as %s", role, body)
	}
	return desk, nil
}

// deskManager signs in as the manager by link and reads the desk, then
// decides the checker's own reservation and fetches its letter of offer.
func (c *Checker) deskManager(ctx context.Context) error {
	desk, err := c.enterDesk(ctx, "manager", c.EnterManager)
	if err != nil {
		return err
	}
	defer c.request(context.WithoutCancel(ctx), desk, http.MethodPost, "/api/privacy/delete", nil)

	res, body, err := c.request(ctx, desk, http.MethodGet, "/api/desk/dashboard", nil)
	if err != nil {
		return err
	}
	var dash struct {
		Total    int `json:"total"`
		Channels []struct {
			Leads int `json:"leads"`
		} `json:"channels"`
	}
	if err := want(res, http.StatusOK); err != nil {
		return fmt.Errorf("the dashboard: %w", err)
	}
	if err := json.Unmarshal([]byte(body), &dash); err != nil || dash.Total != 86 || len(dash.Channels) == 0 {
		return fmt.Errorf("the dashboard shows %.200s", body)
	}
	res, body, err = c.request(ctx, desk, http.MethodGet, "/api/desk/stack", nil)
	if err != nil {
		return err
	}
	var stack struct {
		Floors []struct {
			Homes []struct {
				ID string `json:"id"`
			} `json:"homes"`
		} `json:"floors"`
	}
	if err := want(res, http.StatusOK); err != nil {
		return fmt.Errorf("the stacking plan: %w", err)
	}
	if err := json.Unmarshal([]byte(body), &stack); err != nil {
		return err
	}
	homes := 0
	for _, f := range stack.Floors {
		homes += len(f.Homes)
	}
	if homes != 86 {
		return fmt.Errorf("the stacking plan shows %d homes", homes)
	}

	if c.reserved == "" {
		return errors.New("no reservation was requested to decide")
	}
	res, body, err = c.send(ctx, "/api/desk/session", map[string]any{"username": sample.TeamMembers[0].Username, "password": sample.Password})
	if err != nil {
		return err
	}
	if err := want(res, http.StatusOK); err != nil {
		return fmt.Errorf("signing in on the visitor's copy: %w: %s", err, body)
	}
	defer c.request(context.WithoutCancel(ctx), c.client, http.MethodDelete, "/api/desk/session", nil)
	res, body, err = c.read(ctx, "/api/desk/reservations")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"ref":"`+c.reserved+`"`) {
		return fmt.Errorf("the reservation %s is not waiting on the desk (status %d)", c.reserved, res.StatusCode)
	}
	res, body, err = c.send(ctx, "/api/desk/reservations/"+c.reserved+"/confirm", nil)
	if err != nil {
		return err
	}
	var decided struct {
		Reservation struct {
			Status      string `json:"status"`
			OfferNumber string `json:"offerNumber"`
			Offer       string `json:"offer"`
		} `json:"reservation"`
	}
	if err := want(res, http.StatusOK); err != nil {
		return fmt.Errorf("confirming %s: %w: %s", c.reserved, err, body)
	}
	if err := json.Unmarshal([]byte(body), &decided); err != nil || decided.Reservation.Status != "confirmed" || !strings.HasPrefix(decided.Reservation.OfferNumber, "OR-") || decided.Reservation.Offer == "" {
		return fmt.Errorf("confirming %s answered %.300s", c.reserved, body)
	}
	res, body, err = c.read(ctx, decided.Reservation.Offer)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "application/pdf" || !strings.HasPrefix(body, "%PDF-") || len(body) < 10_000 {
		return fmt.Errorf("the letter of offer answers %d as %q, %d bytes", res.StatusCode, res.Header.Get("Content-Type"), len(body))
	}
	return nil
}

// deskAgent signs in as an agent by link and checks pricing stays shut.
func (c *Checker) deskAgent(ctx context.Context) error {
	desk, err := c.enterDesk(ctx, "agent", c.EnterAgent)
	if err != nil {
		return err
	}
	defer c.request(context.WithoutCancel(ctx), desk, http.MethodPost, "/api/privacy/delete", nil)
	res, body, err := c.request(ctx, desk, http.MethodGet, "/api/desk/pricing", nil)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusForbidden || !strings.Contains(body, "Only the sales manager can do that.") {
		return fmt.Errorf("an agent opened pricing (status %d)", res.StatusCode)
	}
	list := map[string]any{"base": map[string]int{"A": 1}, "floorBP": 100, "riverBP": 400}
	res, _, err = c.request(ctx, desk, http.MethodPost, "/api/desk/pricing", map[string]any{"list": list})
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusForbidden {
		return fmt.Errorf("an agent could publish prices (status %d)", res.StatusCode)
	}
	return nil
}

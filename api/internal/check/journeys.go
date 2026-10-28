package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// visitor is a made-up buyer the journeys fill the forms in as.
var visitor = map[string]any{
	"name":  "Release Check",
	"email": "release-check@example.com",
	"phone": "0712 345 678",
}

// source is where the made-up buyer came from, as a browser records it.
func source() map[string]any {
	at := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"first": map[string]any{"source": "google", "medium": "cpc", "campaign": "release-check", "referrer": "", "landing": "/residences/0704", "at": at},
		"last":  map[string]any{"source": "", "medium": "", "campaign": "", "referrer": "https://www.google.com/", "landing": "/", "at": at},
	}
}

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

type problem struct {
	Type   string            `json:"type"`
	Status int               `json:"status"`
	Errors map[string]string `json:"errors"`
}

func (c *Checker) formMistakes(ctx context.Context) error {
	res, body, err := c.send(ctx, "/api/enquiries", map[string]any{"name": "", "email": "not an address", "phone": "12", "message": "", "source": source()})
	if err != nil {
		return err
	}
	if err := want(res, http.StatusUnprocessableEntity); err != nil {
		return err
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/problem+json" {
		return fmt.Errorf("the mistakes come back as %q", ct)
	}
	var p problem
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		return err
	}
	for _, field := range []string{"name", "email", "phone", "message"} {
		if p.Errors[field] == "" {
			return fmt.Errorf("nothing is said about the %s field: %s", field, body)
		}
	}
	// A form that is not JSON at all is refused before anything is read.
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.API+"/api/enquiries", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, _, err = c.do(c.client, req)
	if err != nil {
		return err
	}
	return want(res, http.StatusUnsupportedMediaType)
}

func (c *Checker) question(ctx context.Context) error {
	res, body, err := c.send(ctx, "/api/enquiries", with(visitor, "message", "Is there parking for two cars with home 0704?", "home", "0704", "source", source()))
	if err != nil {
		return err
	}
	if err := want(res, http.StatusCreated); err != nil {
		return fmt.Errorf("%w: %s", err, body)
	}
	var got struct {
		FirstName string `json:"firstName"`
		Home      struct {
			ID string `json:"id"`
		} `json:"home"`
	}
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		return err
	}
	if got.FirstName != "Release" || got.Home.ID != "0704" {
		return fmt.Errorf("the question was answered with %s", body)
	}
	res, body, err = c.send(ctx, "/api/whatsapp", map[string]any{"home": "0704", "source": source()})
	if err != nil {
		return err
	}
	if err := want(res, http.StatusNoContent); err != nil {
		return fmt.Errorf("a WhatsApp tap: %w: %s", err, body)
	}
	return nil
}

// firstFreeSlot is the first viewing time with room in it.
func (c *Checker) firstFreeSlot(ctx context.Context) (string, error) {
	_, body, err := c.read(ctx, "/api/viewings/calendar")
	if err != nil {
		return "", err
	}
	var cal struct {
		Days []struct {
			Open  bool `json:"open"`
			Slots []struct {
				Start string `json:"start"`
				Left  int    `json:"left"`
			} `json:"slots"`
		} `json:"days"`
	}
	if err := json.Unmarshal([]byte(body), &cal); err != nil {
		return "", err
	}
	for _, d := range cal.Days {
		if !d.Open {
			continue
		}
		for _, s := range d.Slots {
			if s.Left > 0 {
				return s.Start, nil
			}
		}
	}
	return "", errors.New("there is no free viewing time in the calendar")
}

func (c *Checker) bookViewing(ctx context.Context) error {
	slot, err := c.firstFreeSlot(ctx)
	if err != nil {
		return err
	}
	res, body, err := c.send(ctx, "/api/viewings", with(visitor, "slot", slot, "party", 2, "notes", "", "home", "0704", "source", source()))
	if err != nil {
		return err
	}
	if err := want(res, http.StatusCreated); err != nil {
		return fmt.Errorf("%w: %s", err, body)
	}
	var v struct {
		Ref    string `json:"ref"`
		Status string `json:"status"`
		Start  string `json:"start"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return err
	}
	if v.Ref == "" || v.Status != "booked" || v.Start != slot {
		return fmt.Errorf("the booking came back as %s", body)
	}
	c.viewing = v.Ref
	res, body, err = c.read(ctx, "/api/viewings/"+v.Ref+"/calendar.ics")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK || !strings.Contains(body, "BEGIN:VEVENT") || !strings.Contains(body, v.Ref) {
		return fmt.Errorf("the calendar file answers %d", res.StatusCode)
	}
	return nil
}

func (c *Checker) privateViewing(ctx context.Context) error {
	if c.viewing == "" {
		return errors.New("no viewing was booked")
	}
	res, body, err := c.page(ctx, "/viewings/"+c.viewing)
	if err != nil {
		return err
	}
	if err := want(res, http.StatusOK); err != nil {
		return err
	}
	if !strings.Contains(body, c.viewing) || !strings.Contains(body, "Release Check") {
		return errors.New("the visitor's own viewing page does not show their booking")
	}
	// Another visitor, with no cookies, gets nothing.
	res, body, err = c.get(ctx, newClient(), c.Web, "/viewings/"+c.viewing)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNotFound || strings.Contains(body, "Release Check") {
		return fmt.Errorf("another visitor's request for the viewing answers %d", res.StatusCode)
	}
	return nil
}

// concurrentRenders books a second visitor's viewing, then renders both
// visitors' pages many times at once. Each must only ever see their own name:
// the web server makes a fresh app for every request, and nothing of one
// visitor's render may reach another's.
func (c *Checker) concurrentRenders(ctx context.Context) error {
	other := New(c.Web, c.API, c.Release, c.Token, c.out)
	slot, err := other.firstFreeSlot(ctx)
	if err != nil {
		return err
	}
	res, body, err := other.send(ctx, "/api/viewings", with(map[string]any{"name": "Second Visitor", "email": "second-visitor@example.com", "phone": "0722 000 111"}, "slot", slot, "party", 1, "notes", "", "home", "", "source", source()))
	if err != nil {
		return err
	}
	if err := want(res, http.StatusCreated); err != nil {
		return fmt.Errorf("%w: %s", err, body)
	}
	var v struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return err
	}
	defer other.send(context.WithoutCancel(ctx), "/api/privacy/delete", nil)

	type visitorPage struct {
		who     *Checker
		ref     string
		mine    string
		notMine string
	}
	visitors := []visitorPage{
		{c, c.viewing, "Release Check", "Second Visitor"},
		{other, v.Ref, "Second Visitor", "Release Check"},
	}
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 20; i++ {
		for _, vp := range visitors {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, body, err := vp.who.page(ctx, "/viewings/"+vp.ref)
				switch {
				case err != nil:
					errs <- err
				case res.StatusCode != http.StatusOK:
					errs <- fmt.Errorf("a viewing page answered %d", res.StatusCode)
				case !strings.Contains(body, vp.mine):
					errs <- fmt.Errorf("%s's page does not show their name", vp.mine)
				case strings.Contains(body, vp.notMine):
					errs <- fmt.Errorf("%s's page shows %s", vp.mine, vp.notMine)
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	if err, failed := <-errs; failed {
		return err
	}
	return nil
}

func (c *Checker) cancelViewing(ctx context.Context) error {
	res, body, err := c.send(ctx, "/api/viewings/"+c.viewing+"/cancel", nil)
	if err != nil {
		return err
	}
	if err := want(res, http.StatusOK); err != nil {
		return fmt.Errorf("%w: %s", err, body)
	}
	if !strings.Contains(body, `"status":"cancelled"`) {
		return fmt.Errorf("the viewing came back as %s", body)
	}
	return nil
}

func (c *Checker) deleteMine(ctx context.Context) error {
	res, _, err := c.send(ctx, "/api/privacy/delete", nil)
	if err != nil {
		return err
	}
	if err := want(res, http.StatusNoContent); err != nil {
		return err
	}
	res, _, err = c.read(ctx, "/api/viewings/"+c.viewing)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("the viewing is still there after deleting: %d", res.StatusCode)
	}
	return nil
}

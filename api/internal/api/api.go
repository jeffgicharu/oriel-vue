// Package api is Oriel's REST JSON API. The public site reads the homes, their
// prices and plans from it, and sends enquiries, viewings and reservations to
// it; the sales desk runs the business through it. The web server renders
// pages from it on the server, and browsers call it directly afterwards.
//
// The contract is api/openapi.yaml. Every response a test can reach is checked
// against it, so the TypeScript client generated from it cannot drift.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jeffgicharu/oriel-vue/api/internal/config"
	"github.com/jeffgicharu/oriel-vue/api/internal/copies"
	"github.com/jeffgicharu/oriel-vue/api/internal/sample"
)

// maxStreams is the most live-update streams open at once, across every
// visitor. Each costs a goroutine and a small buffer, so this stays far inside
// the API's memory limit.
const maxStreams = 256

// App holds what every request needs.
type App struct {
	cfg     config.Config
	release string
	log     *slog.Logger
	now     func() time.Time

	copies      *copies.Store
	broker      *broker
	copyLimiter *limiter
	formLimiter *limiter
	streamSlots chan struct{}

	// streams ends every live-update stream and the background work when the
	// server shuts down, so browsers reconnect to the release taking over.
	streams context.Context
	stop    context.CancelFunc
}

// New prepares the API: it opens the data folder, building the shared sample
// if there is none, and starts the hourly rebuild and the sweep of expired
// copies.
func New(cfg config.Config, release string, log *slog.Logger) (*App, error) {
	streams, stop := context.WithCancel(context.Background())
	opt := copies.DefaultOptions()
	opt.SampleVersion = sample.Version
	store, err := copies.Open(streams, cfg.DataDir, opt, sample.Build, log)
	if err != nil {
		stop()
		return nil, err
	}
	a := &App{
		cfg:         cfg,
		release:     release,
		log:         log,
		now:         time.Now,
		copies:      store,
		broker:      newBroker(),
		copyLimiter: newLimiter(20, 6),
		formLimiter: newLimiter(60, 12),
		streamSlots: make(chan struct{}, maxStreams),
		streams:     streams,
		stop:        stop,
	}
	go store.Run(streams)
	return a, nil
}

// Close ends live-update streams and background work. Call it when the
// server starts shutting down.
func (a *App) Close() {
	a.stop()
}

// Shutdown closes the databases once requests have finished.
func (a *App) Shutdown() {
	a.copies.Close()
}

// Handler is every route of the API, wrapped in the middleware all of them
// share.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	a.routes(mux)
	// Changes may only come from Oriel's own pages. The web server's own
	// reads carry no Origin, so they pass.
	cross := http.NewCrossOriginProtection()
	cross.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.problem(w, http.StatusForbidden, "cross-origin", "That request did not come from Oriel's own pages.", nil)
	}))
	return a.recoverer(a.headers(cross.Handler(a.withVisitor(mux))))
}

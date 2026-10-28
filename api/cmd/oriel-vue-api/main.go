// Command oriel-vue-api is Oriel's REST JSON API.
//
//	oriel-vue-api serve                  run the API (settings come from the environment)
//	oriel-vue-api version                print the release's commit id
//	oriel-vue-api check-release ...      walk a release's web server and API before it goes live
//	oriel-vue-api check-release --print-paths
//	                                     print the pages and reads to check again through nginx
//	oriel-vue-api desk-links ...         print the one-click sign-in links for the sales desk
//	oriel-vue-api model                  print the building's geometry as JSON
//	oriel-vue-api drawings DIR           write the content pages' drawings into DIR, for the web build
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	// Nairobi's time zone travels inside the binary.
	_ "time/tzdata"

	"github.com/jeffgicharu/oriel-vue/api/internal/api"
	"github.com/jeffgicharu/oriel-vue/api/internal/building"
	"github.com/jeffgicharu/oriel-vue/api/internal/check"
	"github.com/jeffgicharu/oriel-vue/api/internal/config"
	"github.com/jeffgicharu/oriel-vue/api/internal/drawings"
)

// release is the commit id, set when the binary is built.
var release = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(64)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve()
	case "version":
		fmt.Println(release)
	case "check-release":
		err = checkRelease(os.Args[2:])
	case "desk-links":
		err = deskLinks(os.Args[2:])
	case "model":
		err = json.NewEncoder(os.Stdout).Encode(building.ModelData())
	case "drawings":
		err = writeDrawings(os.Args[2:])
	default:
		usage()
		os.Exit(64)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "oriel-vue-api:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: oriel-vue-api serve | version | check-release [--env-file F] --web URL --api URL --release SHA | check-release --print-paths | desk-links [--env-file F] | model")
}

func serve() error {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return err
	}
	application, err := api.New(cfg, release, log)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	srv.RegisterOnShutdown(application.Close)

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	log.Info("serving", "addr", cfg.Addr, "release", release)

	stopped := make(chan error, 1)
	go func() { stopped <- srv.Serve(listener) }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	select {
	case err := <-stopped:
		return err
	case sig := <-signals:
		log.Info("stopping", "signal", sig.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	application.Shutdown()
	return nil
}

// deskLinks prints the two one-click links into the sales desk, one for each
// role. They are signed with the desk key; the key itself is never printed.
func deskLinks(args []string) error {
	fs := flag.NewFlagSet("desk-links", flag.ContinueOnError)
	envFile := fs.String("env-file", "", "read settings from this KEY=VALUE file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *envFile != "" {
		if err := config.LoadEnvFile(*envFile); err != nil {
			return err
		}
	}
	site := strings.TrimRight(os.Getenv("ORIEL_SITE_URL"), "/")
	key := os.Getenv("ORIEL_DESK_KEY")
	if site == "" || len(key) < 16 {
		return errors.New("desk-links needs ORIEL_SITE_URL and ORIEL_DESK_KEY")
	}
	for _, role := range []string{"manager", "agent"} {
		fmt.Printf("%s: %s/desk/enter/%s/%s\n", role, site, role, api.EnterSignature([]byte(key), role))
	}
	return nil
}

func checkRelease(args []string) error {
	fs := flag.NewFlagSet("check-release", flag.ContinueOnError)
	envFile := fs.String("env-file", "", "read settings from this KEY=VALUE file")
	web := fs.String("web", "", "address of the release's web server, such as http://127.0.0.1:8701")
	apiBase := fs.String("api", "", "address of the release's API, such as http://127.0.0.1:8703")
	want := fs.String("release", "", "the commit id the release must report")
	printPaths := fs.Bool("print-paths", false, "print the pages and reads to check through nginx, one per line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *printPaths {
		fmt.Println(strings.Join(check.PublicPaths(), "\n"))
		return nil
	}
	if *envFile != "" {
		if err := config.LoadEnvFile(*envFile); err != nil {
			return err
		}
	}
	if *web == "" || *apiBase == "" || *want == "" {
		return errors.New("check-release needs --web, --api and --release")
	}
	secret := os.Getenv("ORIEL_SECRET")
	if len(secret) < 32 {
		return errors.New("check-release needs ORIEL_SECRET, from the environment or --env-file")
	}
	deskKey := os.Getenv("ORIEL_DESK_KEY")
	if len(deskKey) < 16 {
		return errors.New("check-release needs ORIEL_DESK_KEY, from the environment or --env-file")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c := check.New(*web, *apiBase, *want, api.ReleaseCheckToken([]byte(secret)), os.Stdout)
	c.EnterManager = api.EnterSignature([]byte(deskKey), "manager")
	c.EnterAgent = api.EnterSignature([]byte(deskKey), "agent")
	return c.Run(ctx)
}

// writeDrawings writes the drawings the web app shows, made from the same
// numbers as the building and the sun study: three SVG drawings, the south
// elevation as strokes for the home page to trace, and the places on the map.
func writeDrawings(args []string) error {
	if len(args) != 1 {
		return errors.New("drawings needs the folder to write into")
	}
	dir := args[0]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string][]byte{
		"sun-path.svg":     []byte(drawings.SunPath()),
		"bay-section.svg":  []byte(drawings.BaySection()),
		"location-map.svg": []byte(drawings.LocationMap()),
	}
	for name, v := range map[string]any{"elevation.json": drawings.ElevationStrokes(), "places.json": drawings.Places} {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		files[name] = data
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

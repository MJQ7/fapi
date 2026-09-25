package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"fapi/internal/api"
	"fapi/internal/config"
	"fapi/internal/core"
	"fapi/web"
)

// serve runs fapi in the foreground until it's stopped with Ctrl+C, a stop
// signal (as systemd and Docker send), or POST /api/shutdown.
func serve(arguments []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	configFlag := flags.String("config", "", "settings file overriding the built-in defaults")
	dataDirFlag := flags.String("data-dir", "", "folder where endpoints and logs are saved")
	err := flags.Parse(arguments)
	if err != nil {
		return err // flag has already printed what was wrong
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("serve: unexpected argument %q", flags.Arg(0))
	}

	dataDir, err := config.DataDir(*dataDirFlag)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dataDir, 0o755)
	if err != nil {
		return fmt.Errorf("creating the data folder: %w", err)
	}

	logFile, err := startLogging(dataDir)
	if err != nil {
		return err
	}
	defer closeLog(logFile)

	settings, source, err := loadSettings(*configFlag, dataDir)
	if err != nil {
		return err
	}
	log.Printf("fapi %s (%s), data folder %s", version, edition(), dataDir)

	// An edition built without the web UI reports it as off, so clients
	// (GET /api/config) hide it, whatever the settings say.
	if !web.Included {
		settings.Features.WebUI = false
	}

	fapi, err := core.New(settings, dataDir)
	if err != nil {
		return err
	}

	// Claim the admin port before anything else, so a second copy of fapi
	// stops here instead of fighting over the mock ports.
	adminAddress := net.JoinHostPort(settings.ListenAddress, strconv.Itoa(settings.AdminPort))
	listener, err := net.Listen("tcp4", adminAddress)
	if err != nil {
		return fmt.Errorf("could not use the admin port %d (is fapi already running?): %w", settings.AdminPort, err)
	}

	fapi.Start()
	defer fapi.Stop()

	// stopped is a context: a value that's "done" once stopFapi is called or
	// a stop signal arrives. Waiting on stopped.Done() waits for either.
	stopped, stopFapi := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopFapi()

	handler := api.New(fapi, version, stopFapi, config.SettingsFile(source, dataDir))
	return runAdminServer(stopped, listener, handler, settings)
}

// runAdminServer serves the admin API and web UI until stopped is done.
func runAdminServer(stopped context.Context, listener net.Listener, handler http.Handler, settings config.Config) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		// Requests use stopped as their context, so long-running ones (the
		// live request stream) end when fapi is stopped.
		BaseContext: func(net.Listener) context.Context { return stopped },
	}

	// Serve runs in a goroutine; a channel brings back its error, if any.
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	switch {
	case !web.Included:
		log.Printf("Admin API on http://%s (this edition has no web UI)", listener.Addr())
	case settings.Features.WebUI:
		log.Printf("Web UI and admin API on http://%s", listener.Addr())
	default:
		log.Printf("Admin API on http://%s (the web UI is turned off)", listener.Addr())
	}

	select {
	case err := <-serveErrors:
		return fmt.Errorf("admin server: %w", err)
	case <-stopped.Done():
	}

	log.Printf("Stopping fapi")
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.Shutdown(timeout)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("stopping the admin server: %w", err)
	}
	return nil
}

// loadSettings reads the settings and logs where they came from and what
// differs from the built-in defaults. It also returns the override file it
// used, or "" for none.
func loadSettings(configFlag string, dataDir string) (config.Config, string, error) {
	settings, source, err := config.Load(configFlag, dataDir)
	if err != nil {
		return config.Config{}, "", err
	}

	if source == "" {
		log.Printf("Settings: built-in defaults (no override file)")
		return settings, source, nil
	}
	log.Printf("Settings: built-in defaults, overridden by %s", source)

	defaults, err := config.Defaults()
	if err != nil {
		return config.Config{}, "", err
	}
	differences, err := config.Differences(defaults, settings)
	if err != nil {
		return config.Config{}, "", err
	}
	for _, difference := range differences {
		log.Printf("  %s", difference)
	}
	return settings, source, nil
}

// startLogging sends log messages to the terminal and to fapi.log in the
// data folder. The file is emptied at each start, so it doesn't grow forever.
func startLogging(dataDir string) (*os.File, error) {
	path := filepath.Join(dataDir, "fapi.log")
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating the server log %s: %w", path, err)
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
	return file, nil
}

func closeLog(file *os.File) {
	log.SetOutput(os.Stderr)
	err := file.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fapi: closing the server log: %v\n", err)
	}
}

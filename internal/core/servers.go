package core

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

// startServer starts a mock server on port, unless one is already running.
// The caller must hold the mutex.
func (c *Core) startServer(port int) error {
	_, running := c.servers[port]
	if running {
		return nil
	}

	// "tcp4" listens on IPv4 only. Clients that try IPv6 first for
	// "localhost" should use 127.0.0.1 instead.
	address := net.JoinHostPort(c.settings.ListenAddress, strconv.Itoa(port))
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           c.mockHandler(port),
		ReadHeaderTimeout: 10 * time.Second,
	}
	c.servers[port] = server

	go func() {
		err := server.Serve(listener)
		// ErrServerClosed is the normal result of stopServer, not a problem.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Mock server on port %d failed: %v", port, err)
		}
	}()

	log.Printf("Mock server listening on http://%s", address)
	return nil
}

// stopServerIfUnused stops the mock server on port when no endpoint or
// pass-through uses it any more. The caller must hold the mutex.
func (c *Core) stopServerIfUnused(port int) {
	if c.portInUse(port) {
		return
	}
	c.stopServer(port)
}

// stopServer stops the mock server on port. The caller must hold the mutex.
func (c *Core) stopServer(port int) {
	server, running := c.servers[port]
	if !running {
		return
	}

	// Close stops at once instead of waiting for requests in progress.
	// Waiting (Shutdown) could wait forever here: those requests need the
	// mutex, which the caller is holding.
	err := server.Close()
	if err != nil {
		log.Printf("Stopping the mock server on port %d: %v", port, err)
	}
	delete(c.servers, port)
	log.Printf("Mock server on port %d stopped", port)
}

// mockHandler returns the function that answers every request on port.
func (c *Core) mockHandler(port int) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		c.handleMockRequest(port, writer, request)
	})
}

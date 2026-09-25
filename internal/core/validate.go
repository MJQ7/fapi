package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationError is an error caused by the user's input, as opposed to a
// problem inside fapi. Its message is worded to be shown to the user as-is.
// The admin API answers it with 400 Bad Request.
type ValidationError struct {
	Message string
}

// Error returns the message.
func (e *ValidationError) Error() string {
	return e.Message
}

func invalid(format string, values ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, values...)}
}

// methods lists the HTTP methods an endpoint can use. It's never changed.
var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// NewMock is what's needed to add an endpoint. Core adds the ID.
type NewMock struct {
	Port   int
	Method string
	Path   string
	Status int
	Body   json.RawMessage
}

// validateMock checks a new endpoint. The caller must hold the mutex.
func (c *Core) validateMock(mock NewMock) error {
	err := c.validateFapiPort(mock.Port)
	if err != nil {
		return err
	}

	if !isMethod(mock.Method) {
		return invalid("Method must be one of %s", strings.Join(methods, ", "))
	}

	err = validatePath(mock.Path)
	if err != nil {
		return err
	}

	return validateResponse(mock.Status, mock.Body)
}

// validateResponse checks the status and body of an endpoint or payload.
func validateResponse(status int, body json.RawMessage) error {
	if status < 100 || status > 599 {
		return invalid("Response status must be between 100 and 599")
	}

	if len(body) > 0 && !json.Valid(body) {
		return invalid("Response body must be valid JSON")
	}
	return nil
}

// validateFapiPort checks a port fapi would listen on.
func (c *Core) validateFapiPort(port int) error {
	portRange := c.settings.MockPorts
	if port < portRange.Min || port > portRange.Max {
		return invalid("Port must be between %d and %d", portRange.Min, portRange.Max)
	}
	if port == c.settings.AdminPort {
		return invalid("Port %d is fapi's own admin port; choose another", port)
	}
	return nil
}

// validateUpstream checks a pass-through from fapiPort to upstreamPort on
// host ("" for passThrough.host).
func (c *Core) validateUpstream(fapiPort int, upstreamPort int, host string) error {
	err := c.validateFapiPort(fapiPort)
	if err != nil {
		return err
	}

	if upstreamPort < 1 || upstreamPort > 65535 {
		return invalid("Real API port must be between 1 and 65535")
	}

	// Forwarding a port to itself would send every request back to fapi,
	// forever. That's only possible when the real API is on this machine,
	// which passThrough.host usually is; a host the user names is assumed
	// to be another machine.
	if host == "" && upstreamPort == fapiPort {
		return invalid("Real API port must not be the fapi port itself (%d)", fapiPort)
	}
	return nil
}

// validatePath checks an endpoint path such as "/api/users/:id".
func validatePath(path string) error {
	if !strings.HasPrefix(path, "/") {
		return invalid("Path must start with /")
	}
	if strings.ContainsAny(path, "?# \t") {
		return invalid("Path must not contain spaces, ? or # (query strings are not matched)")
	}
	return nil
}

func isMethod(method string) bool {
	for _, allowed := range methods {
		if method == allowed {
			return true
		}
	}
	return false
}

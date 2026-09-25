package config

import (
	"fmt"
	"net"
)

// validate checks the values that JSON types alone can't, such as ranges.
// Each error names the setting at fault, so the user knows what to fix.
func validate(settings Config) error {
	err := checkPorts(settings.Ports())
	if err != nil {
		return err
	}

	// Mock servers listen on IPv4 only (see the core package).
	address := net.ParseIP(settings.ListenAddress)
	if address == nil || address.To4() == nil {
		return fmt.Errorf(`setting "listenAddress" must be an IPv4 address such as "127.0.0.1" or "0.0.0.0", not %q`, settings.ListenAddress)
	}

	if settings.RequestLog.MaxEntries < 1 {
		return fmt.Errorf(`setting "requestLog.maxEntries" must be at least 1, not %d`, settings.RequestLog.MaxEntries)
	}
	if settings.RequestLog.MaxBodyBytes < 0 {
		return fmt.Errorf(`setting "requestLog.maxBodyBytes" must be 0 or more, not %d`, settings.RequestLog.MaxBodyBytes)
	}

	if settings.PassThrough.Host == "" {
		return fmt.Errorf(`setting "passThrough.host" must not be empty`)
	}
	return nil
}

func isPort(port int) bool {
	return port >= 1 && port <= 65535
}

package core

import (
	"net"
	"strconv"
	"strings"
)

// upstream is where a fapi port's unmatched requests are forwarded to.
type upstream struct {
	port int    // the real API port; 0 means don't forward
	host string // the real API host, or "" for passThrough.host in fapi's settings
}

// upstreamURL returns the start of the URL requests are forwarded to, such
// as "http://localhost:3000". It's also how the real API is shown in logs
// and messages.
func (c *Core) upstreamURL(target upstream) string {
	host := target.host
	if host == "" {
		host = c.settings.PassThrough.Host
	}
	return UpstreamURL(host, target.port)
}

// UpstreamURL returns the start of the URL for a real API on host and port.
// host is a host name or IP address, starting with https:// if the API uses
// HTTPS; otherwise HTTP is used. The port is left out when it's the usual
// one for the scheme, as in "https://api.example.com".
func UpstreamURL(host string, port int) string {
	scheme := "http"
	usualPort := 80
	if withoutScheme, found := strings.CutPrefix(host, "https://"); found {
		scheme = "https"
		usualPort = 443
		host = withoutScheme
	}

	if port == usualPort {
		// An IPv6 address still needs its [ ] in a URL.
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
		return scheme + "://" + host
	}
	// JoinHostPort adds the [ ] an IPv6 address needs, such as [::1]:3000.
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// normalizeUpstreamHost checks a real API host typed by the user and returns
// it the way it's saved: a host name or IP address, starting with https://
// when the API uses HTTPS. "" (or only spaces) means passThrough.host.
func normalizeUpstreamHost(text string) (string, error) {
	host := strings.TrimSpace(text)
	if host == "" {
		return "", nil
	}

	scheme := "http"
	if before, after, found := strings.Cut(host, "://"); found {
		scheme = strings.ToLower(before)
		if scheme != "http" && scheme != "https" {
			return "", invalid("The real API host must start with http://, https:// or neither, not %s://", before)
		}
		host = after
	}

	// Copying an address from the browser often brings a trailing slash.
	host = strings.TrimSuffix(host, "/")
	if host == "" {
		return "", invalid("Enter the real API host, such as api.example.com")
	}
	if strings.ContainsAny(host, "/?#@ \t") {
		return "", invalid("The real API host must be just a host name or IP address, such as api.example.com, without a path")
	}
	_, _, err := net.SplitHostPort(host)
	if err == nil {
		return "", invalid("Enter the real API port in its own box, not in the host")
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]") // [::1] is saved as ::1

	if scheme == "https" {
		return "https://" + host, nil
	}
	return host, nil
}

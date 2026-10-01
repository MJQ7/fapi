package core

// A fapi port can have several proxies, such as 3001 → 3002 and
// 3001 → 3003, but only one of them is on at a time, so it's always clear
// where a request goes. Adding a proxy or turning one on turns the others on
// its port off.

// AddProxy forwards the unmatched requests on fapiPort to upstreamPort on
// host: a host name or IP address, starting with https:// if the real API
// uses HTTPS, or "" for passThrough.host in fapi's settings. The new proxy is
// turned on, and any other on its port off. If the port already has a proxy
// to the same place, that one is turned on instead of adding another.
func (c *Core) AddProxy(fapiPort int, upstreamPort int, host string) (Proxy, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	host, err := normalizeUpstreamHost(host)
	if err != nil {
		return Proxy{}, err
	}
	err = c.validateUpstream(fapiPort, upstreamPort, host)
	if err != nil {
		return Proxy{}, err
	}

	// Start listening first: if the port is taken by another program, the
	// proxy is refused rather than saved but unreachable.
	err = c.startServer(fapiPort)
	if err != nil {
		return Proxy{}, invalid("Could not listen on port %d: %v", fapiPort, err)
	}

	for index, proxy := range c.data.Proxies {
		if proxy.Port == fapiPort && proxy.UpstreamPort == upstreamPort && proxy.Host == host {
			c.turnOn(index)
			return c.data.Proxies[index], c.save()
		}
	}

	c.data.Proxies = append(c.data.Proxies, Proxy{
		ID:           newID(),
		Port:         fapiPort,
		UpstreamPort: upstreamPort,
		Host:         host,
	})
	index := len(c.data.Proxies) - 1
	c.turnOn(index)
	return c.data.Proxies[index], c.save()
}

// SetProxyEnabled turns a proxy on or off. Turning one on turns the others
// on its port off. It returns false if there was no proxy with that ID.
func (c *Core) SetProxyEnabled(id string, enabled bool) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	index := c.proxyIndex(id)
	if index < 0 {
		return false, nil
	}
	if enabled {
		c.turnOn(index)
	} else {
		c.data.Proxies[index].Disabled = true
	}
	return true, c.save()
}

// RemoveProxy removes a proxy and stops listening on its port if nothing
// else uses it. It returns false if there was no proxy with that ID.
func (c *Core) RemoveProxy(id string) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	index := c.proxyIndex(id)
	if index < 0 {
		return false, nil
	}
	c.removeProxyAt(index)
	return true, c.save()
}

// SetUpstream is the older, one-proxy-per-port way of changing proxies,
// still used by PUT /api/upstreams/{port} (and the terminal UI). It adds a
// proxy as AddProxy does, or with an upstreamPort of 0 removes the port's
// proxy: the one shown for it (see portProxy).
func (c *Core) SetUpstream(fapiPort int, upstreamPort int, host string) error {
	if upstreamPort != 0 {
		_, err := c.AddProxy(fapiPort, upstreamPort, host)
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	index := portProxy(c.data.Proxies, fapiPort)
	if index < 0 {
		return nil
	}
	c.removeProxyAt(index)
	return c.save()
}

// SetUpstreamEnabled is the older way of turning a port's proxy on or off,
// still used by PUT /api/upstreams/{port}/enabled. It turns the port's
// shown proxy (see portProxy) on or off. It returns false if that port has
// no proxy.
func (c *Core) SetUpstreamEnabled(fapiPort int, enabled bool) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	index := portProxy(c.data.Proxies, fapiPort)
	if index < 0 {
		return false, nil
	}
	if enabled {
		c.turnOn(index)
	} else {
		c.data.Proxies[index].Disabled = true
	}
	return true, c.save()
}

// turnOn turns the proxy at index on, and the others on its port off. The
// caller must hold the mutex.
func (c *Core) turnOn(index int) {
	port := c.data.Proxies[index].Port
	for other := range c.data.Proxies {
		if c.data.Proxies[other].Port == port {
			c.data.Proxies[other].Disabled = other != index
		}
	}
}

// removeProxyAt removes the proxy at index, and stops listening on its port
// if nothing else uses it. The caller must hold the mutex.
func (c *Core) removeProxyAt(index int) {
	port := c.data.Proxies[index].Port
	c.data.Proxies = append(c.data.Proxies[:index:index], c.data.Proxies[index+1:]...)
	c.stopServerIfUnused(port)
}

// proxyIndex returns where the proxy with id is in Proxies, or -1. The
// caller must hold the mutex.
func (c *Core) proxyIndex(id string) int {
	for index, proxy := range c.data.Proxies {
		if proxy.ID == id {
			return index
		}
	}
	return -1
}

// activeProxy returns where port's proxy that's on is in proxies, or -1 if
// none is.
func activeProxy(proxies []Proxy, port int) int {
	for index, proxy := range proxies {
		if proxy.Port == port && !proxy.Disabled {
			return index
		}
	}
	return -1
}

// portProxy returns where the proxy shown for port is in proxies, for
// clients that only know one proxy per port: the one that's on, or else the
// last one added. It returns -1 if the port has none.
func portProxy(proxies []Proxy, port int) int {
	shown := activeProxy(proxies, port)
	if shown >= 0 {
		return shown
	}
	for index, proxy := range proxies {
		if proxy.Port == port {
			shown = index
		}
	}
	return shown
}

// proxyPorts returns the ports that have a proxy, in order.
func proxyPorts(proxies []Proxy) []int {
	var ports []int
	for _, proxy := range proxies {
		ports = append(ports, proxy.Port)
	}
	return uniqueSorted(ports)
}

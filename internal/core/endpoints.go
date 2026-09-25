package core

// AddMock validates and adds an endpoint, saves it and starts listening on
// its port if nothing else does yet.
func (c *Core) AddMock(newMock NewMock) (Mock, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	err := c.validateMock(newMock)
	if err != nil {
		return Mock{}, err
	}

	// Start listening first: if the port is taken by another program, the
	// endpoint is refused rather than saved but unreachable.
	err = c.startServer(newMock.Port)
	if err != nil {
		return Mock{}, invalid("Could not listen on port %d: %v", newMock.Port, err)
	}

	mock := Mock{
		ID:     newID(),
		Port:   newMock.Port,
		Method: newMock.Method,
		Path:   newMock.Path,
		Status: newMock.Status,
		Body:   normalizeBody(newMock.Body),
	}
	c.data.Mocks = append(c.data.Mocks, mock)
	return mock, c.save()
}

// RemoveMock removes an endpoint and stops listening on its port if nothing
// else uses it. It returns false if there was no endpoint with that ID.
func (c *Core) RemoveMock(id string) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	kept := []Mock{}
	found := false
	removedPort := 0
	for _, mock := range c.data.Mocks {
		if mock.ID == id {
			found = true
			removedPort = mock.Port
			continue
		}
		kept = append(kept, mock)
	}
	if !found {
		return false, nil
	}

	c.data.Mocks = kept
	c.stopServerIfUnused(removedPort)
	return true, c.save()
}

// SetMockEnabled turns one endpoint on or off. It returns false if there was
// no endpoint with that ID.
func (c *Core) SetMockEnabled(id string, enabled bool) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for index := range c.data.Mocks {
		if c.data.Mocks[index].ID == id {
			c.data.Mocks[index].Disabled = !enabled
			return true, c.save()
		}
	}
	return false, nil
}

// SetEnabled turns all endpoints on or off.
func (c *Core) SetEnabled(enabled bool) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data.Enabled = enabled
	return c.save()
}

// SetUpstream forwards unmatched requests on fapiPort to upstreamPort on
// host: a host name or IP address, starting with https:// if the real API
// uses HTTPS, or "" for passThrough.host in fapi's settings. An upstreamPort
// of 0 removes the pass-through.
func (c *Core) SetUpstream(fapiPort int, upstreamPort int, host string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if upstreamPort == 0 {
		delete(c.data.Upstreams, fapiPort)
		delete(c.data.UpstreamHosts, fapiPort)
		delete(c.data.DisabledUpstreams, fapiPort)
		c.stopServerIfUnused(fapiPort)
		return c.save()
	}

	host, err := normalizeUpstreamHost(host)
	if err != nil {
		return err
	}
	err = c.validateUpstream(fapiPort, upstreamPort, host)
	if err != nil {
		return err
	}

	err = c.startServer(fapiPort)
	if err != nil {
		return invalid("Could not listen on port %d: %v", fapiPort, err)
	}

	// Adding a proxy turns it on, even if it replaces one that was off.
	c.data.Upstreams[fapiPort] = upstreamPort
	delete(c.data.DisabledUpstreams, fapiPort)
	if host == "" {
		delete(c.data.UpstreamHosts, fapiPort)
	} else {
		c.data.UpstreamHosts[fapiPort] = host
	}
	return c.save()
}

// SetUpstreamEnabled turns the proxy on fapiPort on or off. It returns false
// if that port has no proxy.
func (c *Core) SetUpstreamEnabled(fapiPort int, enabled bool) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, found := c.data.Upstreams[fapiPort]
	if !found {
		return false, nil
	}
	if enabled {
		delete(c.data.DisabledUpstreams, fapiPort)
	} else {
		c.data.DisabledUpstreams[fapiPort] = true
	}
	return true, c.save()
}

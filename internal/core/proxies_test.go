package core

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"fapi/internal/config"
)

// newTestCore returns a core with its data in a new folder, and a free port
// for it to proxy on.
func newTestCore(t *testing.T) (*Core, int, string) {
	t.Helper()
	settings, err := config.Defaults()
	if err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	core, err := New(settings, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(core.Stop)
	return core, freePort(t), dataDir
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// onFor returns the real API port the core forwards port's requests to, or 0.
func onFor(core *Core, port int) int {
	_, target := core.route(port, "GET", "/")
	return target.port
}

func TestOneProxyOnPerPort(t *testing.T) {
	core, port, _ := newTestCore(t)

	first, err := core.AddProxy(port, 3002, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := core.AddProxy(port, 3003, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := onFor(core, port); got != 3003 {
		t.Fatalf("after adding a second proxy, forwarding to %d, want the new one (3003)", got)
	}
	if proxies := core.Snapshot().Proxies; len(proxies) != 2 || !proxies[0].Disabled || proxies[1].Disabled {
		t.Fatalf("proxies = %+v, want the first off and the second on", proxies)
	}

	// Turning the first back on turns the second off.
	if _, err := core.SetProxyEnabled(first.ID, true); err != nil {
		t.Fatal(err)
	}
	if got := onFor(core, port); got != 3002 {
		t.Fatalf("after turning the first on, forwarding to %d, want 3002", got)
	}
	if core.Snapshot().Proxies[1].Disabled != true {
		t.Error("the second proxy stayed on")
	}

	// Turning it off leaves none on.
	if _, err := core.SetProxyEnabled(first.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := onFor(core, port); got != 0 {
		t.Errorf("with both off, forwarding to %d", got)
	}

	// Adding one that exists turns it on rather than adding another.
	again, err := core.AddProxy(port, 3003, "")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != second.ID || len(core.Snapshot().Proxies) != 2 || onFor(core, port) != 3003 {
		t.Errorf("adding an existing proxy: got %+v, %d proxies", again, len(core.Snapshot().Proxies))
	}
}

func TestProxiesOnOtherPortsUnaffected(t *testing.T) {
	core, port, _ := newTestCore(t)
	other := freePort(t)

	if _, err := core.AddProxy(other, 4000, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := core.AddProxy(port, 3002, ""); err != nil {
		t.Fatal(err)
	}
	if onFor(core, other) != 4000 {
		t.Error("a proxy on another port was turned off")
	}
}

func TestRemoveProxy(t *testing.T) {
	core, port, _ := newTestCore(t)
	first, _ := core.AddProxy(port, 3002, "")
	second, _ := core.AddProxy(port, 3003, "")

	if found, err := core.RemoveProxy(second.ID); !found || err != nil {
		t.Fatalf("RemoveProxy = %v, %v", found, err)
	}
	if onFor(core, port) != 0 {
		t.Error("removing the proxy that was on turned another on")
	}
	if _, running := core.servers[port]; !running {
		t.Error("stopped listening while the port still has a proxy")
	}
	if found, _ := core.RemoveProxy(first.ID); !found {
		t.Fatal("couldn't remove the last proxy")
	}
	if _, running := core.servers[port]; running {
		t.Error("still listening on a port with no proxies or endpoints")
	}
}

// The older, one-per-port API, which the terminal UI uses.
func TestSetUpstream(t *testing.T) {
	core, port, _ := newTestCore(t)

	if err := core.SetUpstream(port, 3002, ""); err != nil {
		t.Fatal(err)
	}
	if err := core.SetUpstream(port, 3003, ""); err != nil {
		t.Fatal(err)
	}
	upstreams, _, disabled := ProxiesByPort(core.Snapshot().Proxies)
	if upstreams[port] != 3003 || disabled[port] {
		t.Fatalf("shown proxy: %d (off %v), want 3003 on", upstreams[port], disabled[port])
	}

	if _, err := core.SetUpstreamEnabled(port, false); err != nil {
		t.Fatal(err)
	}
	upstreams, _, disabled = ProxiesByPort(core.Snapshot().Proxies)
	if upstreams[port] != 3003 || !disabled[port] {
		t.Errorf("after turning it off, shown proxy: %d (off %v), want 3003 off", upstreams[port], disabled[port])
	}

	// Removing takes the shown proxy only.
	if err := core.SetUpstream(port, 0, ""); err != nil {
		t.Fatal(err)
	}
	proxies := core.Snapshot().Proxies
	if len(proxies) != 1 || proxies[0].UpstreamPort != 3002 {
		t.Errorf("after removing, proxies = %+v, want only 3002", proxies)
	}
}

func TestLoadOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	old := `{"enabled": true, "mocks": [], "payloads": [],
		"upstreams": {"3001": 3000, "4001": 443},
		"upstreamHosts": {"4001": "https://api.example.com"},
		"disabledUpstreams": {"4001": true}}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := loadData(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Proxies) != 2 {
		t.Fatalf("proxies = %+v, want 2", data.Proxies)
	}
	first, second := data.Proxies[0], data.Proxies[1]
	if first.Port != 3001 || first.UpstreamPort != 3000 || first.Host != "" || first.Disabled || first.ID == "" {
		t.Errorf("first = %+v", first)
	}
	if second.Port != 4001 || second.UpstreamPort != 443 || second.Host != "https://api.example.com" || !second.Disabled {
		t.Errorf("second = %+v", second)
	}
}

// Files keep the older fields too, so an older fapi can still read them.
func TestSaveKeepsOldFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mocks.json")
	data := Data{Enabled: true, Mocks: []Mock{}, Payloads: []Payload{}, Proxies: []Proxy{
		{ID: "a", Port: 3001, UpstreamPort: 3002, Disabled: true},
		{ID: "b", Port: 3001, UpstreamPort: 3003, Host: "https://api.example.com"},
	}}
	if err := saveData(path, data); err != nil {
		t.Fatal(err)
	}

	contents, _ := os.ReadFile(path)
	var saved struct {
		Upstreams     map[int]int    `json:"upstreams"`
		UpstreamHosts map[int]string `json:"upstreamHosts"`
		Proxies       []Proxy        `json:"proxies"`
	}
	if err := json.Unmarshal(contents, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Upstreams[3001] != 3003 || saved.UpstreamHosts[3001] != "https://api.example.com" || len(saved.Proxies) != 2 {
		t.Errorf("saved = %+v", saved)
	}

	// Read back, "proxies" wins over the older fields.
	loaded, err := loadData(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Proxies) != 2 || loaded.Proxies[0].ID != "a" {
		t.Errorf("loaded = %+v", loaded.Proxies)
	}
}

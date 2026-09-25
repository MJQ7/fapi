package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"fapi/internal/files"
)

// Ports are the settings the web UI's Settings screen can change: the admin
// port, the range of ports endpoints and proxies may use, and the port the
// UIs suggest for a new one.
type Ports struct {
	AdminPort int       `json:"adminPort"`
	MockPorts PortRange `json:"mockPorts"`
}

// Ports returns the port settings.
func (c Config) Ports() Ports {
	return Ports{AdminPort: c.AdminPort, MockPorts: c.MockPorts}
}

// SettingError is a setting with a value that isn't allowed. Its message
// names the setting and is worded to be shown to the user as-is.
type SettingError struct {
	Message string
}

// Error makes SettingError an error.
func (e *SettingError) Error() string {
	return e.Message
}

// checkPorts checks the port settings. It's used both when fapi reads a
// settings file and when a UI saves new ports.
func checkPorts(ports Ports) error {
	portRange := ports.MockPorts

	switch {
	case !isPort(ports.AdminPort):
		return &SettingError{Message: fmt.Sprintf("The admin port (adminPort) must be between 1 and 65535, not %d", ports.AdminPort)}
	case !isPort(portRange.Min):
		return &SettingError{Message: fmt.Sprintf("The lowest mock port (mockPorts.min) must be between 1 and 65535, not %d", portRange.Min)}
	case !isPort(portRange.Max):
		return &SettingError{Message: fmt.Sprintf("The highest mock port (mockPorts.max) must be between 1 and 65535, not %d", portRange.Max)}
	case portRange.Min > portRange.Max:
		return &SettingError{Message: fmt.Sprintf("The lowest mock port (%d) must not be higher than the highest (%d)", portRange.Min, portRange.Max)}
	case portRange.Default < portRange.Min || portRange.Default > portRange.Max:
		return &SettingError{Message: fmt.Sprintf("The default mock port (mockPorts.default) must be between %d and %d, not %d", portRange.Min, portRange.Max, portRange.Default)}
	case portRange.Default == ports.AdminPort:
		return &SettingError{Message: fmt.Sprintf("The default mock port can't be the admin port (%d)", ports.AdminPort)}
	}
	return nil
}

// defaultMockPort picks the default mock port for a settings file that
// doesn't choose one: the built-in default, or, if the file's port range or
// admin port rules that out, the lowest allowed port. Without this, files
// written before the setting existed, such as one allowing only ports 4000
// to 4009, would stop fapi from starting.
func defaultMockPort(settings Config, builtIn int) int {
	portRange := settings.MockPorts
	if builtIn >= portRange.Min && builtIn <= portRange.Max && builtIn != settings.AdminPort {
		return builtIn
	}
	if portRange.Min == settings.AdminPort && portRange.Min < portRange.Max {
		return portRange.Min + 1
	}
	return portRange.Min
}

// SavePorts writes ports into the settings file at path, keeping every other
// setting in it, and creating the file if there isn't one. It returns the
// settings the file now holds. They take effect when fapi next starts.
//
// Errors caused by the values themselves are *SettingError.
func SavePorts(path string, ports Ports) (Config, error) {
	err := checkPorts(ports)
	if err != nil {
		return Config{}, err
	}

	// The file as a map from setting names to values, so settings this
	// function doesn't know about are written back unchanged.
	tree := map[string]any{}
	contents, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("reading settings file %s: %w", path, err)
	}
	if err == nil {
		err = json.Unmarshal(contents, &tree)
		if err != nil {
			return Config{}, fmt.Errorf("settings file %s isn't valid JSON, so it can't be updated: %w", path, err)
		}
	}

	tree["adminPort"] = ports.AdminPort
	tree["mockPorts"] = ports.MockPorts

	contents, err = json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return Config{}, fmt.Errorf("encoding settings: %w", err)
	}
	contents = append(contents, '\n')

	// Check the whole file as fapi will read it at the next start, so saving
	// can never leave a file fapi refuses to start with.
	saved, err := applyOverride(contents)
	if err != nil {
		return Config{}, fmt.Errorf("settings file %s: %w", path, err)
	}

	err = files.WriteAtomic(path, contents)
	if err != nil {
		return Config{}, fmt.Errorf("saving settings: %w", err)
	}
	return saved, nil
}

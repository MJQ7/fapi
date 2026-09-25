// Package config reads fapi's settings: how fapi behaves, as opposed to the
// user's endpoints, which the core package saves in mocks.json.
//
// The built-in defaults live in defaults.json, which is compiled into the
// executable. An optional override file changes some of them. Settings are
// read once, at startup, and then passed to the rest of the program.
package config

import (
	"bytes"
	_ "embed" // for //go:embed
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// defaultsJSON is defaults.json, compiled into the executable.
//
//go:embed defaults.json
var defaultsJSON []byte

// Config holds every setting.
type Config struct {
	AdminPort     int                 `json:"adminPort"`
	ListenAddress string              `json:"listenAddress"`
	MockPorts     PortRange           `json:"mockPorts"`
	Features      Features            `json:"features"`
	RequestLog    RequestLogSettings  `json:"requestLog"`
	PassThrough   PassThroughSettings `json:"passThrough"`
}

// PortRange is the range of ports endpoints and proxies may use, and the
// port the UIs suggest for a new one.
type PortRange struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Default int `json:"default"`
}

// Features turns each optional feature on or off.
type Features struct {
	WebUI                 bool `json:"webUi"`
	PassThrough           bool `json:"passThrough"`
	CORS                  bool `json:"cors"`
	RequestLog            bool `json:"requestLog"`
	RequestLogPersistence bool `json:"requestLogPersistence"`
	LiveUpdates           bool `json:"liveUpdates"`
}

// RequestLogSettings limits how much the request log keeps.
type RequestLogSettings struct {
	MaxEntries   int `json:"maxEntries"`
	MaxBodyBytes int `json:"maxBodyBytes"`
}

// PassThroughSettings says where the real API runs.
type PassThroughSettings struct {
	Host string `json:"host"`
}

// Defaults returns the built-in settings from defaults.json.
func Defaults() (Config, error) {
	var settings Config
	err := decodeStrict(defaultsJSON, &settings)
	if err != nil {
		// Only possible if defaults.json itself was edited incorrectly.
		return Config{}, fmt.Errorf("reading built-in defaults.json: %w", err)
	}
	return settings, nil
}

// Load returns the effective settings: the built-in defaults, changed by the
// override file if there is one. It also returns the path of the override
// file it used, or "" when it used none.
//
// configFlag is the value of --config ("" when not given). dataDir is the
// data directory, where config.json is looked for last.
func Load(configFlag string, dataDir string) (Config, string, error) {
	settings, err := Defaults()
	if err != nil {
		return Config{}, "", err
	}

	path, err := findOverrideFile(configFlag, dataDir)
	if err != nil {
		return Config{}, "", err
	}
	if path == "" {
		return settings, "", nil
	}

	settings, err = LoadFile(path)
	if err != nil {
		return Config{}, "", err
	}
	return settings, path, nil
}

// LoadFile returns the built-in defaults changed by the settings file at
// path. A file that doesn't exist changes nothing.
func LoadFile(path string) (Config, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Defaults()
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading settings file %s: %w", path, err)
	}

	settings, err := applyOverride(contents)
	if err != nil {
		return Config{}, fmt.Errorf("settings file %s: %w", path, err)
	}
	return settings, nil
}

// applyOverride returns the built-in defaults changed by the JSON in
// contents, and checks the result.
func applyOverride(contents []byte) (Config, error) {
	settings, err := Defaults()
	if err != nil {
		return Config{}, err
	}

	// Clear the default mock port so that, after decoding, 0 means the file
	// didn't set it.
	builtInDefaultPort := settings.MockPorts.Default
	settings.MockPorts.Default = 0

	// Decoding into a struct that already holds the defaults only replaces
	// the fields present in the file, so the file only needs what it changes.
	err = decodeStrict(contents, &settings)
	if err != nil {
		return Config{}, err
	}

	if settings.MockPorts.Default == 0 {
		settings.MockPorts.Default = defaultMockPort(settings, builtInDefaultPort)
	}

	err = validate(settings)
	if err != nil {
		return Config{}, err
	}
	return settings, nil
}

// SettingsFile returns the file settings changed in a UI are saved to: the
// override file fapi started with (source, from Load), or config.json in the
// data folder when it started with none.
func SettingsFile(source string, dataDir string) string {
	if source != "" {
		return source
	}
	return filepath.Join(dataDir, "config.json")
}

// findOverrideFile picks the override file, in this order: --config, the
// FAPI_CONFIG environment variable, config.json in the data directory.
func findOverrideFile(configFlag string, dataDir string) (string, error) {
	if configFlag != "" {
		return requireFile(configFlag, "--config")
	}

	fromEnvironment := os.Getenv("FAPI_CONFIG")
	if fromEnvironment != "" {
		return requireFile(fromEnvironment, "FAPI_CONFIG")
	}

	// config.json in the data directory is optional: no file means defaults.
	inDataDir := filepath.Join(dataDir, "config.json")
	_, err := os.Stat(inDataDir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("checking for %s: %w", inDataDir, err)
	}
	return inDataDir, nil
}

// requireFile returns path if the file exists. A file the user named
// explicitly must exist, so a typo in the name isn't silently ignored.
func requireFile(path string, namedBy string) (string, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("settings file %s (from %s) does not exist", path, namedBy)
	}
	if err != nil {
		return "", fmt.Errorf("settings file %s (from %s): %w", path, namedBy, err)
	}
	return path, nil
}

// decodeStrict decodes JSON into target, rejecting settings that don't
// exist (such as the typo "cros") and values of the wrong type.
func decodeStrict(contents []byte, target *Config) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)
	if errors.Is(err, io.EOF) {
		return errors.New("the file is empty; it must contain a JSON object such as {}")
	}
	if err != nil {
		return explainDecodeError(err)
	}

	// Anything after the first JSON value is a mistake, such as two objects.
	if decoder.More() {
		return errors.New("unexpected text after the JSON object")
	}
	return nil
}

// explainDecodeError rewords encoding/json's errors so they name the setting.
func explainDecodeError(err error) error {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return fmt.Errorf("setting %q must be %s, not %s", typeError.Field, describeType(typeError.Type.Kind().String()), typeError.Value)
	}

	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return fmt.Errorf("not valid JSON (at byte %d): %s", syntaxError.Offset, syntaxError.Error())
	}

	// encoding/json reports unknown fields as: json: unknown field "cros"
	message := err.Error()
	if strings.HasPrefix(message, "json: unknown field ") {
		name := strings.TrimPrefix(message, "json: unknown field ")
		return fmt.Errorf("unknown setting %s", name)
	}
	return err
}

// describeType names a Go type the way the JSON settings file spells it.
// goType is the kind of Go type, such as "int" or "struct".
func describeType(goType string) string {
	switch goType {
	case "int":
		return "a whole number"
	case "bool":
		return "true or false"
	case "string":
		return "a string"
	case "struct":
		return "an object"
	default:
		return goType
	}
}

package config

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Differences lists the settings that differ from the built-in defaults, one
// line each, such as `features.cors: true → false`. fapi logs them at startup
// so it's clear what the override file changed.
func Differences(defaults Config, effective Config) ([]string, error) {
	defaultValues, err := flatten(defaults)
	if err != nil {
		return nil, err
	}
	effectiveValues, err := flatten(effective)
	if err != nil {
		return nil, err
	}

	var lines []string
	for name, value := range effectiveValues {
		if defaultValues[name] != value {
			lines = append(lines, fmt.Sprintf("%s: %s → %s", name, defaultValues[name], value))
		}
	}

	// Map order is random in Go, so sort to log in the same order every time.
	sort.Strings(lines)
	return lines, nil
}

// flatten turns settings into a map from dotted names, such as
// "features.cors", to their values written as JSON, such as "true".
// Going through JSON means the names match the settings file exactly.
func flatten(settings Config) (map[string]string, error) {
	contents, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("comparing settings: %w", err)
	}

	var tree map[string]any
	err = json.Unmarshal(contents, &tree)
	if err != nil {
		return nil, fmt.Errorf("comparing settings: %w", err)
	}

	values := map[string]string{}
	err = addValues(values, "", tree)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// addValues adds each value in tree to values, going into nested objects.
func addValues(values map[string]string, prefix string, tree map[string]any) error {
	for name, value := range tree {
		fullName := prefix + name

		// A "type assertion": nested is value as a map, and isObject says
		// whether value really was one.
		nested, isObject := value.(map[string]any)
		if isObject {
			err := addValues(values, fullName+".", nested)
			if err != nil {
				return err
			}
			continue
		}

		text, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("comparing setting %s: %w", fullName, err)
		}
		values[fullName] = string(text)
	}
	return nil
}

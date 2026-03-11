package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// ComponentSettings holds the configuration from component-settings.yaml
type ComponentSettings map[string]string

// LoadComponentSettings loads and parses the component-settings.yaml file
// from the repository root and returns the settings as a map.
func LoadComponentSettings() (ComponentSettings, error) {
	settingsPath := filepath.Join("..", "..", "component-settings.yaml")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read component-settings.yaml: %w", err)
	}

	var settings ComponentSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse component-settings.yaml: %w", err)
	}

	// Expand environment variables in settings values
	expandedSettings := make(ComponentSettings)
	for key, value := range settings {
		expandedSettings[key] = os.ExpandEnv(value)
	}

	return expandedSettings, nil
}

// GetOrDefault returns the value for the given key, or the default value if not found
func (cs ComponentSettings) GetOrDefault(key, defaultValue string) string {
	if value, ok := cs[key]; ok && value != "" {
		return value
	}
	return defaultValue
}

// SetEnvFromSettings sets environment variables from component settings
// This allows the settings to be used throughout the test suite
func SetEnvFromSettings(settings ComponentSettings) error {
	for key, value := range settings {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set env var %s: %w", key, err)
		}
	}
	return nil
}

// MustGetSetting returns the setting value or panics if not found
func (cs ComponentSettings) MustGetSetting(key string) string {
	value, ok := cs[key]
	if !ok || value == "" {
		panic(fmt.Sprintf("required setting %s not found in component-settings.yaml", key))
	}
	return value
}

// expandVariables expands ${VAR} style variables in a string
func expandVariables(s string, vars map[string]string) string {
	result := s
	for key, value := range vars {
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	return result
}

package worker

import (
	"encoding/json"
	"os"
)

type severityConfig struct {
	DefaultSeverity string            `json:"default_severity"`
	Linters         map[string]string `json:"linters"`
}

// LoadSeverityMap reads backend/config/severity-mapping.json (from M1),
// returning the linter->severity map and the fallback default severity.
func LoadSeverityMap(path string) (map[string]string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var cfg severityConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, "", err
	}
	return cfg.Linters, cfg.DefaultSeverity, nil
}

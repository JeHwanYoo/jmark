package measure

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

// HostReadings runs a trivial command and returns the collector readings this host reports.
func HostReadings() ([]string, error) {
	tmp, err := tempFile("jmark-readings-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	c := exec.Command("hyperfine", "-N", "--metrics", "all", "--runs", "1", "--style", "none", "--export-json", tmp, "true")
	if out, err := c.CombinedOutput(); err != nil {
		return nil, jerr("hyperfine readings check failed: %s", out)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return nil, err
	}
	return parseReadings(data)
}

// parseReadings keeps catalog order and drops readings missing from any measurement.
func parseReadings(data []byte) ([]string, error) {
	var export struct {
		Results []struct {
			Measurements []map[string]json.RawMessage `json:"measurements"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, err
	}
	if len(export.Results) == 0 || len(export.Results[0].Measurements) == 0 {
		return nil, jerr("hyperfine reported no measurements")
	}
	out := []string{}
	for _, r := range benchmark.Readings {
		reported := true
		for _, m := range export.Results[0].Measurements {
			var reading struct {
				Value *float64 `json:"value"`
			}
			if raw, ok := m[r.ID]; !ok || json.Unmarshal(raw, &reading) != nil || reading.Value == nil {
				reported = false
			}
		}
		if reported {
			out = append(out, r.ID)
		}
	}
	return out, nil
}

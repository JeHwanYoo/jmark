package measure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type hfExport struct {
	Results []hfResult `json:"results"`
}

type hfResult struct {
	Median     json.RawMessage `json:"median"`
	Summary    *hfSummary      `json:"summary"`
	Parameters hfParameters    `json:"parameters"`
}

type hfSummary struct {
	Wall *hfStat `json:"time_wall_clock"`
}

type hfStat struct {
	Median json.RawMessage `json:"median"`
}

type hfParameters struct {
	Iters json.RawMessage `json:"iters"`
}

func parseHyperfine(data []byte, lo, hi int) (float64, float64, error) {
	var exp hfExport
	if err := json.Unmarshal(data, &exp); err != nil {
		return 0, 0, fmt.Errorf("hyperfine export: %w", err)
	}
	loKey := strconv.Itoa(lo)
	hiKey := strconv.Itoa(hi)
	var loVals, hiVals []float64
	for _, r := range exp.Results {
		it, ok := itersString(r.Parameters.Iters)
		if !ok || (it != loKey && it != hiKey) {
			continue
		}
		med, err := pickMedian(r)
		if err != nil {
			return 0, 0, err
		}
		if it == loKey {
			loVals = append(loVals, med)
		}
		if it == hiKey {
			hiVals = append(hiVals, med)
		}
	}
	if len(loVals) != 1 || len(hiVals) != 1 {
		return 0, 0, fmt.Errorf("hyperfine export missing iters=%d or iters=%d", lo, hi)
	}
	return loVals[0], hiVals[0], nil
}

func pickMedian(r hfResult) (float64, error) {
	raw := medianRaw(r.Summary)
	if !present(raw) {
		raw = r.Median
	}
	if !present(raw) {
		return 0, errors.New("hyperfine export has no wall-clock median")
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, errors.New("hyperfine export has no wall-clock median")
	}
	return f, nil
}

func medianRaw(s *hfSummary) json.RawMessage {
	if s == nil || s.Wall == nil {
		return nil
	}
	return s.Wall.Median
}

func present(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) > 0 && !bytes.Equal(raw, []byte("null"))
}

func itersString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", false
	}
	if raw[0] == '{' {
		var obj struct {
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", false
		}
		return scalarString(obj.Value)
	}
	return scalarString(raw)
}

func scalarString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", false
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", false
		}
		return s, true
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var n json.Number
	if err := dec.Decode(&n); err != nil {
		return "", false
	}
	return n.String(), true
}

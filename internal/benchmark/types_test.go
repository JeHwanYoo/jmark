package benchmark

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestMetricReadingsAreCollectorReadings(t *testing.T) {
	for id, d := range Metrics {
		if len(d.Readings) == 0 {
			t.Fatalf("%s has no readings", id)
		}
		for _, r := range d.Readings {
			if !ReadingKnown(r) {
				t.Fatalf("%s reads unknown %s", id, r)
			}
		}
		if d.Method == "ratio" && len(d.Readings) != 2 {
			t.Fatalf("%s ratio needs numerator and denominator", id)
		}
		if d.Source != "" && d.Source != d.Readings[0] {
			t.Fatalf("%s source %s differs from readings %v", id, d.Source, d.Readings)
		}
	}
}

func TestMetricSupportFollowsReadings(t *testing.T) {
	var darwin []string
	for id := range Metrics {
		if MetricSupported(id, "darwin") {
			darwin = append(darwin, id)
		}
	}
	sort.Strings(darwin)
	want := []string{"cpu_cycles", "cpu_time", "instructions", "ipc", "peak_rss", "system_time", "throughput", "user_time", "wall_time"}
	if !reflect.DeepEqual(darwin, want) {
		t.Fatalf("darwin metrics %v, want %v", darwin, want)
	}
	for id := range Metrics {
		if !MetricSupported(id, "linux") {
			t.Fatalf("%s unsupported on linux", id)
		}
	}
	if MetricSupported("peak_rss", "windows") || !MetricSupported("wall_time", "windows") {
		t.Fatal("windows support")
	}
	if MetricSupported("probe.custom", "linux") {
		t.Fatal("probe metrics are not collector metrics")
	}
}

// The reading table in METRICS.md must describe the same catalog.
func TestMetricsDocumentMatchesReadings(t *testing.T) {
	doc, err := os.ReadFile("../../METRICS.md")
	if err != nil {
		t.Fatal(err)
	}
	platforms := []string{"darwin", "linux", "windows"}
	seen := 0
	for _, line := range strings.Split(string(doc), "\n") {
		cells := strings.Split(strings.Trim(line, "| "), " | ")
		if len(cells) != 5 || !strings.HasPrefix(cells[0], "`") {
			continue
		}
		id := strings.Trim(cells[0], "`")
		r := Readings[seen]
		if r.ID != id {
			t.Fatalf("row %d documents %s, want %s", seen, id, r.ID)
		}
		var used []string
		for _, name := range strings.Split(cells[1], ", ") {
			used = append(used, strings.Trim(name, "`"))
		}
		var users []string
		for metric, d := range Metrics {
			for _, reading := range d.Readings {
				if reading == id {
					users = append(users, metric)
				}
			}
		}
		sort.Strings(used)
		sort.Strings(users)
		if !reflect.DeepEqual(used, users) {
			t.Fatalf("%s used by %v, documented %v", id, users, used)
		}
		for i, p := range platforms {
			if (cells[2+i] == "✓") != ReadingSupported(id, p) {
				t.Fatalf("%s on %s documented as %s", id, p, cells[2+i])
			}
		}
		seen++
	}
	if seen != len(Readings) {
		t.Fatalf("documented %d readings, want %d", seen, len(Readings))
	}
}

package measure

import (
	"reflect"
	"testing"
)

func TestParseReadingsKeepsReportedValues(t *testing.T) {
	data := []byte(`{"results":[{"measurements":[
		{"time_wall_clock":{"value":0.1,"unit":"second"},"cpu_cycles":{"value":10},"cache_misses":{"value":null},"exit_code":0},
		{"time_wall_clock":{"value":0.1,"unit":"second"},"cpu_cycles":{"value":12},"exit_code":0}
	]}]}`)
	got, err := parseReadings(data)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"time_wall_clock", "cpu_cycles"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseReadingsRejectsEmptyExport(t *testing.T) {
	if _, err := parseReadings([]byte(`{"results":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}

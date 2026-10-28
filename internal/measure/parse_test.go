package measure

import (
	"math"
	"strings"
	"testing"
)

func TestObservationsV2(t *testing.T) {
	raw := []byte(`{"schema_version":2,"results":[{"parameters":{"iters":{"value":"2"}},"measurements":[{"time_wall_clock":{"value":1,"unit":"second"},"memory_peak_resident":{"value":100,"unit":"byte"}}]},{"parameters":{"iters":{"value":"10"}},"measurements":[{"time_wall_clock":{"value":3,"unit":"second"},"memory_peak_resident":{"value":200,"unit":"byte"}}]}]}`)
	got, err := parseObservations(raw, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got["time_wall_clock"].Workload[0] != 3 || got["memory_peak_resident"].Baseline[0] != 100 {
		t.Fatal(got)
	}
	for _, bad := range []string{`{"results":[]}`, `{"schema_version":2,"results":[]}`, strings.ReplaceAll(string(raw), `"value":3`, `"value":-3`)} {
		if _, err := parseObservations([]byte(bad), 2, 10); err == nil {
			t.Fatal("invalid export accepted")
		}
	}
}

func TestEncodeHost(t *testing.T) {
	with, err := encodeHost("Darwin", "arm64", "27", "Apple M4", "hyperfine 2.0.0", "14")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"os":"Darwin","arch":"arm64","kernel":"27","cpu":"Apple M4","hyperfine":"hyperfine 2.0.0","ncpu":14}`
	if string(with) != want {
		t.Fatalf("got %s", with)
	}
	without, err := encodeHost("Linux", "x86_64", "6", "", "hyperfine 1.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(without), "ncpu") {
		t.Fatalf("empty ncpu should be omitted: %s", without)
	}
	if !strings.Contains(string(without), `"cpu":""`) {
		t.Fatalf("empty cpu stays: %s", without)
	}
	if _, err := encodeHost("Linux", "x", "1", "", "hf", "nope"); err == nil {
		t.Fatal("non-numeric ncpu should fail")
	}
}

func TestCPUInfoModel(t *testing.T) {
	text := "processor\t: 0\nmodel name\t: Intel(R) Xeon\nmodel name\t: ignored\n"
	if got := cpuinfoModel(text); got != "Intel(R) Xeon" {
		t.Fatalf("got %q", got)
	}
	if got := cpuinfoModel("nothing\n"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestMedian(t *testing.T) {
	if _, err := median(nil); err == nil {
		t.Fatal("empty median should fail")
	}
	cases := []struct {
		in   []float64
		want float64
	}{
		{[]float64{1, 3, 100}, 3},
		{[]float64{100, 1, 3}, 3},
		{[]float64{1, 5}, 3},
		{[]float64{1, 2, 3, 4}, 2.5},
		{[]float64{4}, 4},
	}
	for _, tc := range cases {
		got, err := median(tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got-tc.want) > 0 {
			t.Fatalf("%v: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestSinksProportional(t *testing.T) {
	if !sinksProportional("10", "20", 1, 2) {
		t.Fatal("integer product")
	}
	if sinksProportional("9007199254740993", "18014398509481984", 1, 2) {
		t.Fatal("exact integer mismatch must fail even when float64 collapses it")
	}
	if !sinksProportional("-3", "-6", 1, 2) {
		t.Fatal("negative integers")
	}
	if !sinksProportional("0.5", "1.0", 1, 2) {
		t.Fatal("float ratio")
	}
	if sinksProportional("0.75", "1.25", 1, 2) {
		t.Fatal("float drift")
	}
	// "1" is an integer literal and "1.0" is not, so the float path runs.
	if !sinksProportional("1", "1.0", 1, 1) {
		t.Fatal("mixed 1 and 1.0 at equal iters")
	}
}

func TestSinkNumberConsumesEntireOutput(t *testing.T) {
	for _, output := range []string{"1]", "1}", "1 2", "NaN", "null", "[1]"} {
		if jsonNumber(output) {
			t.Errorf("accepted %q", output)
		}
	}
	for _, output := range []string{"1", "1.0", "-4e2", "9007199254740993"} {
		if !jsonNumber(output) {
			t.Errorf("rejected %q", output)
		}
	}
}

package measure

import (
	"math"
	"strings"
	"testing"
)

func TestParseHyperfineShapes(t *testing.T) {
	cases := []struct {
		name string
		json string
		lo   float64
		hi   float64
	}{
		{
			name: "v1 string iters and top-level median",
			json: `{"results":[
				{"median":1.5,"parameters":{"iters":"2"}},
				{"median":3.25,"parameters":{"iters":"10"}}
			]}`,
			lo: 1.5, hi: 3.25,
		},
		{
			name: "v2 object iters and wall-clock median",
			json: `{"results":[
				{"parameters":{"iters":{"value":"10"}},"summary":{"time_wall_clock":{"median":4}}},
				{"parameters":{"iters":{"value":"2"}},"summary":{"time_wall_clock":{"median":1.5}}}
			]}`,
			lo: 1.5, hi: 4,
		},
		{
			name: "numeric iters value",
			json: `{"results":[
				{"median":8,"parameters":{"iters":{"value":2}}},
				{"median":9,"parameters":{"iters":{"value":10}}}
			]}`,
			lo: 8, hi: 9,
		},
		{
			name: "summary median wins over top-level",
			json: `{"results":[
				{"median":9,"summary":{"time_wall_clock":{"median":1.5}},"parameters":{"iters":"2"}},
				{"median":9,"summary":{"time_wall_clock":{"median":4}},"parameters":{"iters":"10"}}
			]}`,
			lo: 1.5, hi: 4,
		},
		{
			name: "null summary median falls back",
			json: `{"results":[
				{"median":1.25,"summary":{"time_wall_clock":{"median":null}},"parameters":{"iters":"2"}},
				{"median":6,"parameters":{"iters":"10"}}
			]}`,
			lo: 1.25, hi: 6,
		},
		{
			name: "same lo and hi uses the one row twice",
			json: `{"results":[{"median":3,"parameters":{"iters":"5"}}]}`,
			lo:   3, hi: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi := 2, 10
			if tc.name == "same lo and hi uses the one row twice" {
				lo, hi = 5, 5
			}
			gotLo, gotHi, err := parseHyperfine([]byte(tc.json), lo, hi)
			if err != nil {
				t.Fatal(err)
			}
			if gotLo != tc.lo || gotHi != tc.hi {
				t.Fatalf("got %v %v, want %v %v", gotLo, gotHi, tc.lo, tc.hi)
			}
		})
	}
}

func TestParseHyperfineRejects(t *testing.T) {
	cases := []struct {
		name string
		json string
		want string
	}{
		{
			name: "no median",
			json: `{"results":[{"parameters":{"iters":"2"}},{"parameters":{"iters":"10"}}]}`,
			want: "no wall-clock median",
		},
		{
			name: "median is a string",
			json: `{"results":[
				{"summary":{"time_wall_clock":{"median":"slow"}},"parameters":{"iters":"2"}},
				{"median":1,"parameters":{"iters":"10"}}
			]}`,
			want: "no wall-clock median",
		},
		{
			name: "missing hi",
			json: `{"results":[{"median":1,"parameters":{"iters":"2"}}]}`,
			want: "missing iters=2 or iters=10",
		},
		{
			name: "duplicate lo",
			json: `{"results":[
				{"median":1,"parameters":{"iters":"2"}},
				{"median":2,"parameters":{"iters":"2"}},
				{"median":3,"parameters":{"iters":"10"}}
			]}`,
			want: "missing iters=2 or iters=10",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseHyperfine([]byte(tc.json), 2, 10)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err %v, want substring %q", err, tc.want)
			}
		})
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

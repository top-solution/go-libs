package frequency

import (
	"fmt"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

var timeLayout = "2006-01-02T15:04:05.000"

func TestParseFrequency(t *testing.T) {
	var tests = []struct {
		s       string
		want    Frequency
		wantErr bool
	}{
		{s: `100s`, want: Frequency{duration: 100 * time.Second, unit: "s"}},
		{s: `2m`, want: Frequency{duration: 2 * time.Minute, unit: "m"}},
		{s: `2mo`, want: Frequency{months: 2, unit: "mo"}},
		{s: `2h`, want: Frequency{duration: 2 * time.Hour, unit: "h"}},
		{s: `2d`, want: Frequency{days: 2, unit: "d"}},
		{s: `15d`, want: Frequency{days: 15, unit: "d"}},
		{s: `2w`, want: Frequency{weeks: 2, unit: "w"}},
		{s: `2y`, want: Frequency{years: 2, unit: "y"}},
		{s: `-5s`, want: Frequency{duration: -5 * time.Second, unit: "s"}},
		{s: `25h`, wantErr: true}, // after 24h, the minimum resolution becomes 1 day
		{s: `1h30m`, wantErr: true},
		{s: `-5m30s`, wantErr: true},
		{s: `3mm`, wantErr: true},
		{s: ``, wantErr: true},
		{s: `0s`, wantErr: true},
		{s: `3`, wantErr: true},
		{s: `3nm`, wantErr: true},
		{s: `1000`, wantErr: true},
		{s: `w`, wantErr: true},
		{s: `ms`, wantErr: true},
		{s: `1.2w`, wantErr: true},
		{s: `10x`, wantErr: true},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: parse %s", i, tt.s), func(t *testing.T) {
			got, err := ParseFrequency(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatFrequency(t *testing.T) {
	var tests = []struct {
		f    Frequency
		want string
	}{
		{f: Frequency{duration: 100 * time.Second, unit: "s"}, want: `100s`},
		{f: Frequency{duration: 2 * time.Minute, unit: "m"}, want: `2m`},
		{f: Frequency{duration: 2 * time.Hour, unit: "h"}, want: `2h`},
		{f: Frequency{days: 2, unit: "d"}, want: `2d`},
		{f: Frequency{weeks: 3, unit: "w"}, want: `3w`},
		{f: Frequency{months: 1, unit: "mo"}, want: `1mo`},
		{f: Frequency{years: 4, unit: "y"}, want: `4y`},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: want %s", i, tt.want), func(t *testing.T) {
			got := tt.f.String()
			if got != tt.want {
				t.Errorf("Format() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIdempotency(t *testing.T) {
	var tests = []struct {
		s string
	}{
		{s: "1s"},
		{s: "5m"},
		{s: "4h"},
		{s: "3d"},
		{s: "1w"},
		{s: "7mo"},
		{s: "2y"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: %s", i, tt.s), func(t *testing.T) {
			freq, err := ParseFrequency(tt.s)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			got := freq.String()
			if got != tt.s {
				t.Errorf("Format() = %s, want %s", got, tt.s)
			}
		})
	}
}

func TestNextRun(t *testing.T) {
	var tests = []struct {
		f             Frequency
		wantString    string
		lastRunString string
	}{
		{f: Frequency{duration: 15 * time.Millisecond, unit: "ms"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-11-26T15:00:05.365"},
		{f: Frequency{duration: 100 * time.Second, unit: "s"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-11-26T15:01:45.350"},
		{f: Frequency{duration: 2 * time.Minute, unit: "m"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-11-26T15:02:05.350"},
		{f: Frequency{duration: 2 * time.Hour, unit: "h"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-11-26T17:00:05.350"},
		{f: Frequency{days: 2, unit: "d"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-11-28T15:00:05.350"},
		{f: Frequency{weeks: 3, unit: "w"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-12-17T15:00:05.350"},
		{f: Frequency{months: 1, unit: "mo"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-12-26T15:00:05.350"},
		{f: Frequency{years: 4, unit: "y"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2025-11-26T15:00:05.350"},
		{f: Frequency{days: 2, unit: "d"}, lastRunString: "2021-11-26T15:00:05.350", wantString: "2021-11-28T15:00:05.350"},
		// Adding a full month while near the end of a non-30-days one can fall to the next
		{f: Frequency{months: 1, unit: "mo"}, lastRunString: "2021-01-31T15:00:05.350", wantString: "2021-03-03T15:00:05.350"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: %s", i, tt.f), func(t *testing.T) {
			lastRun, _ := time.Parse(timeLayout, tt.lastRunString)
			got := tt.f.NextRun(lastRun).Format(timeLayout)
			if got != tt.wantString {
				t.Errorf("NextRun() = %s, want %s", got, tt.wantString)
			}
		})
	}
}

func TestShouldRun(t *testing.T) {
	var tests = []struct {
		f                 Frequency
		currentTimeString string
		lastRunString     string
		should            bool
	}{
		{f: Frequency{duration: 15 * time.Millisecond, unit: "ms"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-26T15:00:05.565", should: true},
		{f: Frequency{duration: 100 * time.Second, unit: "s"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-26T15:01:50.350", should: true},
		{f: Frequency{duration: 2 * time.Minute, unit: "m"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-26T15:08:05.350", should: true},
		{f: Frequency{duration: 2 * time.Hour, unit: "h"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-26T19:00:05.350", should: true},
		{f: Frequency{days: 2, unit: "d"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-28T15:00:05.360", should: true},
		{f: Frequency{days: 2, unit: "d"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-28T14:00:05.350", should: false},
		{f: Frequency{weeks: 3, unit: "w"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-12-17T15:00:05.350", should: false},
		{f: Frequency{months: 1, unit: "mo"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-12-26T15:00:05.350", should: false},
		{f: Frequency{years: 4, unit: "y"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2025-11-26T15:00:05.350", should: false},
		{f: Frequency{days: 2, unit: "d"}, lastRunString: "2021-11-26T15:00:05.350", currentTimeString: "2021-11-28T15:00:05.350", should: false},
		{f: Frequency{months: 1, unit: "mo"}, lastRunString: "2021-01-31T15:00:05.350", currentTimeString: "2021-03-03T15:00:05.340", should: false},
		{f: Frequency{months: 1, unit: "mo"}, lastRunString: "2021-01-31T15:00:05.350", currentTimeString: "2021-03-03T15:00:05.360", should: true},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: %s", i, tt.f), func(t *testing.T) {
			lastRun, _ := time.Parse(timeLayout, tt.lastRunString)
			currentTime, _ := time.Parse(timeLayout, tt.currentTimeString)
			got := tt.f.ShouldRun(lastRun, currentTime)
			if got != tt.should {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.should)
			}
		})
	}
}

func TestFromDuration(t *testing.T) {
	var tests = []struct {
		d    time.Duration
		want Frequency
	}{
		{d: 100 * time.Second, want: Frequency{duration: 100 * time.Second, unit: "s"}},
		{d: 2 * time.Minute, want: Frequency{duration: 2 * time.Minute, unit: "m"}},
		{d: 2*time.Minute + 30*time.Second, want: Frequency{duration: 150 * time.Second, unit: "s"}},
		{d: 2 * time.Hour, want: Frequency{duration: 2 * time.Hour, unit: "h"}},
		{d: 25 * time.Hour, want: Frequency{duration: 25 * time.Hour, unit: "h"}},
		{d: 24 * time.Hour * 7, want: Frequency{weeks: 1, unit: "w"}},
		{d: 24 * time.Hour * 8, want: Frequency{days: 8, unit: "d"}},
		{d: 25 * time.Millisecond, want: Frequency{duration: 1 * time.Second, unit: "s"}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d: parse %s", i, tt.d), func(t *testing.T) {
			got := FromDuration(tt.d)

			if got != tt.want {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

type tstStruct struct {
	Test struct {
		Frequency Frequency `yaml:"frequency" json:"frequency"`
	} `yaml:"test" json:"test"`
}

func TestUnmarshalYAML(t *testing.T) {
	var tests = []struct {
		yml     string
		want    Frequency
		wantErr bool
	}{
		{yml: `
test:
  frequency: 5s`, want: Frequency{duration: 5 * time.Second, unit: "s"}},
		{yml: `
test:
  frequency: 5`, wantErr: true},
		{yml: `
test:
  frequency:`},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := tstStruct{}
			err := yaml.Unmarshal([]byte(tt.yml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Test.Frequency != tt.want {
				t.Errorf("UnmarshalYAML() = %s, want %s", got.Test.Frequency, tt.want)
			}
		})
	}
}

package frequency

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

var ErrInvalidFrequency = errors.New("invalid duration")

var NilFrequency = Frequency{}

type Frequency struct {
	duration time.Duration
	days     int
	weeks    int
	months   int
	years    int
	unit     string
}

// ParseFrequency parses a frequency from a string. It only accepts 1 integer value and 1 unit
// Valid units are:
// - s: seconds
// - m: minutes
// - h: hours
// - d: days
// - w: weeks
// - mo: months
// - y: years
//
// The total seconds, minutes, and hours must equal less than a full day: after that, you must use days, weeks, months or years
// This func will also be used to Unmarshal from YAML/JSON
func ParseFrequency(s string) (d Frequency, err error) {
	if len(s) < 2 {
		return NilFrequency, ErrInvalidFrequency
	}

	// Split string into individual chars
	a := split(s)

	i := 0

	// Support negative values
	isNegative := false
	if a[i] == '-' {
		isNegative = true
		i++
	}

	// Find the number portion
	start := i
	for ; i < len(a) && isDigit(a[i]); i++ {
	}

	if i >= len(a) || i == start {
		return NilFrequency, ErrInvalidFrequency
	}

	// Parse it
	n, err := strconv.ParseInt(string(a[start:i]), 10, 64)
	if err != nil {
		return NilFrequency, ErrInvalidFrequency
	}

	// Extract the unit
	d.unit = string(a[i])
unit:
	switch a[i] {
	case 'm':
		if i+1 < len(a) {
			switch a[i+1] {
			case 'o': // mo == month
				d.unit = string(a[i : i+2])
				d.months = int(n)
				i++
				break unit
			}
		}
		d.duration = time.Duration(n) * time.Minute
	case 's':
		d.duration = time.Duration(n) * time.Second
	case 'h':
		d.duration = time.Duration(n) * time.Hour
	case 'd':
		d.days = int(n)
	case 'w':
		d.weeks = int(n)
	case 'y':
		d.years = int(n)

	default:
		return NilFrequency, ErrInvalidFrequency
	}
	if i+1 < len(a) {
		return NilFrequency, ErrInvalidFrequency
	}

	if d.duration >= time.Hour*24 {
		return NilFrequency, ErrInvalidFrequency
	}

	if isNegative {
		d.duration = -d.duration
		d.days = -d.days
		d.months = -d.months
		d.years = -d.years
	}
	if d.IsZero() {
		return NilFrequency, ErrInvalidFrequency
	}

	return d, nil
}

// FromDuration returns a frequency from time.Duration, rounding to the nearest unit
func FromDuration(d time.Duration) (Frequency, error) {
	if d < time.Second {
		return NilFrequency, ErrInvalidFrequency
	}
	if (d/time.Second)%(3600*24*365) == 0 {
		return Frequency{years: int((d / time.Second) / (3600 * 24 * 365)), unit: "y"}, nil
	}
	if (d/time.Second)%(3600*24*30) == 0 {
		return Frequency{months: int((d / time.Second) / (3600 * 24 * 30)), unit: "m"}, nil
	}
	if (d/time.Second)%(3600*24*7) == 0 {
		return Frequency{weeks: int((d / time.Second) / (3600 * 24 * 7)), unit: "w"}, nil
	}
	if (d/time.Second)%(3600*24) == 0 {
		return Frequency{days: int((d / time.Second) / (3600 * 24)), unit: "d"}, nil
	}
	if (d/time.Second)%3600 == 0 {
		return Frequency{duration: d.Truncate(time.Hour), unit: "h"}, nil
	}
	if (d/time.Second)%60 == 0 {
		return Frequency{duration: d.Truncate(time.Minute), unit: "m"}, nil
	}
	return Frequency{duration: d.Truncate(time.Second), unit: "s"}, nil
}

// Unit returns the time unit
func (d Frequency) Unit() string {
	return d.unit
}

// Value returns the time value
func (d Frequency) Value() int {
	switch d.unit {
	case "ms":
		return int(d.duration.Milliseconds())
	case "s":
		return roundTime(d.duration.Seconds())
	case "m":
		return roundTime(d.duration.Minutes())
	case "h":
		return roundTime(d.duration.Hours())
	case "d":
		return d.days
	case "w":
		return d.weeks
	case "mo":
		return d.months
	case "y":
		return d.years
	}
	return 0
}

// String implements the std stringer interface
func (d Frequency) String() string {
	return fmt.Sprintf("%d%s", d.Value(), d.unit)
}

// IsZero implements the std zeroer interface
func (d Frequency) IsZero() bool {
	return d.duration == 0 && d.days == 0 && d.weeks == 0 && d.months == 0 && d.years == 0
}

// ShouldRun returns true if, given the time of the last run and the current time, the time is up
func (d Frequency) ShouldRun(lastRun, currentTime time.Time) bool {
	return d.NextRun(lastRun).After(currentTime)
}

// NextRun returns the time for the next run, given the time of the last
func (d Frequency) NextRun(lastRun time.Time) time.Time {
	return lastRun.Add(d.duration).AddDate(d.years, d.months, d.weeks*7+d.days)
}

// UnmarshalYAML Implements the Unmarshaler interface of the yaml pkg
func (f *Frequency) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return err
	}
	ff, err := ParseFrequency(buf)
	if err != nil {
		return err
	}

	*f = ff

	return nil
}

// UnmarshalJSON Implements the Unmarshaler interface of the json pkg
func (f *Frequency) UnmarshalJSON(data []byte) error {
	ff, err := ParseFrequency(string(data))
	if err != nil {
		return err
	}
	*f = ff

	return err
}

// split splits a string into a slice of runes
func split(s string) (a []rune) {
	for _, ch := range s {
		a = append(a, ch)
	}
	return
}

// isDigit returns true if the rune is a digit
func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

func roundTime(input float64) int {
	var result float64
	if input < 0 {
		result = math.Ceil(input - 0.5)
	} else {
		result = math.Floor(input + 0.5)
	}
	i, _ := math.Modf(result)
	return int(i)
}

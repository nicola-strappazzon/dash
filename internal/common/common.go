package common

import (
	"strconv"
	"strings"
)

func ParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func ParseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func ParseUptime(s string) float64 {
	s = strings.TrimSpace(s)

	units := []struct {
		suffix  string
		seconds float64
	}{
		{"micros", 1e-6},
		{"nanos", 1e-9},
		{"ms", 1e-3},
		{"d", 86400},
		{"h", 3600},
		{"m", 60},
		{"s", 1},
	}

	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			return ParseFloat(strings.TrimSuffix(s, u.suffix)) * u.seconds
		}
	}

	return ParseFloat(s)
}

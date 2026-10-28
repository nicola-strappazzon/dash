package mysql

import "testing"

func TestHealthCheckRawValue(t *testing.T) {
	zero, fraction := 0.0, 12.345
	for _, tt := range []struct {
		check HealthCheck
		want  any
	}{
		{HealthCheck{Value: &zero, Display: "0s"}, zero},
		{HealthCheck{Value: &fraction, Display: "12.35%"}, fraction},
		{HealthCheck{Display: "NULL"}, "NULL"},
		{HealthCheck{Display: "Yes"}, "Yes"},
	} {
		if got := tt.check.RawValue(); got != tt.want {
			t.Fatalf("got %v; want %v", got, tt.want)
		}
	}
}

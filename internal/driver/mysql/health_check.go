package mysql

type Status string

const (
	HealthOK       Status = "ok"
	HealthWarning  Status = "warning"
	HealthCritical Status = "critical"
)

type HealthCheck struct {
	Name    string
	Value   *float64
	Display string
	Status  Status
}

// RawValue returns a numeric measurement when available, including zero.
func (c HealthCheck) RawValue() any {
	if c.Value != nil {
		return *c.Value
	}
	return c.Display
}

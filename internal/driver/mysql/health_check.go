package mysql

type Status string

const (
	HealthOK       Status = "ok"
	HealthWarning  Status = "warning"
	HealthCritical Status = "critical"
)

type HealthCheck struct {
	Name    string
	Value   float64
	Display string
	Status  Status
}

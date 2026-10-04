package mysql

type HealthReport struct {
	Checks  []HealthCheck
	Version string
}

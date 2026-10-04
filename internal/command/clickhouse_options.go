package command

type ClickHouseOptions struct {
	Host        string
	Username    string
	Password    string
	TLS         bool
	InsecureTLS bool
	ClusterName string
}

package command

type MySQLOptions struct {
	Host         string
	Username     string
	Password     string
	Database     string
	Table        string
	TLS          bool
	InsecureTLS  bool
	VariableLike string
	StatusLike   string
	TablesLike   string
}

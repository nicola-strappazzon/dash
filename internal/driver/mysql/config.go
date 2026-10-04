package mysql

type Config struct {
	Host               string
	Username           string
	Password           string
	Database           string
	TLS                bool
	InsecureSkipVerify bool
}

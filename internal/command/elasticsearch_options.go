package command

type ElasticsearchOptions struct {
	Address            string
	Username           string
	Password           string
	InsecureTLS        bool
	SnapshotRepository string
}

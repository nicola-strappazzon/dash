# Dash

Dash is a terminal dashboard for inspecting ClickHouse, Elasticsearch, and
MySQL services.

## Run

```bash
go run . --help
```

When developing from source, Dash uses the local `../go-table` module declared
in `go.mod`.

## MySQL

```bash
# Instance health and replication state
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' health

# InnoDB activity and buffer pool information
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' innodb

# Databases and table allocation
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' databases
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' tables --database app --like '%_old'

# Index definition and available InnoDB index sizes
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' indexes --database app --table users

# Server settings and counters
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' variables --like 'innodb%'
go run . mysql -h 127.0.0.1 -u monitor -p '<password>' status --like 'Threads%'
```

Use `--watch 5s` to refresh a view and `--clear` to clear the terminal before
each refresh. MySQL index size is shown as `—` when the server does not expose
`mysql.innodb_index_stats` to the connected user.

## Development

```bash
gofmt -w path/to/changed.go
go test ./...
go vet ./...
git diff --check
```

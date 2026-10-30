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

## Output and monitoring

All service commands and subcommands share these flags:

- `--format table|json|csv` selects the output format (default: `table`).
- `--metric NAME[,NAME...]` selects metrics or columns, ignoring case and
  preserving the requested order. Without an explicit format it prints raw
  values without headers; multiple values share one comma-separated line.
- `--timestamp` prepends Unix time in seconds to CSV records.
- `--delta` requires a positive `--watch` interval and shows increments for
  known cumulative counters. Gauges and unclassified metrics remain unchanged.

```bash
dash mysql replication --metric "Replication lag" --format csv --timestamp --watch 5s
dash mysql status --metric Table_locks_immediate,Table_locks_waited --format csv --timestamp --watch 1s --delta
dash mysql status --metric Table_locks_immediate,Threads_connected --format csv --timestamp --watch 1s --delta
dash elasticsearch nodes --metric CPU --format csv --timestamp --watch 5s
dash clickhouse uptime --cluster-name example --metric UPTIME --format csv --timestamp
dash elasticsearch health --format json
```

CSV uses the table's column order, quotes fields containing commas or newlines,
uses `NULL` for unavailable values, and leaves numeric values unformatted.
Without `--metric`, CSV includes a header, repeated only when the schema changes.
With `--metric`, CSV omits the header. For tabular results it retains the first
column to identify each row; named metrics in key/value views produce
`value1,value2,...`, or `timestamp,value1,value2,...` when requested. `--clear` has no effect on CSV,
JSON, or metric-only output. Multiple views produce consecutive CSV blocks or
one JSON document per view. Diagnostic messages go to stderr.

Metric names are the displayed column names, field names in vertical views,
or row labels in key/value views (such as `Threads_connected`). Object results,
such as task details or allocation explanations, expose dotted paths, for
example `response.created` or `node_allocation_decisions.0.node_name`. Composite
InnoDB summaries expose separate numeric fields, including `Queries inside`,
`Queries queued`, and `Buffer pool reads/s`.

`--json` has been replaced by `--format json`. For compatibility, the original
`replication --metric "Replication lag" --timestamp` invocation still selects
CSV implicitly; new invocations should specify `--format csv` explicitly.

Gnuplot can read CSV timestamps with `set datafile separator ","`,
`set xdata time`, and `set timefmt "%s"`.

With `--delta`, the entire first execution is omitted while the baseline is
collected. The second sample reports `current - previous` for classified
counters. A decrease, unavailable value, new series, or reappearing series
establishes a new baseline: its delta is `-` in tables, `null` in JSON, and
`NULL` in CSV until a comparable sample exists. Deltas use exact integer
arithmetic and keep a separate baseline per view, row, and counter.

Classification currently covers documented cumulative MySQL `status` variables
(including `Table_locks_*`, command and handler counts, connections, bytes,
and common InnoDB counters), Elasticsearch thread pool rejection counts, and
SLM snapshot counters. MySQL gauges such as `Threads_connected`, replication
lag, and unknown variables retain their current values. ClickHouse's existing
views contain gauges and remain unchanged. Elasticsearch `stats --watch`
already reports rates; `--delta` does not subtract those rates again.

Counter classifications follow the [MySQL status variable reference](https://dev.mysql.com/doc/refman/8.4/en/server-status-variables.html),
[Elasticsearch thread pool statistics](https://www.elastic.co/docs/troubleshoot/elasticsearch/task-queue-backlog),
and [SLM statistics](https://www.elastic.co/docs/api/doc/elasticsearch/v8/operation/operation-slm-get-stats).

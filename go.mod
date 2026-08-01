module github.com/nicola-strappazzon/dash

go 1.26.5

require (
	github.com/elastic/go-elasticsearch/v8 v8.19.6
	github.com/fatih/color v1.19.0
	github.com/nicola-strappazzon/go-table v0.0.0-20260717080631-f327e1643619
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/elastic/elastic-transport-go/v8 v8.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

replace github.com/nicola-strappazzon/go-table => ../go-table

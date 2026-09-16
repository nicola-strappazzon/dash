package uptime

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/clickhouse/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/clickhouse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "uptime", "Show cluster uptime", Cluster(opts))
}

// Cluster returns a RenderFunc that reads the cluster name from opts at
// render time, once flags have been parsed, rather than capturing it at
// command construction time.
func Cluster(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, ch *clickhouse.ClickHouse) error {
		return Render(ctx, ch, opts.ClickHouse.ClusterName)
	}
}

func Render(ctx context.Context, ch *clickhouse.ClickHouse, cluster string) error {
	if cluster == "" {
		return fmt.Errorf("missing ClickHouse cluster name: pass --cluster-name")
	}

	uptime, err := ch.Uptime(ctx, cluster)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Cluster uptime")
	for _, u := range uptime {
		tbl.Add(u.Host, int64(u.UptimeSeconds), u.Version)
	}
	tbl.Column(0, table.Column{Name: "HOST"})
	tbl.Column(1, table.Column{
		Name:      "UPTIME",
		Format:    table.Duration,
		Alignment: table.Right,
		Width:     9,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "< 3600", Color: color.FgRed},
			{Condition: "< 86400", Color: color.FgYellow},
		},
	})
	tbl.Column(2, table.Column{Name: "VERSION"})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SortBy(0)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

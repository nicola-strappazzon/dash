package replica

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
	return runner.SectionCommand(ctx, opts, "replica", "Show replication health", Cluster(opts))
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

	replicas, err := ch.Replicas(ctx, cluster)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Replication health")
	for _, r := range replicas {
		tbl.Add(
			r.Host,
			int64(r.ReplicatedTables),
			int64(r.MaxDelay),
			int64(r.MaxQueue),
			r.MaxReplicationLag,
			int64(r.ReadonlyReplicas),
			int64(r.ExpiredSessions),
		)
	}
	tbl.Column(0, table.Column{Name: "HOST"})
	tbl.Column(1, table.Column{Name: "TABLES", Alignment: table.Right, Width: 6})
	tbl.Column(2, table.Column{
		Name:      "MAX_DELAY",
		Format:    table.Duration,
		Alignment: table.Right,
		Width:     9,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 300", Color: color.FgRed},
			{Condition: ">= 60", Color: color.FgYellow},
		},
	})
	tbl.Column(3, table.Column{
		Name:      "MAX_QUEUE",
		Alignment: table.Right,
		Width:     9,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	tbl.Column(4, table.Column{
		Name:      "MAX_LAG",
		Alignment: table.Right,
		Width:     7,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	tbl.Column(5, table.Column{
		Name:      "READONLY",
		Alignment: table.Right,
		Width:     8,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgRed},
		},
	})
	tbl.Column(6, table.Column{
		Name:      "EXPIRED",
		Alignment: table.Right,
		Width:     7,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgRed},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

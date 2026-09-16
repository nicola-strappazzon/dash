package disk

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
	return runner.SectionCommand(ctx, opts, "disk", "Show disk space usage", Cluster(opts))
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

	disks, err := ch.Disks(ctx, cluster)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Disk space")
	for _, d := range disks {
		tbl.Add(d.Host, int64(d.Free), int64(d.Total), d.UsedPercent)
	}
	tbl.Column(0, table.Column{Name: "HOST"})
	tbl.Column(1, table.Column{Name: "FREE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(2, table.Column{Name: "TOTAL", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(3, table.Column{
		Name:      "USED%",
		Format:    table.Percentage,
		Alignment: table.Right,
		Width:     6,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 90", Color: color.FgRed},
			{Condition: ">= 80", Color: color.FgYellow},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

package allocation

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "allocation", "Show shard allocation per node", Render)
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	allocations, err := es.Allocation(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Allocation")
	for _, a := range allocations {
		tbl.Add(
			a.Node,
			parse.Int(a.Shards),
			parse.Float(a.DiskIndices),
			parse.Float(a.DiskUsed),
			parse.Float(a.DiskAvail),
			parse.Float(a.DiskTotal),
			parse.Float(a.DiskPercent),
		)
	}
	tbl.Column(0, table.Column{Name: "NODE", Truncate: 18})
	tbl.Column(1, table.Column{
		Name:      "SHARDS",
		Alignment: table.Right,
		Width:     6,
		Colors: []table.ColorRule{
			{Condition: "== 0", Color: color.FgYellow},
		},
	})
	tbl.Column(2, table.Column{Name: "DISK.INDICES", Format: table.Bytes, Alignment: table.Right, Width: 12})
	tbl.Column(3, table.Column{Name: "DISK.USED", Format: table.Bytes, Alignment: table.Right, Width: 12})
	tbl.Column(4, table.Column{Name: "DISK.AVAIL", Format: table.Bytes, Alignment: table.Right, Width: 12})
	tbl.Column(5, table.Column{Name: "DISK.TOTAL", Format: table.Bytes, Alignment: table.Right, Width: 12})
	tbl.Column(6, table.Column{
		Name:      "DISK%",
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

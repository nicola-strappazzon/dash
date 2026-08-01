package nodes

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
	return runner.SectionCommand(ctx, opts, "nodes", "Show cluster nodes", Render)
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	nodes, err := es.Nodes(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Cluster nodes")
	for _, n := range nodes {
		tbl.Add(
			n.ID,
			n.Name,
			n.IP,
			n.Master,
			parse.Float(n.HeapPercent),
			parse.Float(n.RAMPercent),
			parse.Float(n.CPU),
			parse.Float(n.DiskUsedPercent),
			parse.Uptime(n.Uptime),
		)
	}
	tbl.Column(0, table.Column{Name: "ID"})
	tbl.Column(1, table.Column{Name: "NAME"})
	tbl.Column(2, table.Column{Name: "IP"})
	tbl.Column(3, table.Column{Name: "MASTER", Alignment: table.Center})
	tbl.Column(4, table.Column{Name: "HEAP", Format: table.Percentage, Alignment: table.Right, Width: 5})
	tbl.Column(5, table.Column{Name: "RAM", Format: table.Percentage, Alignment: table.Right, Width: 5})
	tbl.Column(6, table.Column{
		Name:      "CPU",
		Format:    table.Percentage,
		Alignment: table.Right,
		Width:     6,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 80", Color: color.FgRed},
			{Condition: ">= 70", Color: color.FgYellow},
		},
	})
	tbl.Column(7, table.Column{
		Name:      "DISK",
		Format:    table.Percentage,
		Alignment: table.Right,
		Width:     8,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 90", Color: color.FgRed},
			{Condition: ">= 80", Color: color.FgYellow},
		},
	})
	tbl.Column(8, table.Column{
		Name:      "UPTIME",
		Format:    table.Duration,
		Width:     9,
		Alignment: table.Right,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "< 3600", Color: color.FgRed},
			{Condition: "< 86400", Color: color.FgYellow},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SortBy(1)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

package indices

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := runner.SectionCommand(ctx, opts, "indices", "Show indices", Render)

	// Running "indices" bare (no other section to combine with, e.g.
	// "indices health") prints help instead of silently rendering — this
	// command also holds "hot", so nothing should happen by assumption; run
	// "indices show" to render.
	render := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return render(cmd, args)
	}

	cmd.AddCommand(
		newShowCommand(ctx, opts),
		newHotCommand(ctx, opts),
	)

	return cmd
}

func newShowCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show indices",
		Args:  cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return Render(ctx, es)
			})
		}),
	}
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	indices, err := es.Indices(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Indices")
	for _, i := range indices {
		tbl.Add(
			i.Name,
			strings.ToUpper(i.Health),
			i.Status,
			parse.Int(i.Primary),
			parse.Int(i.Replica),
			parse.Int(i.DocsCount),
			parse.Int(i.DocsDeleted),
			parse.Float(i.StoreSize),
			parse.Float(i.PriStoreSize),
		)
	}
	tbl.Column(0, table.Column{Name: "NAME", Truncate: 38})
	tbl.Column(1, table.Column{
		Name: "HEALTH",
		Colors: []table.ColorRule{
			{Condition: `== "GREEN"`, Color: color.FgGreen},
			{Condition: `== "YELLOW"`, Color: color.FgYellow},
			{Condition: `== "RED"`, Color: color.FgRed},
		},
	})
	tbl.Column(2, table.Column{
		Name: "STATUS",
		Colors: []table.ColorRule{
			{Condition: `== "open"`, Color: color.FgGreen},
			{Condition: `== "close"`, Color: color.FgYellow},
		},
	})
	tbl.Column(3, table.Column{Name: "PRI", Alignment: table.Right, Width: 3})
	tbl.Column(4, table.Column{Name: "REP", Alignment: table.Right, Width: 3})
	tbl.Column(5, table.Column{
		Name:      "DOCS",
		Alignment: table.Right,
		Width:     12,
		ZeroFill:  true,
		Precision: 12,
		Scale:     0,
	})
	tbl.Column(6, table.Column{
		Name:      "DELETED",
		Alignment: table.Right,
		Width:     11,
		Color:     color.FgGreen,
		ZeroFill:  true,
		Precision: 11,
		Scale:     0,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	tbl.Column(7, table.Column{Name: "STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(8, table.Column{Name: "PRI.STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Total(5)
	tbl.Total(6)
	tbl.Total(7)
	tbl.Total(8)
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

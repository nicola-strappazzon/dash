package thread

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
	cmd := runner.SectionCommand(ctx, opts, "thread", "Show write thread pool", Render)

	// Running "thread" bare (no other section to combine with, e.g. "thread
	// health") prints help instead of silently rendering — this command also
	// holds "hot", so nothing should happen by assumption; run "thread show"
	// to render.
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
		Short: "Show write thread pool",
		Args:  cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return Render(ctx, es)
			})
		}),
	}
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	pools, err := es.ThreadPools(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Write thread pool")
	for _, p := range pools {
		tbl.Add(
			p.NodeName,
			p.Name,
			parse.Int(p.Active),
			parse.Int(p.Queue),
			parse.Int(p.Rejected),
			parse.Int(p.Size),
		)
	}
	tbl.Column(0, table.Column{Name: "NODE"})
	tbl.Column(1, table.Column{Name: "NAME"})
	tbl.Column(2, table.Column{Name: "ACTIVE", Alignment: table.Right, Width: 6})
	tbl.Column(3, table.Column{
		Name:      "QUEUE",
		Alignment: table.Right,
		Width:     5,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 100", Color: color.FgRed},
			{Condition: ">= 1", Color: color.FgYellow},
		},
	})
	tbl.Column(4, table.Column{
		Name:      "REJECTED",
		Alignment: table.Right,
		Width:     8,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 1", Color: color.FgRed},
		},
	})
	tbl.Column(5, table.Column{Name: "SIZE", Alignment: table.Right, Width: 4})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SortBy(0)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

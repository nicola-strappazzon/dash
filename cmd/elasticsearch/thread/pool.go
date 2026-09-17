package thread

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func newPoolCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "pool",
		Short: "Show search thread pool activity, queues and rejections",
		Long: "Show active search tasks, queued tasks and cumulative rejections per node. " +
			"Rejections are counted since each node started, not just during the current interval. " +
			"Tasks are internal operations, not necessarily individual client searches.",
		Args: cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return renderSearchPool(ctx, es)
			})
		}),
	}
}

func renderSearchPool(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	pools, err := es.SearchThreadPools(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Search thread pool")
	for _, p := range pools {
		tbl.Add(p.NodeName, parse.Int(p.Active), parse.Int(p.Queue), parse.Int(p.Rejected))
	}
	tbl.Column(0, table.Column{Name: "NODE"})
	tbl.Column(1, table.Column{Name: "ACTIVE", Alignment: table.Right})
	tbl.Column(2, table.Column{Name: "QUEUE", Alignment: table.Right})
	tbl.Column(3, table.Column{Name: "REJECTED (TOTAL)", Alignment: table.Right})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SortBy(0)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println()
	return nil
}

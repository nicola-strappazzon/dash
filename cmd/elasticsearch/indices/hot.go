package indices

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func newHotCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var node string
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "hot --node <node-name>",
		Short: "Find which index is driving indexing/refresh/flush load on a node",
		Long: "Samples _nodes/stats/indices (indexing, refresh, flush, level=shards) for the " +
			"given node twice, `--interval` apart, and shows the per-second delta for every " +
			"index with shards on that node — the one at the top is what's actually keeping " +
			"the node busy right now, unlike cumulative totals which favor old indices that " +
			"aren't active anymore.",
		Args: cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			if node == "" {
				return fmt.Errorf("--node is required")
			}
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return renderHot(ctx, es, node, interval)
			})
		}),
	}
	cmd.Flags().StringVar(&node, "node", "", "the node to sample (required)")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "how long to wait between the two samples")

	return cmd
}

func renderHot(ctx context.Context, es *elasticsearch.Elasticsearch, node string, interval time.Duration) error {
	before, err := es.IndexStats(ctx, node)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(interval):
	}

	after, err := es.IndexStats(ctx, node)
	if err != nil {
		return err
	}

	seconds := interval.Seconds()
	type row struct {
		index string
		delta elasticsearch.IndexTotals
	}
	rows := make([]row, 0, len(after))
	for index, a := range after {
		b := before[index] // zero value if the index wasn't there yet, delta is just "after"
		rows = append(rows, row{
			index: index,
			delta: elasticsearch.IndexTotals{
				IndexTotal:  a.IndexTotal - b.IndexTotal,
				IndexTimeMs: a.IndexTimeMs - b.IndexTimeMs,
				Refresh:     a.Refresh - b.Refresh,
				RefreshMs:   a.RefreshMs - b.RefreshMs,
				Flush:       a.Flush - b.Flush,
				FlushMs:     a.FlushMs - b.FlushMs,
			},
		})
	}

	// Sort by combined time spent per second (indexing + refresh + flush), the
	// closest proxy we have to "how much CPU this index is costing per second".
	sort.Slice(rows, func(i, j int) bool {
		loadI := rows[i].delta.IndexTimeMs + rows[i].delta.RefreshMs + rows[i].delta.FlushMs
		loadJ := rows[j].delta.IndexTimeMs + rows[j].delta.RefreshMs + rows[j].delta.FlushMs
		return loadI > loadJ
	})

	tbl := table.New()
	tbl.Title(fmt.Sprintf("Hot indices on %s", node))
	for _, r := range rows {
		tbl.Add(
			r.index,
			float64(r.delta.IndexTotal)/seconds,
			float64(r.delta.IndexTimeMs)/seconds,
			float64(r.delta.Refresh)/seconds,
			float64(r.delta.RefreshMs)/seconds,
			float64(r.delta.Flush)/seconds,
			float64(r.delta.FlushMs)/seconds,
		)
	}
	tbl.Column(0, table.Column{Name: "INDEX", Truncate: 34})
	tbl.Column(1, table.Column{Name: "INDEX/s", Alignment: table.Right, Width: 8, Precision: 1})
	tbl.Column(2, table.Column{
		Name:      "IDX_MS/s",
		Alignment: table.Right,
		Width:     9,
		Precision: 1,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 1000", Color: color.FgRed},
			{Condition: ">= 200", Color: color.FgYellow},
		},
	})
	tbl.Column(3, table.Column{Name: "REFRESH/s", Alignment: table.Right, Width: 9, Precision: 1})
	tbl.Column(4, table.Column{
		Name:      "REFR_MS/s",
		Alignment: table.Right,
		Width:     9,
		Precision: 1,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 1000", Color: color.FgRed},
			{Condition: ">= 200", Color: color.FgYellow},
		},
	})
	tbl.Column(5, table.Column{Name: "FLUSH/s", Alignment: table.Right, Width: 8, Precision: 1})
	tbl.Column(6, table.Column{
		Name:      "FLSH_MS/s",
		Alignment: table.Right,
		Width:     9,
		Precision: 1,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 1000", Color: color.FgRed},
			{Condition: ">= 200", Color: color.FgYellow},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

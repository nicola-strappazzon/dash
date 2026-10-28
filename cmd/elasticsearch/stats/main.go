package stats

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var previous map[string]elasticsearch.IndexStatistics
	var indexFilter string
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show index storage and read/write activity",
		Long: "Show cumulative index activity, or rates between samples with --watch. " +
			"Lists all indices by default. Filter with --index using a name, alias, wildcard pattern or comma-separated list.\n\n" +
			"DOCS includes nested Lucene documents. SIZE and writes use primary shards; " +
			"TOTAL.SIZE, searches and GETs include replicas. SEARCH counts shard operations, " +
			"not application requests. INDEX includes inserts and updates. SEARCH.ms is " +
			"the average shard query time, not end-to-end latency.\n\n" +
			"With --watch, rates show '-' on the first sample and after shard assignment " +
			"changes, index replacement or detected counter resets. These counters are not a historical access log.",
		Example: "  dash elasticsearch --address https://localhost:9200 stats\n" +
			"  dash elasticsearch --address https://localhost:9200 stats --index example-index\n" +
			"  dash elasticsearch --address https://localhost:9200 stats --index 'example-*' --watch 10s",
		ValidArgsFunction: cobra.NoFileCompletions,
		Args:              cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				defer es.Close()
				indices, err := es.IndexStatistics(ctx, indexFilter)
				if err != nil {
					return err
				}
				if err := render(ctx, indices, previous, opts.Watch > 0); err != nil {
					return err
				}
				previous = make(map[string]elasticsearch.IndexStatistics, len(indices))
				for _, index := range indices {
					previous[index.Name] = index
				}
				return nil
			})
		}),
	}
	cmd.Flags().StringVar(&indexFilter, "index", "", "only show stats for this index (name, alias, pattern or comma-separated list)")
	return cmd
}

func render(ctx context.Context, indices []elasticsearch.IndexStatistics, previous map[string]elasticsearch.IndexStatistics, watch bool) error {
	tbl := table.New()
	tbl.Title("Index statistics")
	tbl.TitleSeparator(false)
	for _, index := range indices {
		a := index.Activity
		var searches, gets, indexed, deleted any = a.Searches, a.Gets, a.Indexed, a.Deleted
		var searchMillis any = "-"
		if a.Searches > 0 {
			searchMillis = fmt.Sprintf("%.2f", float64(a.SearchTimeMillis)/float64(a.Searches))
			if command.JSON(ctx) {
				searchMillis = float64(a.SearchTimeMillis) / float64(a.Searches)
			}
		}
		if watch {
			searches, gets, indexed, deleted, searchMillis = "-", "-", "-", "-", "-"
			if before, exists := previous[index.Name]; exists {
				if rates, valid := index.RatesSince(before); valid {
					searches = fmt.Sprintf("%.2f", rates.Searches)
					gets = fmt.Sprintf("%.2f", rates.Gets)
					indexed = fmt.Sprintf("%.2f", rates.Indexed)
					deleted = fmt.Sprintf("%.2f", rates.Deleted)
					if command.JSON(ctx) {
						searches, gets, indexed, deleted = rates.Searches, rates.Gets, rates.Indexed, rates.Deleted
					}
					if rates.Searches > 0 {
						searchMillis = fmt.Sprintf("%.2f", rates.SearchMillis)
						if command.JSON(ctx) {
							searchMillis = rates.SearchMillis
						}
					}
				}
			}
		}
		tbl.Add(index.Name, index.Documents, index.PrimaryBytes, index.TotalBytes, searches, searchMillis, gets, indexed, deleted)
	}
	tbl.Column(0, table.Column{Name: "NAME", MaxWidth: 40})
	tbl.Column(1, table.Column{Name: "DOCS", Alignment: table.Right})
	tbl.Column(2, table.Column{Name: "SIZE", Format: table.Bytes, Alignment: table.Right})
	tbl.Column(3, table.Column{Name: "TOTAL.SIZE", Format: table.Bytes, Alignment: table.Right})
	suffix := ""
	if watch {
		suffix = "/s"
	}
	tbl.Column(4, table.Column{Name: "SEARCH" + suffix, Alignment: table.Right})
	tbl.Column(5, table.Column{Name: "SEARCH.ms", Alignment: table.Right})
	tbl.Column(6, table.Column{Name: "GET" + suffix, Alignment: table.Right})
	tbl.Column(7, table.Column{Name: "INDEX" + suffix, Alignment: table.Right})
	tbl.Column(8, table.Column{Name: "DELETE" + suffix, Alignment: table.Right})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.FitWidth(table.TerminalWidth())
	if err := command.Print(ctx, tbl); err != nil {
		return err
	}
	if len(indices) == 0 {
		command.Message(ctx, "  No open indices matched.")
	}
	if watch {
		command.Message(ctx, "  Rates: '-' means no comparable sample. SEARCH counts shard operations; INDEX includes updates.")
	}
	return nil
}

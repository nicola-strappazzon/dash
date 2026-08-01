package shards

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

var (
	nodeFilter  string
	indexFilter string
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := runner.SectionCommand(ctx, opts, "shards", "Show shards", Render)
	cmd.Flags().StringVar(&nodeFilter, "node", "", "only show shards allocated to this node, sorted by size (biggest first)")
	cmd.Flags().StringVar(&indexFilter, "index", "", "only show shards for this index (name or pattern, e.g. cr-*); without --node, shows a per-node summary instead of the full list")

	return cmd
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	sortBy := "state,index,shard"
	if nodeFilter != "" {
		sortBy = "store:desc"
	}

	shards, err := es.Shards(ctx, sortBy, indexFilter)
	if err != nil {
		return err
	}

	if indexFilter != "" && nodeFilter == "" {
		return renderNodeSummary(shards, indexFilter)
	}

	tbl := table.New()
	tbl.Title("Shards")
	for _, s := range shards {
		if nodeFilter != "" && s.Node != nodeFilter {
			continue
		}
		tbl.Add(
			s.Index,
			parse.Int(s.Shard),
			strings.ToUpper(s.Prirep),
			s.State,
			parse.Int(s.Docs),
			parse.Float(s.Store),
			s.IP,
			s.Node,
		)
	}
	tbl.Column(0, table.Column{Name: "INDEX", Truncate: 34})
	tbl.Column(1, table.Column{Name: "SHARD", Alignment: table.Right, Width: 5})
	tbl.Column(2, table.Column{
		Name:      "P/R",
		Alignment: table.Center,
		Width:     3,
		Colors: []table.ColorRule{
			{Condition: `== "P"`, Color: color.FgGreen},
			{Condition: `== "R"`, Color: color.FgYellow},
		},
	})
	tbl.Column(3, table.Column{
		Name: "STATE",
		Colors: []table.ColorRule{
			{Condition: `== "STARTED"`, Color: color.FgGreen},
			{Condition: `== "INITIALIZING"`, Color: color.FgYellow},
			{Condition: `== "RELOCATING"`, Color: color.FgYellow},
			{Condition: `== "UNASSIGNED"`, Color: color.FgRed},
		},
	})
	tbl.Column(4, table.Column{Name: "DOCS", Alignment: table.Right, Width: 11})
	tbl.Column(5, table.Column{Name: "STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(6, table.Column{Name: "IP"})
	tbl.Column(7, table.Column{Name: "NODE", Truncate: 18})
	if nodeFilter != "" {
		tbl.Total(5)
	}
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

// renderNodeSummary groups shards by node, so you can compare how a given
// index (or pattern, e.g. cr-*) is spread across the cluster at a glance,
// instead of eyeballing a full per-shard listing.
func renderNodeSummary(shards []elasticsearch.Shard, indexFilter string) error {
	type agg struct {
		count int64
		store float64
	}
	byNode := map[string]agg{}
	for _, s := range shards {
		a := byNode[s.Node]
		a.count++
		a.store += parse.Float(s.Store)
		byNode[s.Node] = a
	}

	nodes := make([]string, 0, len(byNode))
	for node := range byNode {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return byNode[nodes[i]].count > byNode[nodes[j]].count
	})

	tbl := table.New()
	tbl.Title(fmt.Sprintf("Shards per node for %s", indexFilter))
	for _, node := range nodes {
		tbl.Add(node, byNode[node].count, byNode[node].store)
	}
	tbl.Column(0, table.Column{Name: "NODE"})
	tbl.Column(1, table.Column{Name: "SHARDS", Alignment: table.Right, Width: 6})
	tbl.Column(2, table.Column{Name: "STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Total(1)
	tbl.Total(2)
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

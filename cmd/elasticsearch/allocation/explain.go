package allocation

import (
	"context"
	"fmt"
	"strings"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func newExplainCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var index, node string
	var shard int
	var primary bool
	cmd := &cobra.Command{
		Use:   "explain --index <name> --shard <id>",
		Short: "Explain why a shard is unassigned or cannot move",
		Long: "Explain allocation for a replica by default; use --primary for the primary. " +
			"Use --node to select a copy currently assigned to a specific node. " +
			"Without --node, Elasticsearch selects a matching copy. " +
			"Shows node decisions and full reasons without changing allocation.",
		Example: "  dash elasticsearch allocation explain --index example-index --shard 0\n" +
			"  dash elasticsearch allocation explain --index example-index --shard 0 --primary\n" +
			"  dash elasticsearch allocation explain --index example-index --shard 0 --node data-01 --watch 10s",
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(index) == "" {
				return fmt.Errorf("--index must not be empty")
			}
			if shard < 0 {
				return fmt.Errorf("--shard must be zero or greater")
			}
			return nil
		},
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				defer es.Close()
				explanation, err := es.ExplainAllocation(ctx, index, shard, primary, node)
				if err != nil {
					return err
				}
				if command.JSON(ctx) {
					return command.PrintJSON(explanation)
				}
				renderExplanation(explanation)
				return nil
			})
		}),
	}
	cmd.Flags().StringVar(&index, "index", "", "index name (required)")
	cmd.Flags().IntVar(&shard, "shard", 0, "shard ID, starting at zero (required)")
	cmd.Flags().BoolVar(&primary, "primary", false, "explain the primary instead of a replica")
	cmd.Flags().StringVar(&node, "node", "", "node name or ID currently holding the copy (omit for unassigned copies)")
	cmd.MarkFlagRequired("index")
	cmd.MarkFlagRequired("shard")
	return cmd
}

func renderExplanation(e *elasticsearch.AllocationExplanation) {
	copy := "replica"
	if e.Primary {
		copy = "primary"
	}
	row := table.NewRow()
	row.Title("Shard allocation explanation")
	row.Add(table.Field{Name: "Index", Value: e.Index})
	row.Add(table.Field{Name: "Shard", Value: e.Shard})
	row.Add(table.Field{Name: "Copy", Value: copy})
	row.Add(table.Field{Name: "State", Value: e.CurrentState})
	if e.CurrentNode != nil {
		row.Add(table.Field{Name: "Current node", Value: e.CurrentNode.Name})
	}
	if e.UnassignedInfo != nil {
		row.Add(table.Field{Name: "Unassigned reason", Value: e.UnassignedInfo.Reason})
		row.Add(table.Field{Name: "Unassigned since", Value: e.UnassignedInfo.At})
		if e.UnassignedInfo.Details != "" {
			row.Add(table.Field{Name: "Details", Value: e.UnassignedInfo.Details})
		}
	}
	for _, field := range []table.Field{
		{Name: "Explanation", Value: e.Explanation},
		{Name: "Can allocate", Value: e.CanAllocate},
		{Name: "Allocation", Value: e.AllocateExplanation},
		{Name: "Can remain", Value: e.CanRemain},
		{Name: "Can move", Value: e.CanMove},
		{Name: "Movement", Value: e.MoveExplanation},
		{Name: "Can rebalance cluster", Value: e.CanRebalanceCluster},
		{Name: "Can rebalance to node", Value: e.CanRebalanceToOtherNode},
		{Name: "Rebalance", Value: e.RebalanceExplanation},
	} {
		if field.Value != "" {
			row.Add(field)
		}
	}
	if e.RemainingDelayMillis > 0 {
		row.Add(table.Field{Name: "Remaining delay", Value: float64(e.RemainingDelayMillis) / 1000, Format: table.Duration})
	}
	row.SetWidth(table.TerminalWidth())
	row.Print()
	printDecisions("Current node", e.CanRemainDecisions)
	printDecisions("Cluster rebalance", e.CanRebalanceDecisions)
	if len(e.NodeDecisions) > 0 {
		tbl := table.New()
		tbl.Title("Node decisions")
		tbl.TitleSeparator(false)
		for _, node := range e.NodeDecisions {
			deciders := make([]string, 0, len(node.Deciders))
			for _, d := range node.Deciders {
				deciders = append(deciders, d.Decider)
			}
			tbl.Add(node.Name, strings.ToUpper(node.Decision), strings.Join(deciders, ", "))
		}
		tbl.Column(0, table.Column{Name: "NODE", MaxWidth: 32})
		tbl.Column(1, table.Column{Name: "DECISION", MaxWidth: 24})
		tbl.Column(2, table.Column{Name: "DECIDERS", MaxWidth: 70})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(3)
		tbl.FitWidth(table.TerminalWidth())
		tbl.Print()
		for _, node := range e.NodeDecisions {
			printDecisions(node.Name, node.Deciders)
		}
	}
	fmt.Println()
}

func printDecisions(scope string, decisions []elasticsearch.AllocationDecision) {
	for _, d := range decisions {
		fmt.Printf("  %s [%s / %s]: %s\n", scope, d.Decider, d.Decision, d.Explanation)
	}
}

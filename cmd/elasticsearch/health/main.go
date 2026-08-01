package health

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "health", "Show cluster health", Render)
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	health, err := es.Health(ctx)
	if err != nil {
		return err
	}

	row := table.NewRow()
	row.Title("Cluster Health")
	row.Add(table.Field{Name: "Name", Value: health.ClusterName})
	row.Add(table.Field{
		Name:  "Status",
		Value: health.Status,
		Colors: []table.ColorRule{
			{Condition: `== "green"`, Color: color.FgGreen},
			{Condition: `== "yellow"`, Color: color.FgYellow},
			{Condition: `== "red"`, Color: color.FgRed},
		},
	})
	row.Add(table.Field{Name: "number_of_nodes", Value: health.NumberOfNodes})
	row.Add(table.Field{Name: "number_of_data_nodes", Value: health.NumberOfDataNodes})
	row.Add(table.Field{Name: "active_primary_shards", Value: health.ActivePrimaryShards})
	row.Add(table.Field{Name: "active_shards", Value: health.ActiveShards})
	row.Add(table.Field{
		Name:  "relocating_shards",
		Value: health.RelocatingShards,
		Colors: []table.ColorRule{
			{Condition: "== 0", Color: color.FgGreen},
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	row.Add(table.Field{
		Name:  "initializing_shards",
		Value: health.InitializingShards,
		Colors: []table.ColorRule{
			{Condition: "== 0", Color: color.FgGreen},
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	row.Add(table.Field{
		Name:  "unassigned_shards",
		Value: health.UnassignedShards,
		Colors: []table.ColorRule{
			{Condition: "== 0", Color: color.FgGreen},
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	row.Add(table.Field{
		Name:  "number_of_pending_tasks",
		Value: health.NumberOfPendingTasks,
		Colors: []table.ColorRule{
			{Condition: "== 0", Color: color.FgGreen},
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	row.Add(table.Field{
		Name:   "active_shards_percent_as_number",
		Value:  health.ActiveShardsPercentAsNumber,
		Format: table.Percentage,
		Colors: []table.ColorRule{
			{Condition: "== 100", Color: color.FgGreen},
			{Condition: "< 100", Color: color.FgYellow},
		},
	})
	row.SetWidth(table.TerminalWidth())
	row.Print()
	fmt.Println("")

	return nil
}

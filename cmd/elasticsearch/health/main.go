package health

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	elasticsearchdriver "github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

type healthResponse struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "health", "Show cluster health", Render)
}

func Render(ctx context.Context, es *elasticsearchdriver.Elasticsearch) error {
	var health healthResponse

	res, err := es.Client().Cluster.Health(
		es.Client().Cluster.Health.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error en la respuesta: %s", res.String())
	}

	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return fmt.Errorf("error al decodificar _cluster/health: %w", err)
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
	row.Print()
	fmt.Println("")

	return nil
}

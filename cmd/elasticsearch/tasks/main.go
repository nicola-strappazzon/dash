package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var subtasks bool
	var describe string
	cmd := runner.SectionCommand(ctx, opts, "tasks", "Show active top-level Elasticsearch tasks and progress", func(ctx context.Context, es *elasticsearch.Elasticsearch) error {
		if describe != "" {
			detail, err := es.TaskDetail(ctx, describe)
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			encoder.SetEscapeHTML(false)
			return encoder.Encode(detail)
		}
		return render(ctx, es, subtasks)
	})
	cmd.Flags().BoolVar(&subtasks, "subtasks", false, "include subtasks")
	cmd.Flags().StringVar(&describe, "describe", "", "show complete JSON details for a task ID (node:id)")
	cmd.MarkFlagsMutuallyExclusive("describe", "subtasks")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("describe") && describe == "" {
			return fmt.Errorf("--describe requires a task ID")
		}
		return nil
	}
	return cmd
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	return render(ctx, es, false)
}

func render(ctx context.Context, es *elasticsearch.Elasticsearch, subtasks bool) error {
	tasks, err := es.Tasks(ctx)
	if err != nil {
		return err
	}
	tbl := table.New()
	tbl.Title("Active tasks")
	tbl.TitleSeparator(false)
	hidden := 0
	for _, task := range tasks {
		if !subtasks && task.ParentTaskID != "" && task.ParentTaskID != "-1" {
			hidden++
			continue
		}
		var processed, total any = "-", "-"
		if task.Status != nil && task.Status.Total != nil {
			processed = task.Status.Created + task.Status.Updated + task.Status.Deleted + task.Status.Noops + task.Status.VersionConflicts
			total = *task.Status.Total
		}
		tbl.Add(task.TaskID, task.Node, task.Action, float64(task.RunningTimeNanos)/1e9, processed, total, task.Description)
	}
	tbl.Column(0, table.Column{Name: "ID"})
	tbl.Column(1, table.Column{Name: "NODE", MaxWidth: 20})
	tbl.Column(2, table.Column{Name: "ACTION", MaxWidth: 45})
	tbl.Column(3, table.Column{Name: "TIME", Format: table.Duration, Alignment: table.Right})
	tbl.Column(4, table.Column{Name: "PROCESSED", Alignment: table.Right})
	tbl.Column(5, table.Column{Name: "TOTAL", Alignment: table.Right})
	tbl.Column(6, table.Column{Name: "DESCRIPTION", MaxWidth: 60})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.FitWidth(table.TerminalWidth())
	tbl.Print()
	if hidden > 0 {
		fmt.Printf("  %d subtasks hidden (use tasks --subtasks to include them)\n", hidden)
	}
	fmt.Println("")
	return nil
}

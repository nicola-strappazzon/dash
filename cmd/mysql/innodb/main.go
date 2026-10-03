package innodb

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "innodb", "Show InnoDB internals", Render)
}

func Render(ctx context.Context, db *mysql.MySQL) error {
	status, err := db.InnoDB(ctx)
	if err != nil {
		return err
	}

	printOverview(status)
	return nil
}

func printOverview(status mysql.InnoDBStatus) {
	row := table.NewRow().Title("InnoDB")
	row.Add(table.Field{
		Name:  "History list length",
		Value: status.HistoryLength,
		Colors: []table.ColorRule{
			{Condition: "> 100000", Color: color.FgRed},
			{Condition: "> 10000", Color: color.FgYellow},
			{Condition: ">= 0", Color: color.FgGreen},
		},
	})
	row.Add(table.Field{
		Name:  "Queries",
		Value: fmt.Sprintf("%d inside · %d queued", status.QueriesInsideInnoDB, status.QueriesInQueue),
		Colors: []table.ColorRule{
			{Condition: `!= "0 inside · 0 queued"`, Color: color.FgYellow},
			{Condition: `== "0 inside · 0 queued"`, Color: color.FgGreen},
		},
	})

	if status.BufferPoolSizePages > 0 {
		usedPages := status.BufferPoolSizePages - status.BufferPoolFreePages
		row.Add(table.Field{Name: "Buffer pool", Value: fmt.Sprintf("%s used · %s free", pages(usedPages), pages(status.BufferPoolFreePages))})
		row.Add(table.Field{
			Name:  "Dirty pages",
			Value: dirtyPagePercent(status),
			Colors: []table.ColorRule{
				{Condition: ">= 75", Color: color.FgRed},
				{Condition: ">= 50", Color: color.FgYellow},
				{Condition: ">= 0", Color: color.FgGreen},
			},
			Precision: 2,
			Suffix:    "%",
		})
		row.Add(table.Field{
			Name:  "Buffer pool hit rate",
			Value: status.BufferPoolHitRatePct,
			Colors: []table.ColorRule{
				{Condition: "< 90", Color: color.FgRed},
				{Condition: "< 99", Color: color.FgYellow},
				{Condition: ">= 99", Color: color.FgGreen},
			},
			Precision: 2,
			Suffix:    "%",
		})
		if status.BufferPoolReadsPerSec != 0 || status.BufferPoolWritesPerSec != 0 {
			row.Add(table.Field{Name: "Buffer pool activity", Value: fmt.Sprintf("%.2f reads/s · %.2f writes/s", status.BufferPoolReadsPerSec, status.BufferPoolWritesPerSec)})
		}
	}

	if status.InsertsPerSec != 0 || status.UpdatesPerSec != 0 || status.DeletesPerSec != 0 {
		row.Add(table.Field{Name: "Row operations", Value: fmt.Sprintf("%.2f ins/s · %.2f upd/s · %.2f del/s", status.InsertsPerSec, status.UpdatesPerSec, status.DeletesPerSec)})
	}
	if status.RowReadsPerSec != 0 {
		row.Add(table.Field{Name: "Row reads", Value: fmt.Sprintf("%.2f/s", status.RowReadsPerSec)})
	}
	if status.LogSequenceNumber > status.LastCheckpoint {
		row.Add(table.Field{Name: "Checkpoint age", Value: status.LogSequenceNumber - status.LastCheckpoint, Format: table.Bytes})
	}

	row.Print()
	fmt.Println("")
}

func dirtyPagePercent(status mysql.InnoDBStatus) float64 {
	if status.BufferPoolSizePages == 0 {
		return 0
	}
	return float64(status.BufferPoolModifiedPages) * 100 / float64(status.BufferPoolSizePages)
}

func pages(value int64) string {
	if value >= 1_000_000 {
		return fmt.Sprintf("%.2fM pages", float64(value)/1_000_000)
	}
	if value >= 1_000 {
		return fmt.Sprintf("%.1fK pages", float64(value)/1_000)
	}
	return fmt.Sprintf("%d pages", value)
}

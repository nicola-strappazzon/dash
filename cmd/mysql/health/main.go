package health

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	tracker := &mysql.DeadlockTracker{}
	return runner.SectionCommand(ctx, opts, "health", "Show MySQL health", func(ctx context.Context, db *mysql.MySQL) error {
		return Render(ctx, db, tracker)
	})
}

func Render(ctx context.Context, db *mysql.MySQL, tracker *mysql.DeadlockTracker) error {
	report, err := db.Health(ctx)
	if err != nil {
		return err
	}
	checks := report.Checks
	checks = append([]mysql.HealthCheck{{
		Name:    "MySQL version",
		Display: report.Version,
		Status:  mysql.HealthOK,
	}}, checks...)
	if report.HasDeadlocks {
		checks = append(checks, tracker.Check(report.Deadlocks, time.Now()))
	}

	tbl := table.New()
	tbl.Title("MySQL health")
	tbl.TitleSeparator(false)
	for _, check := range checks {
		tbl.Add(check.Name, check.Display, strings.ToUpper(string(check.Status)))
	}
	tbl.Column(0, table.Column{Name: "CHECK"})
	tbl.Column(1, table.Column{Name: "VALUE", Alignment: table.Right})
	tbl.Column(2, table.Column{
		Name: "STATUS",
		Colors: []table.ColorRule{
			{Condition: `== "CRITICAL"`, Color: color.FgRed},
			{Condition: `== "WARNING"`, Color: color.FgYellow},
			{Condition: `== "OK"`, Color: color.FgGreen},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")
	return nil
}

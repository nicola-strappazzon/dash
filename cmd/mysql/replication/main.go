package replication

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "replication", "Show MySQL replication details", Render)
}

func Render(ctx context.Context, db *mysql.MySQL) error {
	report, isReplica, err := db.ReplicationDetails(ctx)
	if err != nil {
		return err
	}
	if !isReplica {
		if command.JSON(ctx) {
			return command.PrintJSON(map[string]bool{"is_replica": false})
		}
		fmt.Println("MySQL server is not configured as a replica.")
		return nil
	}

	if err := printChecks(ctx, report.Checks); err != nil {
		return err
	}
	return printStatus(ctx, report.Status)
}

func printChecks(ctx context.Context, checks []mysql.HealthCheck) error {
	tbl := table.New()
	tbl.Title("MySQL replication")
	for _, check := range checks {
		tbl.Add(check.Name, check.Display, strings.ToUpper(string(check.Status)))
	}
	tbl.Column(0, table.Column{Name: "CHECK"})
	tbl.Column(1, table.Column{Name: "VALUE", Alignment: table.Right, MaxWidth: 100})
	tbl.Column(2, table.Column{Name: "STATUS", Colors: statusColors()})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.FitWidth(table.TerminalWidth())
	tbl.SetWidth(table.TerminalWidth())
	return command.Print(ctx, tbl)
}

func printStatus(ctx context.Context, status mysql.ReplicationStatus) error {
	tbl := table.New()
	tbl.Title("Replication coordinates")
	tbl.Add("Source", fmt.Sprintf("%s:%d", status.SourceHost, status.SourcePort))
	tbl.Add("Source log", fmt.Sprintf("%s:%d", status.SourceLogFile, status.ReadSourceLogPos))
	tbl.Add("Executed source log", fmt.Sprintf("%s:%d", status.RelaySourceLogFile, status.ExecSourceLogPos))
	tbl.Add("Relay log", fmt.Sprintf("%s:%d", status.RelayLogFile, status.RelayLogPos))
	tbl.Add("Relay log space", table.Field{Value: status.RelayLogSpace, Format: table.Bytes}.Render())
	tbl.Add("Auto position", status.AutoPosition)
	if status.RetrievedGTIDSet != "" {
		tbl.Add("Retrieved GTID", status.RetrievedGTIDSet)
	}
	if status.ExecutedGTIDSet != "" {
		tbl.Add("Executed GTID", status.ExecutedGTIDSet)
	}
	tbl.Column(0, table.Column{Name: "METRIC"})
	tbl.Column(1, table.Column{Name: "VALUE", MaxWidth: 100})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.FitWidth(table.TerminalWidth())
	tbl.SetWidth(table.TerminalWidth())
	return command.Print(ctx, tbl)
}

func statusColors() []table.ColorRule {
	return []table.ColorRule{
		{Condition: `== "CRITICAL"`, Color: color.FgRed},
		{Condition: `== "WARNING"`, Color: color.FgYellow},
		{Condition: `== "OK"`, Color: color.FgGreen},
	}
}

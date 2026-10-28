package replication

import (
	"context"
	"fmt"
	"strconv"

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
		if command.Machine(ctx) {
			return command.PrintData(ctx, map[string]bool{"is_replica": false})
		}
		fmt.Println("MySQL server is not configured as a replica.")
		return nil
	}
	return printReport(ctx, report)
}

func printReport(ctx context.Context, report mysql.ReplicationReport) error {
	tbl := command.NewTable()
	tbl.Title("MySQL replication")
	for _, row := range reportRows(report, command.Machine(ctx)) {
		tbl.Add(row[0], row[1])
	}
	tbl.Column(0, table.Column{Name: "METRIC"})
	tbl.Column(1, table.Column{Name: "VALUE", MaxWidth: table.TerminalWidth()})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.FitWidth(table.TerminalWidth())
	tbl.SetWidth(table.TerminalWidth())
	return command.Print(ctx, tbl)
}

func reportRows(report mysql.ReplicationReport, raw bool) [][2]string {
	var rows [][2]string
	for _, check := range report.Checks {
		value := check.Display
		if raw && check.Name == "Replication lag" {
			value = report.Status.SecondsBehindSource
			if value == "" {
				value = "NULL"
			}
		}
		rows = append(rows, [2]string{check.Name, value})
	}
	status := report.Status
	rows = append(rows, [2]string{"Source", fmt.Sprintf("%s:%d", status.SourceHost, status.SourcePort)})
	rows = append(rows, [2]string{"Source log", fmt.Sprintf("%s:%d", status.SourceLogFile, status.ReadSourceLogPos)})
	rows = append(rows, [2]string{"Executed source log", fmt.Sprintf("%s:%d", status.RelaySourceLogFile, status.ExecSourceLogPos)})
	rows = append(rows, [2]string{"Relay log", fmt.Sprintf("%s:%d", status.RelayLogFile, status.RelayLogPos)})
	space := table.Field{Value: status.RelayLogSpace, Format: table.Bytes}.Render()
	position := fmt.Sprint(status.AutoPosition)
	if raw {
		space = strconv.FormatInt(status.RelayLogSpace, 10)
		position = "0"
		if status.AutoPosition {
			position = "1"
		}
	}
	rows = append(rows, [2]string{"Relay log space", space})
	rows = append(rows, [2]string{"Auto position", position})
	if status.RetrievedGTIDSet != "" {
		rows = append(rows, [2]string{"Retrieved GTID", status.RetrievedGTIDSet})
	}
	if status.ExecutedGTIDSet != "" {
		rows = append(rows, [2]string{"Executed GTID", status.ExecutedGTIDSet})
	}
	return rows
}

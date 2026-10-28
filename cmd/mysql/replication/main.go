package replication

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var metric string
	var timestamp bool
	cmd := runner.SectionCommand(ctx, opts, "replication", "Show MySQL replication details", func(ctx context.Context, db *mysql.MySQL) error {
		return render(ctx, db, metric, timestamp)
	})
	cmd.Flags().StringVar(&metric, "metric", "", "show only the raw value of a metric, e.g. 'Replication lag'")
	cmd.Flags().BoolVar(&timestamp, "timestamp", false, "prefix the metric value with a Unix timestamp in seconds (requires --metric)")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if timestamp && !cmd.Flags().Changed("metric") {
			return fmt.Errorf("--timestamp requires --metric")
		}
		if cmd.Flags().Changed("metric") {
			if strings.TrimSpace(metric) == "" {
				return fmt.Errorf("--metric requires a metric name")
			}
			if len(args) > 0 {
				return fmt.Errorf("--metric cannot be combined with additional sections")
			}
		}
		return nil
	}
	return cmd
}

func Render(ctx context.Context, db *mysql.MySQL) error {
	return render(ctx, db, "", false)
}

func render(ctx context.Context, db *mysql.MySQL, metric string, timestamp bool) error {
	report, isReplica, err := db.ReplicationDetails(ctx)
	if err != nil {
		return err
	}
	if !isReplica {
		if metric != "" {
			return fmt.Errorf("MySQL server is not configured as a replica")
		}
		if command.JSON(ctx) {
			return command.PrintJSON(map[string]bool{"is_replica": false})
		}
		fmt.Println("MySQL server is not configured as a replica.")
		return nil
	}

	if metric != "" {
		value, err := metricValue(report, metric)
		if err != nil {
			return err
		}
		return printMetric(os.Stdout, value, timestamp, time.Now())
	}
	return printReport(ctx, report)
}

func printReport(ctx context.Context, report mysql.ReplicationReport) error {
	tbl := table.New()
	tbl.Title("MySQL replication")
	for _, row := range reportRows(report) {
		tbl.Add(row[0], row[1])
	}
	tbl.Column(0, table.Column{Name: "METRIC"})
	tbl.Column(1, table.Column{Name: "VALUE", MaxWidth: 100})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.FitWidth(table.TerminalWidth())
	tbl.SetWidth(table.TerminalWidth())
	return command.Print(ctx, tbl)
}

func reportRows(report mysql.ReplicationReport) [][2]string {
	var rows [][2]string
	for _, check := range report.Checks {
		rows = append(rows, [2]string{check.Name, check.Display})
	}
	status := report.Status
	rows = append(rows, [2]string{"Source", fmt.Sprintf("%s:%d", status.SourceHost, status.SourcePort)})
	rows = append(rows, [2]string{"Source log", fmt.Sprintf("%s:%d", status.SourceLogFile, status.ReadSourceLogPos)})
	rows = append(rows, [2]string{"Executed source log", fmt.Sprintf("%s:%d", status.RelaySourceLogFile, status.ExecSourceLogPos)})
	rows = append(rows, [2]string{"Relay log", fmt.Sprintf("%s:%d", status.RelayLogFile, status.RelayLogPos)})
	rows = append(rows, [2]string{"Relay log space", table.Field{Value: status.RelayLogSpace, Format: table.Bytes}.Render()})
	rows = append(rows, [2]string{"Auto position", fmt.Sprint(status.AutoPosition)})
	if status.RetrievedGTIDSet != "" {
		rows = append(rows, [2]string{"Retrieved GTID", status.RetrievedGTIDSet})
	}
	if status.ExecutedGTIDSet != "" {
		rows = append(rows, [2]string{"Executed GTID", status.ExecutedGTIDSet})
	}
	return rows
}

func metricValue(report mysql.ReplicationReport, name string) (string, error) {
	for _, row := range reportRows(report) {
		if strings.EqualFold(row[0], strings.TrimSpace(name)) {
			switch row[0] {
			case "Replication lag":
				if report.Status.SecondsBehindSource == "" {
					return "NULL", nil
				}
				return report.Status.SecondsBehindSource, nil
			case "Relay log space":
				return strconv.FormatInt(report.Status.RelayLogSpace, 10), nil
			case "Auto position":
				if report.Status.AutoPosition {
					return "1", nil
				}
				return "0", nil
			}
			return row[1], nil
		}
	}
	return "", fmt.Errorf("unknown replication metric %q", name)
}

func printMetric(w io.Writer, value string, timestamp bool, now time.Time) error {
	if timestamp {
		_, err := fmt.Fprintf(w, "%d,%s\n", now.Unix(), value)
		return err
	}
	_, err := fmt.Fprintln(w, value)
	return err
}

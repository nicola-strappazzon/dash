package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func Repeat(ctx context.Context, opts *Options, run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if opts.Format != "" && opts.Format != "table" && opts.Format != "json" && opts.Format != "csv" {
			return fmt.Errorf("invalid --format %q: choose table, json or csv", opts.Format)
		}
		if cmd.Flags().Changed("metric") && strings.TrimSpace(opts.Metric) == "" {
			return fmt.Errorf("--metric requires a metric name")
		}
		if opts.Timestamp && opts.Format != "csv" {
			// Preserve the original replication --metric --timestamp invocation.
			if opts.Metric != "" && !cmd.Flags().Changed("format") {
				opts.Format = "csv"
			} else {
				return fmt.Errorf("--timestamp requires --format csv")
			}
		}
		out := outputFrom(ctx)
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()

		for {
			out.matched = false
			if opts.Clear && !Machine(ctx) {
				clearTerminal()
			}

			if err := run(cmd, args); err != nil {
				return err
			}

			if opts.Metric != "" && !out.matched {
				return fmt.Errorf("unknown metric %q", opts.Metric)
			}

			if opts.Watch <= 0 {
				return nil
			}

			timer := time.NewTimer(opts.Watch)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}
	}
}

func clearTerminal() {
	fmt.Print("\033[H\033[2J")
}

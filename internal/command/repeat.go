package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
)

func Repeat(ctx context.Context, opts *Options, run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()

		for {
			if opts.Clear {
				clearTerminal()
			}

			if err := run(cmd, args); err != nil {
				return err
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

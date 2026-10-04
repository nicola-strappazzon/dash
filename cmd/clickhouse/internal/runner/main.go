package runner

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/clickhouse"
	"github.com/spf13/cobra"
)

type RenderFunc func(context.Context, *clickhouse.ClickHouse) error

// registry maps a section name (e.g. "uptime") to the RenderFunc that
// renders it, so that sections can be combined and looked up regardless of
// which subcommand the user invoked, e.g. `dash clickhouse uptime`.
var registry = map[string]RenderFunc{}

// Register associates a section name with its RenderFunc. It's called once
// per section, typically from that section's NewCommand.
func Register(section string, render RenderFunc) {
	registry[section] = render
}

func SectionCommand(ctx context.Context, opts *command.Options, section, short string, render RenderFunc) *cobra.Command {
	Register(section, render)

	return &cobra.Command{
		Use:   section,
		Short: short,
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return RenderSections(ctx, opts, append([]string{section}, args...))
		}),
	}
}

// RenderSections validates and renders the given sections, in order, against
// a single ClickHouse connection. It allows any registered sections to be
// combined, e.g. `dash clickhouse uptime replicas`.
func RenderSections(ctx context.Context, opts *command.Options, sections []string) error {
	if err := ValidateSections(sections); err != nil {
		return err
	}

	return WithClickHouse(opts, func(ch *clickhouse.ClickHouse) error {
		defer ch.Close()

		for _, section := range sections {
			render, ok := registry[section]
			if !ok {
				return fmt.Errorf("unknown clickhouse section %q", section)
			}
			if err := render(ctx, ch); err != nil {
				return err
			}
		}

		return nil
	})
}

func ValidateSections(sections []string) error {
	seen := map[string]bool{}
	for _, section := range sections {
		if seen[section] {
			return fmt.Errorf("duplicate clickhouse section %q", section)
		}
		seen[section] = true
	}

	return nil
}

func WithClickHouse(opts *command.Options, run func(*clickhouse.ClickHouse) error) error {
	if opts.ClickHouse.Host == "" {
		return fmt.Errorf("missing ClickHouse host: pass --host")
	}

	ch, err := clickhouse.New(clickhouse.Config{
		Host:               opts.ClickHouse.Host,
		Username:           opts.ClickHouse.Username,
		Password:           opts.ClickHouse.Password,
		TLS:                opts.ClickHouse.TLS,
		InsecureSkipVerify: opts.ClickHouse.InsecureTLS,
	})
	if err != nil {
		return err
	}

	return run(ch)
}

package runner

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/spf13/cobra"
)

type RenderFunc func(context.Context, *elasticsearch.Elasticsearch) error

// registry maps a section name (e.g. "health", "recovery") to the RenderFunc
// that renders it, so that sections can be combined and looked up regardless
// of which subcommand the user invoked, e.g. `dash elasticsearch health recovery`.
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
// a single Elasticsearch connection. It allows any registered sections to be
// combined, e.g. `dash elasticsearch health recovery`.
func RenderSections(ctx context.Context, opts *command.Options, sections []string) error {
	if err := ValidateSections(sections); err != nil {
		return err
	}

	return WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
		for _, section := range sections {
			render, ok := registry[section]
			if !ok {
				return fmt.Errorf("unknown elasticsearch section %q", section)
			}
			if err := render(ctx, es); err != nil {
				return err
			}
		}

		return nil
	})
}

func ValidateSections(sections []string) error {
	seen := map[string]bool{}
	for _, section := range sections {
		if section == "all" && len(sections) > 1 {
			return fmt.Errorf("all cannot be combined with other elasticsearch sections")
		}
		if seen[section] {
			return fmt.Errorf("duplicate elasticsearch section %q", section)
		}
		seen[section] = true
	}

	return nil
}

func WithElasticsearch(opts *command.Options, run func(*elasticsearch.Elasticsearch) error) error {
	if opts.Elasticsearch.Address == "" {
		return fmt.Errorf("missing Elasticsearch address: pass --address")
	}

	es, err := elasticsearch.New(elasticsearch.Config{
		Addresses:          []string{opts.Elasticsearch.Address},
		Username:           opts.Elasticsearch.Username,
		Password:           opts.Elasticsearch.Password,
		InsecureSkipVerify: opts.Elasticsearch.InsecureTLS,
	})
	if err != nil {
		return err
	}

	return run(es)
}

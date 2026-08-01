package elasticsearchcmd

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/dashboard"
	"github.com/spf13/cobra"
)

func sectionCommand(ctx context.Context, opts *command.Options, section, short string) *cobra.Command {
	return &cobra.Command{
		Use:   section,
		Short: short,
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return renderSections(ctx, opts, append([]string{section}, args...))
		}),
	}
}

func renderSections(ctx context.Context, opts *command.Options, sections []string) error {
	if err := validateSections(sections); err != nil {
		return err
	}

	return withElasticsearch(opts, func(es *dashboard.Elasticsearch) error {
		for _, section := range sections {
			switch section {
			case "all":
				if err := renderClusterHealth(ctx, es); err != nil {
					return err
				}
				if err := renderClusterNodes(ctx, es); err != nil {
					return err
				}
				if err := renderIndices(ctx, es); err != nil {
					return err
				}
				if err := renderShards(ctx, es); err != nil {
					return err
				}
				if err := renderRecovery(ctx, es); err != nil {
					return err
				}
				if err := renderSnapshots(ctx, es, opts.ESSnapshotRepository); err != nil {
					return err
				}
				if err := renderSLM(ctx, es); err != nil {
					return err
				}
			case "health":
				if err := renderClusterHealth(ctx, es); err != nil {
					return err
				}
			case "nodes":
				if err := renderClusterNodes(ctx, es); err != nil {
					return err
				}
			case "indices":
				if err := renderIndices(ctx, es); err != nil {
					return err
				}
			case "shards":
				if err := renderShards(ctx, es); err != nil {
					return err
				}
			case "recovery":
				if err := renderRecovery(ctx, es); err != nil {
					return err
				}
			case "snapshots":
				if err := renderSnapshots(ctx, es, opts.ESSnapshotRepository); err != nil {
					return err
				}
			case "slm":
				if err := renderSLM(ctx, es); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown elasticsearch section %q", section)
			}
		}

		return nil
	})
}

func validateSections(sections []string) error {
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

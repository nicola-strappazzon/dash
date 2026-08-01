package elasticsearchcmd

import (
	"context"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func newAllCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return sectionCommand(ctx, opts, "all", "Show all Elasticsearch dashboard sections")
}

package runner

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	elasticsearchdriver "github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/spf13/cobra"
)

type RenderFunc func(context.Context, *elasticsearchdriver.Elasticsearch) error

func SectionCommand(ctx context.Context, opts *command.Options, section, short string, render RenderFunc) *cobra.Command {
	return &cobra.Command{
		Use:   section,
		Short: short,
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return WithElasticsearch(opts, func(es *elasticsearchdriver.Elasticsearch) error {
				return render(ctx, es)
			})
		}),
	}
}

func WithElasticsearch(opts *command.Options, run func(*elasticsearchdriver.Elasticsearch) error) error {
	if opts.Elasticsearch.Address == "" {
		return fmt.Errorf("missing Elasticsearch address: pass --address")
	}

	es, err := elasticsearchdriver.New(elasticsearchdriver.Config{
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

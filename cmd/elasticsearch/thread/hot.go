package thread

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/spf13/cobra"
)

func newHotCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var node string
	var full bool
	cmd := &cobra.Command{
		Use:   "hot",
		Short: "Show what's actually keeping a node's CPU busy right now",
		Long: "Hits _nodes/hot_threads, which returns a snapshot of the busiest JVM threads " +
			"(stack traces) on the target node(s) — use this to find out what's driving high " +
			"CPU (e.g. a Lucene merge, a recovery, a heavy search) instead of just seeing the " +
			"number in `nodes`.\n\n" +
			"By default only prints the per-thread summary line (node, thread name, % CPU); " +
			"pass --full for the complete stack traces underneath.\n\n" +
			"The output looks like a Java exception dump but it isn't one — nothing crashed or " +
			"threw an error; it's just a stack-sample snapshot of where each busy thread " +
			"currently is, taken 10 times 500ms apart by default and grouped by how many of " +
			"those samples landed on the same code path.",
		Args: cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return renderHotThreads(ctx, es, node, full)
			})
		}),
	}
	cmd.Flags().StringVar(&node, "node", "", "only query this node (name, id or IP); default is every node, which can be a lot of output")
	cmd.Flags().BoolVar(&full, "full", false, "show the full stack traces instead of just the per-thread summary line")

	return cmd
}

func renderHotThreads(ctx context.Context, es *elasticsearch.Elasticsearch, node string, full bool) error {
	body, err := es.ThreadHot(ctx, node)
	if err != nil {
		return err
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	// bufio.Scanner's default 64KB buffer can be too small for hot_threads'
	// longest stack trace lines, so give it more room.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	if full {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		return scanner.Err()
	}

	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(trimmed, ":::"): // node identifier, e.g. ::: {elastic02-pre}{...}
			fmt.Println()
			fmt.Println(trimmed)
		case strings.HasPrefix(trimmed, "Hot threads at"): // per-node header (interval, sample count)
			fmt.Println("  " + trimmed)
		case strings.Contains(trimmed, "cpu usage by thread"): // the one summary line per thread
			fmt.Println("  " + trimmed)
		}
	}
	fmt.Println()

	return scanner.Err()
}

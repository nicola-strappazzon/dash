package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
)

// nodeFilterSuffixes are the exclude/include settings (relative to either
// "cluster.routing.allocation." or "index.routing.allocation.") that decide
// whether a node is allowed to receive shards.
var nodeFilterSuffixes = []string{
	"exclude._name",
	"exclude._ip",
	"exclude._host",
	"include._name",
	"include._ip",
	"include._host",
}

// nodeFilterKeys are nodeFilterSuffixes at the cluster scope. "clear" resets
// all of them so a node excluded (or required) by any of the three filter
// types goes back to normal.
var nodeFilterKeys = func() []string {
	keys := make([]string, len(nodeFilterSuffixes))
	for i, suffix := range nodeFilterSuffixes {
		keys[i] = "cluster.routing.allocation." + suffix
	}
	return keys
}()

// setNodeFilter sets a single routing.allocation.{exclude,include}._name
// setting (suffix, e.g. "exclude._name") after warning if the node name
// doesn't match a node currently in the cluster (likely a typo). With index
// set, it's scoped to that index (index.routing.allocation.<suffix>) instead
// of the whole cluster (cluster.routing.allocation.<suffix>).
func setNodeFilter(ctx context.Context, opts *command.Options, suffix, name, index string, apply bool) error {
	return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
		if err := warnIfUnknownNode(ctx, es, name); err != nil {
			return err
		}

		if index != "" {
			if err := warnIfUnknownIndex(ctx, es, index); err != nil {
				return err
			}
			return putIndexSettings(ctx, es, index, map[string]any{"index.routing.allocation." + suffix: name}, apply)
		}

		return putClusterSettings(ctx, es, map[string]any{"cluster.routing.allocation." + suffix: name}, apply)
	})
}

// clearNodeFilters nulls every filter key in nodeFilterKeys, in both scopes,
// so any node excluded or required by a previous exclude-node/include-node
// call becomes eligible for allocation again. With index set, only that
// index's settings are cleared instead of the whole cluster's.
func clearNodeFilters(ctx context.Context, opts *command.Options, index string, apply bool) error {
	return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
		if index != "" {
			if err := warnIfUnknownIndex(ctx, es, index); err != nil {
				return err
			}
			values := map[string]any{}
			for _, suffix := range nodeFilterSuffixes {
				values["index.routing.allocation."+suffix] = nil
			}
			return putIndexSettings(ctx, es, index, values, apply)
		}

		values := map[string]any{}
		for _, key := range nodeFilterKeys {
			values[key] = nil
		}

		return putClusterSettings(ctx, es, values, apply)
	})
}

// putClusterSettings prints what would be sent in dry-run mode, or applies
// values via es.PutClusterSettings and re-renders the settings table to
// confirm what actually took effect.
func putClusterSettings(ctx context.Context, es *elasticsearch.Elasticsearch, values map[string]any, apply bool) error {
	if !apply {
		payload, err := json.MarshalIndent(map[string]any{"persistent": values, "transient": values}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(color.YellowString("Dry run (nothing applied). This is what would be sent to PUT _cluster/settings:"))
		fmt.Println(string(payload))
		fmt.Println(color.YellowString("Re-run with --yes to apply it."))
		return nil
	}

	acknowledged, err := es.PutClusterSettings(ctx, values)
	if err != nil {
		return err
	}

	printAcknowledged(acknowledged)

	return Render(ctx, es)
}

// putIndexSettings prints what would be sent in dry-run mode, or applies
// values via es.PutIndexSettings and prints the value that actually took
// effect on index (which may be a wildcard pattern).
func putIndexSettings(ctx context.Context, es *elasticsearch.Elasticsearch, index string, values map[string]any, apply bool) error {
	if !apply {
		payload, err := json.MarshalIndent(values, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(color.YellowString("Dry run (nothing applied). This is what would be sent to PUT %s/_settings:", index))
		fmt.Println(string(payload))
		fmt.Println(color.YellowString("Re-run with --yes to apply it."))
		return nil
	}

	acknowledged, err := es.PutIndexSettings(ctx, index, values)
	if err != nil {
		return err
	}

	printAcknowledged(acknowledged)

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	return printIndexSettingValues(ctx, es, index, keys)
}

func printAcknowledged(acknowledged bool) {
	if acknowledged {
		fmt.Println(color.GreenString("✓ applied"))
	} else {
		fmt.Println(color.RedString("✗ Elasticsearch didn't acknowledge the change (acknowledged: false)"))
	}
	fmt.Println("")
}

// printIndexSettingValues fetches and prints the current value of keys on
// index (which may be a wildcard pattern), confirming what actually took
// effect — unlike the cluster "Allocation Settings" table, this reflects
// index-level settings like index.refresh_interval.
func printIndexSettingValues(ctx context.Context, es *elasticsearch.Elasticsearch, index string, keys []string) error {
	perIndex, err := es.IndexSettingValues(ctx, index, keys)
	if err != nil {
		return err
	}

	indexNames := make([]string, 0, len(perIndex))
	for name := range perIndex {
		indexNames = append(indexNames, name)
	}
	sort.Strings(indexNames)

	for _, name := range indexNames {
		for _, key := range keys {
			value, ok := perIndex[name][key]
			if !ok {
				fmt.Printf("%s  %s: (unset, back to default)\n", name, key)
				continue
			}
			fmt.Printf("%s  %s: %v\n", name, key, value)
		}
	}

	return nil
}

// warnIfUnknownIndex prints a heads-up (but doesn't block) when index doesn't
// exist in the cluster, since this is the most common way to end up with a
// silently-useless filter.
func warnIfUnknownIndex(ctx context.Context, es *elasticsearch.Elasticsearch, index string) error {
	exists, err := es.IndexExists(ctx, index)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Println(color.YellowString("⚠ couldn't find an index named %q in the cluster.", index))
	}

	return nil
}

// warnIfUnknownNode prints a heads-up (but doesn't block) when name doesn't
// match any node currently in the cluster, since this is the most common way
// to end up with a silently-useless filter.
func warnIfUnknownNode(ctx context.Context, es *elasticsearch.Elasticsearch, name string) error {
	nodes, err := es.Nodes(ctx)
	if err != nil {
		return err
	}

	for _, n := range nodes {
		if n.Name == name {
			return nil
		}
	}

	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.Name
	}
	fmt.Println(color.YellowString("⚠ couldn't find a node named %q in the cluster. Current nodes: %v", name, names))

	return nil
}

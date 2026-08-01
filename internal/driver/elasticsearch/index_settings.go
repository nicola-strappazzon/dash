package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// PutIndexSettings PUTs values to <index>/_settings, where index may be a
// name or a wildcard pattern (e.g. "cr-*"), and reports whether Elasticsearch
// acknowledged the change. Index settings have no persistent/transient
// split, so values are sent as-is.
func (e *Elasticsearch) PutIndexSettings(ctx context.Context, index string, values map[string]any) (bool, error) {
	payload, err := json.Marshal(values)
	if err != nil {
		return false, err
	}

	res, err := e.client.Indices.PutSettings(
		bytes.NewReader(payload),
		e.client.Indices.PutSettings.WithContext(ctx),
		e.client.Indices.PutSettings.WithIndex(index),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("unexpected response from %s/_settings: %s", index, res.String())
	}

	var ack struct {
		Acknowledged bool `json:"acknowledged"`
	}
	if err := json.NewDecoder(res.Body).Decode(&ack); err != nil {
		return false, fmt.Errorf("failed to decode %s/_settings response: %w", index, err)
	}

	return ack.Acknowledged, nil
}

// IndexSettingValues fetches the current value of keys for index (which may
// be a wildcard pattern), keyed by the concrete index name(s) it matched.
func (e *Elasticsearch) IndexSettingValues(ctx context.Context, index string, keys []string) (map[string]map[string]any, error) {
	res, err := e.client.Indices.GetSettings(
		e.client.Indices.GetSettings.WithContext(ctx),
		e.client.Indices.GetSettings.WithIndex(index),
		e.client.Indices.GetSettings.WithName(keys...),
		e.client.Indices.GetSettings.WithFlatSettings(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from %s/_settings: %s", index, res.String())
	}

	var perIndex map[string]struct {
		Settings map[string]any `json:"settings"`
	}
	if err := json.NewDecoder(res.Body).Decode(&perIndex); err != nil {
		return nil, fmt.Errorf("failed to decode %s/_settings response: %w", index, err)
	}

	values := make(map[string]map[string]any, len(perIndex))
	for name, entry := range perIndex {
		values[name] = entry.Settings
	}

	return values, nil
}

// IndexExists reports whether index exists in the cluster.
func (e *Elasticsearch) IndexExists(ctx context.Context, index string) (bool, error) {
	res, err := e.client.Indices.Exists(
		[]string{index},
		e.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return false, nil
	}
	if res.IsError() {
		return false, fmt.Errorf("unexpected response from %s (exists check): %s", index, res.String())
	}

	return true, nil
}

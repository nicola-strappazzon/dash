package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type ClusterSettings struct {
	Persistent map[string]any `json:"persistent"`
	Transient  map[string]any `json:"transient"`
	Defaults   map[string]any `json:"defaults"`
}

// ClusterSettings fetches _cluster/settings (flat, including defaults).
func (e *Elasticsearch) ClusterSettings(ctx context.Context) (ClusterSettings, error) {
	var cs ClusterSettings

	res, err := e.client.Cluster.GetSettings(
		e.client.Cluster.GetSettings.WithContext(ctx),
		e.client.Cluster.GetSettings.WithFlatSettings(true),
		e.client.Cluster.GetSettings.WithIncludeDefaults(true),
	)
	if err != nil {
		return cs, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return cs, fmt.Errorf("unexpected response from _cluster/settings: %s", res.String())
	}

	if err := json.NewDecoder(res.Body).Decode(&cs); err != nil {
		return cs, fmt.Errorf("failed to decode _cluster/settings: %w", err)
	}

	return cs, nil
}

// PutClusterSettings sets values in both the persistent and transient scopes
// (a nil value clears that key in either scope, whichever it was stored in)
// and reports whether Elasticsearch acknowledged the change.
func (e *Elasticsearch) PutClusterSettings(ctx context.Context, values map[string]any) (bool, error) {
	body := map[string]any{
		"persistent": values,
		"transient":  values,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return false, err
	}

	res, err := e.client.Cluster.PutSettings(
		bytes.NewReader(payload),
		e.client.Cluster.PutSettings.WithContext(ctx),
		e.client.Cluster.PutSettings.WithFlatSettings(true),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("unexpected response from _cluster/settings: %s", res.String())
	}

	var ack struct {
		Acknowledged bool `json:"acknowledged"`
	}
	if err := json.NewDecoder(res.Body).Decode(&ack); err != nil {
		return false, fmt.Errorf("failed to decode _cluster/settings response: %w", err)
	}

	return ack.Acknowledged, nil
}

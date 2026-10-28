package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

type AllocationExplanation struct {
	Index        string `json:"index"`
	Shard        int    `json:"shard"`
	Primary      bool   `json:"primary"`
	CurrentState string `json:"current_state"`
	Explanation  string `json:"explanation"`
	CurrentNode  *struct {
		Name string `json:"name"`
	} `json:"current_node"`
	UnassignedInfo *struct {
		Reason  string `json:"reason"`
		At      string `json:"at"`
		Details string `json:"details"`
	} `json:"unassigned_info"`
	CanAllocate             string               `json:"can_allocate"`
	AllocateExplanation     string               `json:"allocate_explanation"`
	CanRemain               string               `json:"can_remain_on_current_node"`
	CanRemainDecisions      []AllocationDecision `json:"can_remain_decisions"`
	CanMove                 string               `json:"can_move_to_other_node"`
	MoveExplanation         string               `json:"move_explanation"`
	CanRebalanceCluster     string               `json:"can_rebalance_cluster"`
	CanRebalanceDecisions   []AllocationDecision `json:"can_rebalance_cluster_decisions"`
	CanRebalanceToOtherNode string               `json:"can_rebalance_to_other_node"`
	RebalanceExplanation    string               `json:"rebalance_explanation"`
	RemainingDelayMillis    int64                `json:"remaining_delay_in_millis"`
	NodeDecisions           []struct {
		Name     string               `json:"node_name"`
		Decision string               `json:"node_decision"`
		Deciders []AllocationDecision `json:"deciders"`
	} `json:"node_allocation_decisions"`
}

type AllocationDecision struct {
	Decider     string `json:"decider"`
	Decision    string `json:"decision"`
	Explanation string `json:"explanation"`
}

// ExplainAllocation only diagnoses the specified copy; it does not change allocation.
func (e *Elasticsearch) ExplainAllocation(ctx context.Context, index string, shard int, primary bool, node string) (*AllocationExplanation, error) {
	body, _ := json.Marshal(struct {
		Index       string `json:"index"`
		Shard       int    `json:"shard"`
		Primary     bool   `json:"primary"`
		CurrentNode string `json:"current_node,omitempty"`
	}{index, shard, primary, node})
	res, err := e.client.Cluster.AllocationExplain(
		e.client.Cluster.AllocationExplain.WithContext(ctx),
		e.client.Cluster.AllocationExplain.WithBody(bytes.NewReader(body)),
		e.client.Cluster.AllocationExplain.WithIncludeYesDecisions(false),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cluster/allocation/explain: %s", res.String())
	}
	var explanation AllocationExplanation
	if err := json.NewDecoder(res.Body).Decode(&explanation); err != nil {
		return nil, fmt.Errorf("failed to decode allocation explanation: %w", err)
	}
	sort.Slice(explanation.NodeDecisions, func(i, j int) bool {
		return explanation.NodeDecisions[i].Name < explanation.NodeDecisions[j].Name
	})
	return &explanation, nil
}

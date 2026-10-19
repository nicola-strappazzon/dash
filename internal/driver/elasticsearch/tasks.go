package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

type Task struct {
	TaskID           string      `json:"-"`
	ParentTaskID     string      `json:"parent_task_id"`
	Node             string      `json:"-"`
	Action           string      `json:"action"`
	Description      string      `json:"description"`
	RunningTimeNanos int64       `json:"running_time_in_nanos"`
	Status           *TaskStatus `json:"status"`
}

type TaskStatus struct {
	Total            *int64 `json:"total"`
	Created          int64  `json:"created"`
	Updated          int64  `json:"updated"`
	Deleted          int64  `json:"deleted"`
	Noops            int64  `json:"noops"`
	VersionConflicts int64  `json:"version_conflicts"`
}

// TaskDetail preserves all fields, including action-specific status and results.
func (e *Elasticsearch) TaskDetail(ctx context.Context, id string) (json.RawMessage, error) {
	res, err := e.client.Tasks.Get(id, e.client.Tasks.Get.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _tasks/%s: %s", id, res.String())
	}
	var detail json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("failed to decode _tasks/%s: %w", id, err)
	}
	return detail, nil
}

// Tasks lists active tasks, longest running first, with node names and detailed status.
func (e *Elasticsearch) Tasks(ctx context.Context) ([]Task, error) {
	res, err := e.client.Tasks.List(
		e.client.Tasks.List.WithContext(ctx),
		e.client.Tasks.List.WithDetailed(true),
		e.client.Tasks.List.WithGroupBy("nodes"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _tasks: %s", res.String())
	}
	var response struct {
		Nodes map[string]struct {
			Name  string          `json:"name"`
			Tasks map[string]Task `json:"tasks"`
		} `json:"nodes"`
		NodeFailures []json.RawMessage `json:"node_failures"`
		TaskFailures []json.RawMessage `json:"task_failures"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode _tasks: %w", err)
	}
	if len(response.NodeFailures)+len(response.TaskFailures) > 0 {
		return nil, fmt.Errorf("incomplete _tasks response: %d node failures, %d task failures", len(response.NodeFailures), len(response.TaskFailures))
	}
	var tasks []Task
	for _, node := range response.Nodes {
		for id, task := range node.Tasks {
			task.TaskID = id
			task.Node = node.Name
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].RunningTimeNanos == tasks[j].RunningTimeNanos {
			return tasks[i].TaskID < tasks[j].TaskID
		}
		return tasks[i].RunningTimeNanos > tasks[j].RunningTimeNanos
	})
	return tasks, nil
}

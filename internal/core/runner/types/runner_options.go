package types

import "encoding/json"

type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RunnerOptions struct {
	EnableNetwork bool `json:"enable_network"`
	TaskID        string `json:"task_id"`
}

func (r *RunnerOptions) Json() string {
	b, _ := json.Marshal(r)
	return string(b)
}

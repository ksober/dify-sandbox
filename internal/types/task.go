package types

import "time"

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待执行
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
)

// Task 任务信息
type Task struct {
	ID        string     `json:"id"`                   // 任务ID
	Status    TaskStatus `json:"status"`               // 任务状态
	CreatedAt time.Time  `json:"created_at"`           // 创建时间
	StartedAt *time.Time `json:"started_at,omitempty"` // 开始执行时间
	EndedAt   *time.Time `json:"ended_at,omitempty"`   // 结束时间

	// 执行结果
	Stdout string `json:"stdout,omitempty"` // 标准输出
	Stderr string `json:"stderr,omitempty"` // 标准错误
	Error  string `json:"error,omitempty"`  // 执行错误信息
}

// SubmitTaskResponse 提交任务响应
type SubmitTaskResponse struct {
	TaskID string `json:"task_id"` // 任务ID
}

// QueryTaskResponse 查询任务响应
type QueryTaskResponse struct {
	Task *Task `json:"task"`
}

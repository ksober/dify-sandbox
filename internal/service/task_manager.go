package service

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/langgenius/dify-sandbox/internal/types"
)

// TaskManager 任务管理器
type TaskManager struct {
	tasks map[string]*types.Task
	mu    sync.RWMutex
}

var globalTaskManager *TaskManager
var taskManagerOnce sync.Once

// GetTaskManager 获取全局任务管理器单例
func GetTaskManager() *TaskManager {
	taskManagerOnce.Do(func() {
		globalTaskManager = &TaskManager{
			tasks: make(map[string]*types.Task),
		}
	})
	return globalTaskManager
}

// CreateTask 创建新任务
func (tm *TaskManager) CreateTask() *types.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	taskID := uuid.New().String()
	task := &types.Task{
		ID:        taskID,
		Status:    types.TaskStatusPending,
		CreatedAt: time.Now(),
	}
	tm.tasks[taskID] = task
	return task
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(taskID string) (*types.Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	return task, exists
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(taskID string, status types.TaskStatus) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return false
	}

	task.Status = status
	now := time.Now()

	switch status {
	case types.TaskStatusRunning:
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
	case types.TaskStatusCompleted, types.TaskStatusFailed:
		if task.EndedAt == nil {
			task.EndedAt = &now
		}
	}

	return true
}

// UpdateTaskResult 更新任务执行结果
func (tm *TaskManager) UpdateTaskResult(taskID string, stdout, stderr, errorMsg string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return false
	}

	task.Stdout = stdout
	task.Stderr = stderr
	task.Error = errorMsg

	return true
}

// DeleteTask 删除任务（可选，用于清理）
func (tm *TaskManager) DeleteTask(taskID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tasks, taskID)
}

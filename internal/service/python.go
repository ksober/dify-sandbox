package service

import (
	"time"

	"github.com/langgenius/dify-sandbox/internal/core/runner/python"
	runner_types "github.com/langgenius/dify-sandbox/internal/core/runner/types"
	"github.com/langgenius/dify-sandbox/internal/static"
	"github.com/langgenius/dify-sandbox/internal/types"
)

func RunPython3Code(code string, preload string, options *runner_types.RunnerOptions) *types.DifySandboxResponse {
	if err := checkOptions(options); err != nil {
		return types.ErrorResponse(-400, err.Error())
	}

	if !static.GetDifySandboxGlobalConfigurations().EnablePreload {
		preload = ""
	}

	timeout := time.Duration(
		static.GetDifySandboxGlobalConfigurations().WorkerTimeout * int(time.Second),
	)

	runner := python.PythonRunner{}
	stdout, stderr, done, err := runner.Run(
		code, timeout, nil, preload, options,
	)
	if err != nil {
		return types.ErrorResponse(-500, err.Error())
	}

	stdout_str := ""
	stderr_str := ""

	defer close(done)
	defer close(stdout)
	defer close(stderr)

	for {
		select {
		case <-done:
			return types.SuccessResponse(&RunCodeResponse{
				Stdout: stdout_str,
				Stderr: stderr_str,
			})
		case out := <-stdout:
			stdout_str += string(out)
		case err := <-stderr:
			stderr_str += string(err)
		}
	}
}

type ListDependenciesResponse struct {
	Dependencies []runner_types.Dependency `json:"dependencies"`
}

func ListPython3Dependencies() *types.DifySandboxResponse {
	return types.SuccessResponse(&ListDependenciesResponse{
		Dependencies: python.ListDependencies(),
	})
}

type RefreshDependenciesResponse struct {
	Dependencies []runner_types.Dependency `json:"dependencies"`
}

func RefreshPython3Dependencies() *types.DifySandboxResponse {
	return types.SuccessResponse(&RefreshDependenciesResponse{
		Dependencies: python.RefreshDependencies(),
	})
}

type UpdateDependenciesResponse struct{}

func UpdateDependencies() *types.DifySandboxResponse {
	err := python.PreparePythonDependenciesEnv()
	if err != nil {
		return types.ErrorResponse(-500, err.Error())
	}

	return types.SuccessResponse(&UpdateDependenciesResponse{})
}

// SubmitPython3Task 提交Python3代码执行任务（异步）
func SubmitPython3Task(code string, preload string, options *runner_types.RunnerOptions) *types.DifySandboxResponse {
	if err := checkOptions(options); err != nil {
		return types.ErrorResponse(-400, err.Error())
	}

	// 创建任务
	taskManager := GetTaskManager()
	task := taskManager.CreateTask()

	// 异步执行
	go runPython3TaskAsync(task.ID, code, preload, options)

	return types.SuccessResponse(&types.SubmitTaskResponse{
		TaskID: task.ID,
	})
}

// runPython3TaskAsync 异步执行Python3代码
func runPython3TaskAsync(taskID string, code string, preload string, options *runner_types.RunnerOptions) {
	taskManager := GetTaskManager()

	// 更新状态为运行中
	taskManager.UpdateTaskStatus(taskID, types.TaskStatusRunning)

	if !static.GetDifySandboxGlobalConfigurations().EnablePreload {
		preload = ""
	}

	timeout := time.Duration(
		static.GetDifySandboxGlobalConfigurations().WorkerTimeout * int(time.Second),
	)

	runner := python.PythonRunner{}
	stdout, stderr, done, err := runner.Run(
		code, timeout, nil, preload, options,
	)

	if err != nil {
		taskManager.UpdateTaskStatus(taskID, types.TaskStatusFailed)
		taskManager.UpdateTaskResult(taskID, "", "", err.Error())
		return
	}

	stdout_str := ""
	stderr_str := ""

	defer close(done)
	defer close(stdout)
	defer close(stderr)

	for {
		select {
		case <-done:
			taskManager.UpdateTaskStatus(taskID, types.TaskStatusCompleted)
			taskManager.UpdateTaskResult(taskID, stdout_str, stderr_str, "")
			return
		case out := <-stdout:
			stdout_str += string(out)
		case err := <-stderr:
			stderr_str += string(err)
		}
	}
}

// QueryTask 查询任务状态
func QueryTask(taskID string) *types.DifySandboxResponse {
	taskManager := GetTaskManager()
	task, exists := taskManager.GetTask(taskID)
	if !exists {
		return types.ErrorResponse(-404, "task not found")
	}

	return types.SuccessResponse(&types.QueryTaskResponse{
		Task: task,
	})
}

package service

import (
	"time"

	"github.com/langgenius/dify-sandbox/internal/core/runner/nodejs"
	runner_types "github.com/langgenius/dify-sandbox/internal/core/runner/types"
	"github.com/langgenius/dify-sandbox/internal/static"
	"github.com/langgenius/dify-sandbox/internal/types"
)

func RunNodeJsCode(code string, preload string, options *runner_types.RunnerOptions) *types.DifySandboxResponse {
	if err := checkOptions(options); err != nil {
		return types.ErrorResponse(-400, err.Error())
	}

	if !static.GetDifySandboxGlobalConfigurations().EnablePreload {
		preload = ""
	}

	timeout := time.Duration(
		static.GetDifySandboxGlobalConfigurations().WorkerTimeout * int(time.Second),
	)

	runner := nodejs.NodeJsRunner{}
	stdout, stderr, done, err := runner.Run(code, timeout, nil, preload, options)
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

// SubmitNodeJsTask 提交NodeJs代码执行任务（异步）
func SubmitNodeJsTask(code string, preload string, options *runner_types.RunnerOptions) *types.DifySandboxResponse {
	if err := checkOptions(options); err != nil {
		return types.ErrorResponse(-400, err.Error())
	}

	// 创建任务
	taskManager := GetTaskManager()
	task := taskManager.CreateTask()

	// 异步执行
	go runNodeJsTaskAsync(task.ID, code, preload, options)

	return types.SuccessResponse(&types.SubmitTaskResponse{
		TaskID: task.ID,
	})
}

// runNodeJsTaskAsync 异步执行NodeJs代码
func runNodeJsTaskAsync(taskID string, code string, preload string, options *runner_types.RunnerOptions) {
	taskManager := GetTaskManager()

	// 更新状态为运行中
	taskManager.UpdateTaskStatus(taskID, types.TaskStatusRunning)

	if !static.GetDifySandboxGlobalConfigurations().EnablePreload {
		preload = ""
	}

	timeout := time.Duration(
		static.GetDifySandboxGlobalConfigurations().WorkerTimeout * int(time.Second),
	)

	runner := nodejs.NodeJsRunner{}
	stdout, stderr, done, err := runner.Run(code, timeout, nil, preload, options)

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

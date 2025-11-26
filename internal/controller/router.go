package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/langgenius/dify-sandbox/internal/middleware"
	"github.com/langgenius/dify-sandbox/internal/static"
)

func Setup(Router *gin.Engine) {
	PublicGroup := Router.Group("")
	PrivateGroup := Router.Group("/v1/sandbox/")

	PrivateGroup.Use(middleware.Auth())

	{
		// health check
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
	}

	InitRunRouter(PrivateGroup)
	InitDependencyRouter(PrivateGroup)
}

func InitDependencyRouter(Router *gin.RouterGroup) {
	dependencyRouter := Router.Group("dependencies")
	{
		dependencyRouter.GET("", GetDependencies)
		dependencyRouter.POST("update", UpdateDependencies)
		dependencyRouter.GET("refresh", RefreshDependencies)
	}
}

func InitRunRouter(Router *gin.RouterGroup) {
	runRouter := Router.Group("")
	{
		// 同步执行接口（保留向后兼容）
		runRouter.POST(
			"run",
			middleware.MaxRequest(static.GetDifySandboxGlobalConfigurations().MaxRequests),
			middleware.MaxWorker(static.GetDifySandboxGlobalConfigurations().MaxWorkers),
			RunSandboxController,
		)
		// 异步执行接口
		taskRouter := Router.Group("tasks")
		{
			taskRouter.POST(
				"",
				middleware.MaxRequest(static.GetDifySandboxGlobalConfigurations().MaxRequests),
				middleware.MaxWorker(static.GetDifySandboxGlobalConfigurations().MaxWorkers),
				SubmitTaskController,
			)
			taskRouter.GET(":task_id", QueryTaskController)
		}
	}
}

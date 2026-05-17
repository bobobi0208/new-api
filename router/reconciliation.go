package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterReconciliationRoutes 在传入的路由组下挂载所有"上游对账"相关接口。
//
// 单独抽出本函数的目的是：让 api-router.go 只需要加一行调用，
// 后续路由增减都在本文件内完成，避免触发现有 api-router.go 的合并冲突。
func RegisterReconciliationRoutes(apiRouter *gin.RouterGroup) {
	group := apiRouter.Group("/reconciliation")
	group.Use(middleware.AdminAuth())
	{
		group.GET("/records", controller.GetReconciliationRecords)
		group.GET("/records/:id", controller.GetReconciliationRecord)
		group.GET("/alerts", controller.GetReconciliationAlerts)
		group.GET("/latest", controller.GetReconciliationLatestPerChannel)
		group.POST("/trigger", controller.TriggerReconciliation)
		group.GET("/settings", controller.GetReconciliationSettings)
		group.PUT("/settings", controller.UpdateReconciliationSettings)
		group.GET("/channels", controller.ListReconciliationChannelConfigs)
		group.POST("/channels", controller.UpsertReconciliationChannelConfigController)
		group.DELETE("/channels/:channel_id", controller.DeleteReconciliationChannelConfigController)
	}
}

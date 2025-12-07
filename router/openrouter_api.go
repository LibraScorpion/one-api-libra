//go:build openrouter
// +build openrouter

package router

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
)

// SetOpenRouterAPIRoute 设置OpenRouter风格的API路由
func SetOpenRouterAPIRoute(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.CORS())

	// ========== 认证相关 API ==========
	authRouter := apiRouter.Group("/auth")
	{
		// Google OAuth 登录
		authRouter.GET("/google", controller.GoogleLoginHandler)                  // 获取Google登录URL
		authRouter.GET("/google/callback", controller.GoogleCallbackHandler)      // Google OAuth回调
		authRouter.POST("/google/token", controller.GoogleTokenLoginHandler)      // 前端直接传递Google Token
		authRouter.POST("/google/link", middleware.UserAuth(), controller.LinkGoogleAccount)     // 关联Google账号
		authRouter.DELETE("/google/link", middleware.UserAuth(), controller.UnlinkGoogleAccount) // 取消关联Google账号

		// 传统登录注册
		authRouter.POST("/register", controller.Register)
		authRouter.POST("/login", controller.Login)
		authRouter.POST("/logout", middleware.UserAuth(), controller.Logout)
		authRouter.POST("/refresh", controller.RefreshToken)

		// 用户信息
		authRouter.GET("/user", middleware.UserAuth(), controller.GetSelf)
		authRouter.PUT("/user", middleware.UserAuth(), controller.UpdateSelf)
	}

	// ========== 模型相关 API ==========
	modelRouter := apiRouter.Group("/models")
	{
		modelRouter.GET("", controller.GetModels)                     // 获取模型列表
		modelRouter.GET("/:id", controller.GetModelDetail)            // 获取模型详情
		modelRouter.GET("/search", controller.SearchModels)           // 搜索模型
		modelRouter.GET("/rankings", controller.GetModelRankings)     // 获取模型排行
		modelRouter.POST("/compare", controller.CompareModels)        // 模型对比
		modelRouter.GET("/providers", controller.GetModelProviders)   // 获取供应商列表
		modelRouter.GET("/capabilities", controller.GetCapabilities)  // 获取能力列表
	}

	// ========== 额度管理 API ==========
	creditRouter := apiRouter.Group("/credits")
	creditRouter.Use(middleware.UserAuth())
	{
		creditRouter.GET("/balance", controller.GetBalance)                    // 获取余额
		creditRouter.POST("/recharge", controller.CreateRechargeOrder)         // 创建充值订单
		creditRouter.GET("/transactions", controller.GetCreditTransactions)    // 获取交易记录
		creditRouter.POST("/alerts", controller.SetBalanceAlert)               // 设置余额预警
		creditRouter.GET("/alerts", controller.GetBalanceAlerts)               // 获取预警设置
		creditRouter.GET("/usage/summary", controller.GetUsageSummary)         // 获取用量汇总
		creditRouter.POST("/auto-recharge", controller.SetAutoRecharge)        // 设置自动充值
	}

	// ========== API Key 管理 ==========
	keyRouter := apiRouter.Group("/keys")
	keyRouter.Use(middleware.UserAuth())
	{
		keyRouter.GET("", controller.GetAPIKeys)                      // 获取Key列表
		keyRouter.POST("", controller.CreateAPIKey)                   // 创建新Key
		keyRouter.GET("/:id", controller.GetAPIKeyDetail)             // 获取Key详情
		keyRouter.PUT("/:id", controller.UpdateAPIKey)                // 更新Key设置
		keyRouter.DELETE("/:id", controller.DeleteAPIKey)             // 删除Key
		keyRouter.POST("/:id/rotate", controller.RotateAPIKey)        // 轮换Key
		keyRouter.GET("/:id/logs", controller.GetAPIKeyLogs)          // 获取Key使用日志
		keyRouter.POST("/:id/whitelist", controller.SetIPWhitelist)   // 设置IP白名单
		keyRouter.POST("/:id/ratelimit", controller.SetRateLimit)     // 设置速率限制
	}

	// ========== 活动记录 API ==========
	activityRouter := apiRouter.Group("/activity")
	activityRouter.Use(middleware.UserAuth())
	{
		activityRouter.GET("/logs", controller.GetActivityLogs)           // 获取活动日志
		activityRouter.GET("/stats", controller.GetActivityStats)         // 获取统计数据
		activityRouter.GET("/usage", controller.GetUsageAnalytics)        // 获取用量分析
		activityRouter.POST("/export", controller.ExportActivityData)     // 导出数据
		activityRouter.GET("/realtime", controller.GetRealtimeActivity)   // 实时活动（WebSocket）
	}

	// ========== 设置相关 API ==========
	settingsRouter := apiRouter.Group("/settings")
	settingsRouter.Use(middleware.UserAuth())
	{
		// 偏好设置
		settingsRouter.GET("/preferences", controller.GetPreferences)
		settingsRouter.PUT("/preferences", controller.UpdatePreferences)

		// 个人资料
		settingsRouter.GET("/profile", controller.GetProfile)
		settingsRouter.PUT("/profile", controller.UpdateProfile)
		settingsRouter.POST("/avatar", controller.UploadAvatar)

		// 安全设置
		settingsRouter.POST("/password", controller.ChangePassword)
		settingsRouter.POST("/2fa/enable", controller.Enable2FA)
		settingsRouter.POST("/2fa/disable", controller.Disable2FA)
		settingsRouter.GET("/sessions", controller.GetSessions)
		settingsRouter.DELETE("/sessions/:id", controller.RevokeSession)

		// 通知设置
		settingsRouter.GET("/notifications", controller.GetNotificationSettings)
		settingsRouter.PUT("/notifications", controller.UpdateNotificationSettings)
		settingsRouter.POST("/webhook", controller.TestWebhook)
	}

	// ========== 支付相关 API ==========
	paymentRouter := apiRouter.Group("/payments")
	paymentRouter.Use(middleware.UserAuth())
	{
		paymentRouter.GET("/methods", controller.GetPaymentMethods)        // 获取支付方式
		paymentRouter.POST("/methods", controller.AddPaymentMethod)        // 添加支付方式
		paymentRouter.DELETE("/methods/:id", controller.DeletePaymentMethod) // 删除支付方式
		paymentRouter.GET("/orders", controller.GetPaymentOrders)          // 获取支付订单
		paymentRouter.GET("/orders/:id", controller.GetOrderDetail)        // 获取订单详情
		paymentRouter.POST("/orders/:id/cancel", controller.CancelOrder)   // 取消订单
		paymentRouter.GET("/invoices", controller.GetInvoices)             // 获取发票列表
		paymentRouter.GET("/invoices/:id/download", controller.DownloadInvoice) // 下载发票
	}

	// ========== 排行榜 API ==========
	rankingRouter := apiRouter.Group("/rankings")
	{
		rankingRouter.GET("/models", controller.GetModelRankings)       // 模型排行榜
		rankingRouter.GET("/providers", controller.GetProviderRankings) // 供应商排行榜
		rankingRouter.GET("/usage", controller.GetUsageRankings)        // 使用量排行榜
		rankingRouter.GET("/trending", controller.GetTrendingModels)    // 热门趋势
		rankingRouter.GET("/community", controller.GetCommunityRatings) // 社区评分
	}

	// ========== 管理员 API ==========
	adminRouter := apiRouter.Group("/admin")
	adminRouter.Use(middleware.UserAuth(), middleware.AdminAuth())
	{
		// 用户管理
		adminRouter.GET("/users", controller.GetUsers)
		adminRouter.GET("/users/:id", controller.GetUserDetail)
		adminRouter.PUT("/users/:id", controller.UpdateUser)
		adminRouter.DELETE("/users/:id", controller.DeleteUser)
		adminRouter.POST("/users/:id/suspend", controller.SuspendUser)
		adminRouter.POST("/users/:id/activate", controller.ActivateUser)

		// 渠道管理
		adminRouter.GET("/channels", controller.GetChannels)
		adminRouter.POST("/channels", controller.CreateChannel)
		adminRouter.PUT("/channels/:id", controller.UpdateChannel)
		adminRouter.DELETE("/channels/:id", controller.DeleteChannel)
		adminRouter.POST("/channels/:id/test", controller.TestChannel)

		// 模型管理
		adminRouter.POST("/models", controller.CreateModel)
		adminRouter.PUT("/models/:id", controller.UpdateModel)
		adminRouter.DELETE("/models/:id", controller.DeleteModel)
		adminRouter.POST("/models/sync", controller.SyncModels)

		// 系统监控
		adminRouter.GET("/system/stats", controller.GetSystemStats)
		adminRouter.GET("/system/health", controller.GetSystemHealth)
		adminRouter.GET("/system/logs", controller.GetSystemLogs)
		adminRouter.POST("/system/cache/clear", controller.ClearCache)
	}

	// ========== 公共 API ==========
	publicRouter := apiRouter.Group("/public")
	{
		publicRouter.GET("/status", controller.GetSystemStatus)        // 系统状态
		publicRouter.GET("/models", controller.GetPublicModels)        // 公开模型列表
		publicRouter.GET("/pricing", controller.GetPricingInfo)        // 价格信息
		publicRouter.GET("/docs", controller.GetAPIDocumentation)      // API文档
	}

	// ========== WebSocket 端点 ==========
	wsRouter := apiRouter.Group("/ws")
	wsRouter.Use(middleware.UserAuth())
	{
		wsRouter.GET("/activity", controller.WebSocketActivity)    // 实时活动流
		wsRouter.GET("/notifications", controller.WebSocketNotifications) // 实时通知
	}
}

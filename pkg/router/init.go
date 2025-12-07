package router

import (
	"context"
	"time"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

var (
	// GlobalEngine 全局路由引擎实例
	GlobalEngine *Engine
)

// InitRouter 初始化路由引擎
func InitRouter() error {
	logger.SysLog("Initializing smart router engine...")

	// 创建路由引擎
	GlobalEngine = NewEngine()

	// 数据库迁移
	err := model.DB.AutoMigrate(&model.ChannelHealth{})
	if err != nil {
		logger.SysError("Failed to migrate channel_health table: " + err.Error())
		return err
	}

	// 预加载缓存
	ctx := context.Background()
	go func() {
		time.Sleep(3 * time.Second) // 等待数据库完全就绪
		err := GlobalEngine.cache.Preload(ctx)
		if err != nil {
			logger.SysError("Failed to preload router cache: " + err.Error())
		}
	}()

	// 启动健康检查任务
	go StartHealthCheckTask()

	logger.SysLog("Smart router engine initialized successfully")
	return nil
}

// StartHealthCheckTask 启动健康检查任务
func StartHealthCheckTask() {
	logger.SysLog("Starting health check task...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		performHealthCheck()
	}
}

// performHealthCheck 执行健康检查
func performHealthCheck() {
	ctx := context.Background()

	// 获取所有启用的渠道
	channels, err := model.GetAllChannels(0, 0, "all")
	if err != nil {
		logger.SysError("Failed to get all channels for health check: " + err.Error())
		return
	}

	for _, channel := range channels {
		if channel.Status != model.ChannelStatusEnabled {
			continue
		}

		// 这里简化处理：只检查渠道是否被标记为不健康
		// 实际生产环境应该发送真实的测试请求
		health, err := model.GetChannelHealth(channel.Id)
		if err != nil {
			logger.SysError("Failed to get health for channel " + string(rune(channel.Id)) + ": " + err.Error())
			continue
		}

		// 如果超过 5 分钟没有更新，标记为 unknown
		if time.Now().Unix()-health.UpdatedAt > 300 {
			logger.Debugf(ctx, "Channel #%d has not been updated for 5 minutes, marking as unknown", channel.Id)
			health.Status = "unknown"
			_ = model.UpdateChannelHealth(health)
		}

		// 如果渠道不健康且连续失败次数 >= 5，自动禁用
		if health.Status == "unhealthy" && health.ConsecutiveFails >= 5 {
			logger.SysError("Channel " + string(rune(channel.Id)) + " has too many failures, auto-disabling")
			model.UpdateChannelStatusById(channel.Id, model.ChannelStatusAutoDisabled)
		}
	}
}

// GetGlobalEngine 获取全局路由引擎
func GetGlobalEngine() *Engine {
	return GlobalEngine
}

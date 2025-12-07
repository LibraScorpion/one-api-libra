package router

import (
	"context"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// Engine 路由引擎
type Engine struct {
	strategyFactory *StrategyFactory
	cache           *ChannelCache
	defaultStrategy RouterStrategy
}

// NewEngine 创建路由引擎
func NewEngine() *Engine {
	return &Engine{
		strategyFactory: NewStrategyFactory(),
		cache:           NewChannelCache(),
		defaultStrategy: StrategyPriority, // 默认使用优先级策略
	}
}

// SelectChannel 选择渠道
func (e *Engine) SelectChannel(ctx context.Context, req *SelectRequest) (*SelectResult, error) {
	startTime := time.Now()

	// 1. 获取候选渠道
	channels, err := e.getCandidateChannels(ctx, req.Group, req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate channels: %w", err)
	}

	if len(channels) == 0 {
		return nil, fmt.Errorf("no available channels for group=%s, model=%s", req.Group, req.Model)
	}

	logger.Debugf(ctx, "Found %d candidate channels for group=%s, model=%s", len(channels), req.Group, req.Model)

	// 2. 过滤不健康的渠道
	healthyChannels := e.filterHealthyChannels(channels)
	if len(healthyChannels) == 0 {
		// 如果没有健康的渠道，尝试使用 unknown 状态的渠道
		logger.SysLog(fmt.Sprintf("No healthy channels, trying unknown status channels for group=%s, model=%s", req.Group, req.Model))
		healthyChannels = e.filterUnknownChannels(channels)
		if len(healthyChannels) == 0 {
			return nil, fmt.Errorf("no healthy channels available for group=%s, model=%s", req.Group, req.Model)
		}
	}

	// 3. 获取渠道指标
	channelsWithMetrics := e.loadChannelMetrics(ctx, healthyChannels)

	// 4. 选择策略
	strategy := req.Strategy
	if strategy == "" {
		strategy = e.defaultStrategy
	}
	strategyImpl := e.strategyFactory.GetStrategy(strategy)

	// 5. 执行选择
	selectedChannel := strategyImpl.Select(channelsWithMetrics)
	if selectedChannel == nil {
		return nil, fmt.Errorf("strategy %s failed to select channel", strategy)
	}

	// 6. 构建结果
	result := &SelectResult{
		Channel:        selectedChannel,
		Reason:         fmt.Sprintf("Selected by %s strategy", strategyImpl.Name()),
		CandidateCount: len(channels),
		DecisionTime:   time.Since(startTime),
	}

	logger.Debugf(ctx, "Selected channel #%d for group=%s, model=%s, strategy=%s, decision_time=%v",
		selectedChannel.Id, req.Group, req.Model, strategyImpl.Name(), result.DecisionTime)

	// 7. 记录决策日志（异步）
	go e.logDecision(req, result, "success", 0)

	return result, nil
}

// getCandidateChannels 获取候选渠道
func (e *Engine) getCandidateChannels(ctx context.Context, group string, modelName string) ([]*model.Channel, error) {
	// 先尝试从缓存获取
	channels, err := e.cache.GetChannels(ctx, group, modelName)
	if err != nil {
		logger.SysError(fmt.Sprintf("Failed to get channels from cache: %v", err))
		// 缓存失败，直接查询数据库
		return model.GetSatisfiedChannels(group, modelName)
	}
	return channels, nil
}

// filterHealthyChannels 过滤健康的渠道
func (e *Engine) filterHealthyChannels(channels []*model.Channel) []*model.Channel {
	var healthy []*model.Channel
	for _, ch := range channels {
		if ch.Status == model.ChannelStatusEnabled && model.IsChannelHealthy(ch.Id) {
			healthy = append(healthy, ch)
		}
	}
	return healthy
}

// filterUnknownChannels 过滤 unknown 状态的渠道（作为后备）
func (e *Engine) filterUnknownChannels(channels []*model.Channel) []*model.Channel {
	var unknown []*model.Channel
	for _, ch := range channels {
		if ch.Status == model.ChannelStatusEnabled {
			health, _ := model.GetChannelHealth(ch.Id)
			if health != nil && health.Status == "unknown" {
				unknown = append(unknown, ch)
			}
		}
	}
	return unknown
}

// loadChannelMetrics 加载渠道指标
func (e *Engine) loadChannelMetrics(ctx context.Context, channels []*model.Channel) []*ChannelWithMetrics {
	metrics := make([]*ChannelWithMetrics, 0, len(channels))

	for _, ch := range channels {
		// 获取健康状态
		health, err := model.GetChannelHealth(ch.Id)
		if err != nil {
			logger.SysError(fmt.Sprintf("Failed to get health for channel %d: %v", ch.Id, err))
			continue
		}

		// 计算成功率
		successRate := 1.0
		totalRequests := health.SuccessCount + health.FailureCount
		if totalRequests > 0 {
			successRate = float64(health.SuccessCount) / float64(totalRequests)
		}

		// 构建带指标的渠道
		metrics = append(metrics, &ChannelWithMetrics{
			Channel:     ch,
			AvgLatency:  health.AvgLatency,
			Cost:        0, // TODO: 从配置中获取成本
			SuccessRate: successRate,
			Concurrent:  0, // TODO: 从 Redis 获取当前并发数
			Status:      ChannelStatus(health.Status),
		})
	}

	return metrics
}

// logDecision 记录路由决策日志
func (e *Engine) logDecision(req *SelectRequest, result *SelectResult, status string, attempt int) {
	decision := &RoutingDecision{
		RequestID:       req.RequestID,
		Timestamp:       time.Now(),
		UserID:          req.UserID,
		Group:           req.Group,
		Model:           req.Model,
		Strategy:        string(req.Strategy),
		CandidateCount:  result.CandidateCount,
		SelectedChannel: result.Channel.Id,
		Reason:          result.Reason,
		Attempt:         attempt,
		Result:          status,
		DecisionTimeMs:  result.DecisionTime.Milliseconds(),
	}

	// TODO: 异步写入日志到数据库或消息队列
	logger.Debugf(context.Background(), "Routing decision: %+v", decision)
}

// SetDefaultStrategy 设置默认策略
func (e *Engine) SetDefaultStrategy(strategy RouterStrategy) {
	e.defaultStrategy = strategy
}

// GetStrategy 获取策略实例（用于外部调用 OnChannelSuccess/Failed）
func (e *Engine) GetStrategy(strategyType RouterStrategy) Strategy {
	return e.strategyFactory.GetStrategy(strategyType)
}

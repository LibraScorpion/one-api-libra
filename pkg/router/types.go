package router

import (
	"time"

	"github.com/songquanpeng/one-api/model"
)

// RouterStrategy 路由策略类型
type RouterStrategy string

const (
	StrategyWeightRoundRobin RouterStrategy = "weight_rr"      // 权重轮询
	StrategyPriority         RouterStrategy = "priority"       // 优先级
	StrategyLowestCost       RouterStrategy = "lowest_cost"    // 最低成本
	StrategyLowestLatency    RouterStrategy = "lowest_latency" // 最低延迟
	StrategyRoundRobin       RouterStrategy = "round_robin"    // 轮询
)

// ChannelStatus 渠道健康状态
type ChannelStatus string

const (
	StatusHealthy   ChannelStatus = "healthy"
	StatusUnhealthy ChannelStatus = "unhealthy"
	StatusUnknown   ChannelStatus = "unknown"
)

// SelectRequest 路由选择请求
type SelectRequest struct {
	RequestID string        // 请求 ID
	UserID    int           // 用户 ID
	Group     string        // 用户分组
	Model     string        // 请求模型
	Strategy  RouterStrategy // 路由策略
}

// SelectResult 路由选择结果
type SelectResult struct {
	Channel        *model.Channel // 选中的渠道
	Reason         string         // 选择原因
	CandidateCount int            // 候选渠道数量
	DecisionTime   time.Duration  // 决策耗时
}

// WeightedChannel 带权重的渠道
type WeightedChannel struct {
	Channel         *model.Channel
	Weight          int   // 静态权重
	CurrentWeight   int   // 当前权重
	EffectiveWeight int   // 有效权重（失败时降低）
	Priority        int64 // 优先级
}

// ChannelWithMetrics 带指标的渠道
type ChannelWithMetrics struct {
	Channel    *model.Channel
	AvgLatency int     // 平均延迟（ms）
	Cost       float64 // 成本（每百万 token）
	SuccessRate float64 // 成功率
	Concurrent  int     // 当前并发数
	Status     ChannelStatus // 健康状态
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	ChannelID    int
	Success      bool
	Latency      int
	ErrorMessage string
	Timestamp    time.Time
}

// RoutingDecision 路由决策日志
type RoutingDecision struct {
	RequestID       string    `json:"request_id"`
	Timestamp       time.Time `json:"timestamp"`
	UserID          int       `json:"user_id"`
	Group           string    `json:"group"`
	Model           string    `json:"model"`
	Strategy        string    `json:"strategy"`
	CandidateCount  int       `json:"candidate_count"`
	SelectedChannel int       `json:"selected_channel_id"`
	Reason          string    `json:"reason"`
	Attempt         int       `json:"attempt"`
	Result          string    `json:"result"`
	DecisionTimeMs  int64     `json:"decision_time_ms"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries        int           // 最大重试次数
	InitialDelay      time.Duration // 初始延迟
	MaxDelay          time.Duration // 最大延迟
	BackoffMultiplier float64       // 退避倍数
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

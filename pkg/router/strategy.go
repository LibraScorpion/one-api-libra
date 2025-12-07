package router

import (
	"math/rand"
	"sort"

	"github.com/songquanpeng/one-api/model"
)

// Strategy 路由策略接口
type Strategy interface {
	Select(channels []*ChannelWithMetrics) *model.Channel
	Name() string
}

// WeightRoundRobinStrategy 平滑加权轮询策略
type WeightRoundRobinStrategy struct {
	state map[int]*WeightedChannel // channelID -> WeightedChannel
}

func NewWeightRoundRobinStrategy() *WeightRoundRobinStrategy {
	return &WeightRoundRobinStrategy{
		state: make(map[int]*WeightedChannel),
	}
}

func (s *WeightRoundRobinStrategy) Name() string {
	return string(StrategyWeightRoundRobin)
}

func (s *WeightRoundRobinStrategy) Select(channels []*ChannelWithMetrics) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 构建加权渠道列表
	weightedChannels := make([]*WeightedChannel, 0, len(channels))
	for _, ch := range channels {
		// 从状态中获取或创建 WeightedChannel
		wc, exists := s.state[ch.Channel.Id]
		if !exists {
			weight := 1
			if ch.Channel.Weight != nil {
				weight = int(*ch.Channel.Weight)
			}
			wc = &WeightedChannel{
				Channel:         ch.Channel,
				Weight:          weight,
				CurrentWeight:   0,
				EffectiveWeight: weight,
			}
			s.state[ch.Channel.Id] = wc
		}
		weightedChannels = append(weightedChannels, wc)
	}

	// 平滑加权轮询算法
	var selected *WeightedChannel
	totalWeight := 0

	for _, wc := range weightedChannels {
		// 累加当前权重
		wc.CurrentWeight += wc.EffectiveWeight
		totalWeight += wc.EffectiveWeight

		// 选择当前权重最大的
		if selected == nil || wc.CurrentWeight > selected.CurrentWeight {
			selected = wc
		}
	}

	if selected != nil {
		// 减去总权重
		selected.CurrentWeight -= totalWeight
		return selected.Channel
	}

	return nil
}

// OnChannelFailed 渠道失败时降低有效权重
func (s *WeightRoundRobinStrategy) OnChannelFailed(channelID int) {
	if wc, exists := s.state[channelID]; exists {
		wc.EffectiveWeight = max(1, wc.EffectiveWeight-1)
	}
}

// OnChannelSuccess 渠道成功时恢复有效权重
func (s *WeightRoundRobinStrategy) OnChannelSuccess(channelID int) {
	if wc, exists := s.state[channelID]; exists {
		if wc.EffectiveWeight < wc.Weight {
			wc.EffectiveWeight++
		}
	}
}

// PriorityStrategy 优先级策略
type PriorityStrategy struct{}

func NewPriorityStrategy() *PriorityStrategy {
	return &PriorityStrategy{}
}

func (s *PriorityStrategy) Name() string {
	return string(StrategyPriority)
}

func (s *PriorityStrategy) Select(channels []*ChannelWithMetrics) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 按优先级分组
	priorityGroups := make(map[int64][]*ChannelWithMetrics)
	maxPriority := int64(0)

	for _, ch := range channels {
		priority := int64(0)
		if ch.Channel.Priority != nil {
			priority = *ch.Channel.Priority
		}
		priorityGroups[priority] = append(priorityGroups[priority], ch)
		if priority > maxPriority {
			maxPriority = priority
		}
	}

	// 获取最高优先级组
	topChannels := priorityGroups[maxPriority]

	// 在最高优先级组内随机选择
	if len(topChannels) > 0 {
		return topChannels[rand.Intn(len(topChannels))].Channel
	}

	return nil
}

// LowestCostStrategy 最低成本策略
type LowestCostStrategy struct{}

func NewLowestCostStrategy() *LowestCostStrategy {
	return &LowestCostStrategy{}
}

func (s *LowestCostStrategy) Name() string {
	return string(StrategyLowestCost)
}

func (s *LowestCostStrategy) Select(channels []*ChannelWithMetrics) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 按成本排序
	sorted := make([]*ChannelWithMetrics, len(channels))
	copy(sorted, channels)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cost < sorted[j].Cost
	})

	// 在成本最低的前 3 个中随机选择（避免单点过载）
	topN := min(3, len(sorted))
	return sorted[rand.Intn(topN)].Channel
}

// LowestLatencyStrategy 最低延迟策略
type LowestLatencyStrategy struct{}

func NewLowestLatencyStrategy() *LowestLatencyStrategy {
	return &LowestLatencyStrategy{}
}

func (s *LowestLatencyStrategy) Name() string {
	return string(StrategyLowestLatency)
}

func (s *LowestLatencyStrategy) Select(channels []*ChannelWithMetrics) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 按平均延迟排序
	sorted := make([]*ChannelWithMetrics, len(channels))
	copy(sorted, channels)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AvgLatency < sorted[j].AvgLatency
	})

	// 在延迟最低的前 3 个中随机选择
	topN := min(3, len(sorted))
	return sorted[rand.Intn(topN)].Channel
}

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	counter int
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		counter: 0,
	}
}

func (s *RoundRobinStrategy) Name() string {
	return string(StrategyRoundRobin)
}

func (s *RoundRobinStrategy) Select(channels []*ChannelWithMetrics) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 简单轮询（注意：多实例需要使用 Redis 计数器）
	index := s.counter % len(channels)
	s.counter++
	return channels[index].Channel
}

// StrategyFactory 策略工厂
type StrategyFactory struct {
	strategies map[RouterStrategy]Strategy
}

func NewStrategyFactory() *StrategyFactory {
	return &StrategyFactory{
		strategies: map[RouterStrategy]Strategy{
			StrategyWeightRoundRobin: NewWeightRoundRobinStrategy(),
			StrategyPriority:         NewPriorityStrategy(),
			StrategyLowestCost:       NewLowestCostStrategy(),
			StrategyLowestLatency:    NewLowestLatencyStrategy(),
			StrategyRoundRobin:       NewRoundRobinStrategy(),
		},
	}
}

func (f *StrategyFactory) GetStrategy(strategyType RouterStrategy) Strategy {
	if strategy, ok := f.strategies[strategyType]; ok {
		return strategy
	}
	// 默认返回优先级策略
	return f.strategies[StrategyPriority]
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

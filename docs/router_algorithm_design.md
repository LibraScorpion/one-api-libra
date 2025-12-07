# API Key 池子路由算法设计文档

## 一、现状分析

### 1.1 当前实现
- **文件位置**：`middleware/distributor.go`、`model/ability.go`
- **核心逻辑**：
  ```go
  // 1. 查询最高优先级的渠道
  // 2. 在相同优先级中随机选择（RANDOM()/RAND()）
  // 3. 返回单个渠道
  ```

### 1.2 现有问题
❌ **算法单一**：仅支持优先级+随机，无权重、成本、延迟等策略
❌ **无健康检查**：故障渠道无法自动下线
❌ **无重试机制**：单次失败无法自动切换到备用渠道
❌ **无并发控制**：可能导致某些渠道过载
❌ **性能问题**：每次请求都查数据库
❌ **无路由日志**：无法追踪路由决策过程
❌ **无成本优化**：无法根据成本选择最优渠道

---

## 二、核心设计目标

### 2.1 功能目标
✅ 支持多种路由策略（权重、优先级、成本、延迟、轮询）
✅ 实时健康检查与自动下线
✅ 智能失败重试与回退
✅ 并发控制与限流
✅ 高性能（内存缓存 + Redis）
✅ 完整的路由决策日志

### 2.2 性能目标
- 路由决策延迟 < 5ms
- 支持 10000+ QPS
- 缓存命中率 > 95%
- 健康检查周期 < 30s

---

## 三、架构设计

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    API Request                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Router Middleware                          │
│  - 解析请求（model、group、user）                        │
│  - 调用 RouterEngine.SelectChannel()                    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              RouterEngine（核心路由引擎）                │
│  ┌───────────────────────────────────────────────────┐  │
│  │  1. ChannelPoolManager（渠道池管理器）            │  │
│  │     - 从缓存/DB 加载可用渠道                      │  │
│  │     - 过滤已禁用/不健康的渠道                     │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  ChannelCache（渠道缓存）                   │  │  │
│  │  │  - Redis: channel:{group}:{model} -> []ch  │  │  │
│  │  │  - Memory: LRU Cache                       │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
│                        │                                 │
│                        ▼                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  2. StrategySelector（策略选择器）                │  │
│  │     - 根据配置选择路由策略                        │  │
│  │     - 策略：Weight/Priority/Cost/Latency/RoundRobin │
│  └───────────────────────────────────────────────────┘  │
│                        │                                 │
│                        ▼                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  3. ChannelScorer（渠道评分器）                   │  │
│  │     - 根据策略对渠道打分                          │  │
│  │     - 权重计算、成本计算、延迟评估                │  │
│  └───────────────────────────────────────────────────┘  │
│                        │                                 │
│                        ▼                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  4. ConcurrencyController（并发控制器）           │  │
│  │     - 检查渠道并发数                              │  │
│  │     - 检查限流配置（QPS/QPM）                     │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  Redis: channel:{id}:concurrent -> counter │  │  │
│  │  │  Redis: channel:{id}:qps:{min} -> counter  │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
│                        │                                 │
│                        ▼                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  5. HealthChecker（健康检查器）                   │  │
│  │     - 检查渠道健康状态                            │  │
│  │     - 自动下线故障渠道                            │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  ChannelHealth 表                          │  │  │
│  │  │  - success_count, failure_count            │  │  │
│  │  │  - avg_latency, status                     │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
│                        │                                 │
│                        ▼                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  6. FallbackHandler（回退处理器）                 │  │
│  │     - 失败时自动选择备用渠道                      │  │
│  │     - 重试计数与策略                              │  │
│  └───────────────────────────────────────────────────┘  │
│                        │                                 │
│                        ▼                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  7. DecisionLogger（决策日志器）                  │  │
│  │     - 记录路由决策过程                            │  │
│  │     - 用于调试和审计                              │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  routing_logs 表                           │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            Selected Channel（选中的渠道）                │
│  - channel_id                                           │
│  - channel_key                                          │
│  - base_url                                             │
│  - decision_reason                                      │
└─────────────────────────────────────────────────────────┘
```

---

## 四、核心算法详解

### 4.1 路由策略

#### 策略 1：权重轮询（Weight Round-Robin）
**适用场景**：负载均衡，按渠道能力分配流量

```go
// 算法：平滑加权轮询（Smooth Weighted Round-Robin）
// 参考 Nginx 的 swrr 算法

type WeightedChannel struct {
    Channel       *Channel
    Weight        int    // 静态权重
    CurrentWeight int    // 当前权重
    EffectiveWeight int  // 有效权重（失败时降低）
}

func SelectByWeight(channels []*WeightedChannel) *Channel {
    if len(channels) == 0 {
        return nil
    }

    var selected *WeightedChannel
    totalWeight := 0

    for _, ch := range channels {
        // 累加当前权重
        ch.CurrentWeight += ch.EffectiveWeight
        totalWeight += ch.EffectiveWeight

        // 选择当前权重最大的
        if selected == nil || ch.CurrentWeight > selected.CurrentWeight {
            selected = ch
        }
    }

    if selected != nil {
        // 减去总权重
        selected.CurrentWeight -= totalWeight
        return selected.Channel
    }

    return nil
}

// 失败时降低有效权重
func OnChannelFailed(ch *WeightedChannel) {
    ch.EffectiveWeight = max(1, ch.EffectiveWeight - 1)
}

// 成功时恢复有效权重
func OnChannelSuccess(ch *WeightedChannel) {
    if ch.EffectiveWeight < ch.Weight {
        ch.EffectiveWeight++
    }
}
```

**优点**：
- 流量分配平滑
- 自动适应渠道质量（失败时降权）
- 不需要额外状态存储

**缺点**：
- 需要维护 CurrentWeight 状态
- 多实例部署需要共享状态（Redis）

---

#### 策略 2：优先级 + 权重（Priority + Weight）
**适用场景**：优先使用高质量渠道，同优先级内负载均衡

```go
func SelectByPriorityAndWeight(channels []*Channel) *Channel {
    if len(channels) == 0 {
        return nil
    }

    // 1. 按优先级分组
    priorityGroups := groupByPriority(channels)

    // 2. 获取最高优先级组
    maxPriority := getMaxPriority(priorityGroups)
    topChannels := priorityGroups[maxPriority]

    // 3. 在最高优先级组内按权重选择
    return SelectByWeight(topChannels)
}
```

---

#### 策略 3：最低成本（Lowest Cost）
**适用场景**：成本敏感场景

```go
type ChannelWithCost struct {
    Channel *Channel
    Cost    float64  // 每百万 token 成本
}

func SelectByLowestCost(channels []*ChannelWithCost) *Channel {
    if len(channels) == 0 {
        return nil
    }

    // 按成本排序，取前 N 个（避免单点）
    sort.Slice(channels, func(i, j int) bool {
        return channels[i].Cost < channels[j].Cost
    })

    // 在成本最低的前 3 个中随机选择（避免单点过载）
    topN := min(3, len(channels))
    selected := channels[rand.Intn(topN)]

    return selected.Channel
}
```

---

#### 策略 4：最低延迟（Lowest Latency）
**适用场景**：延迟敏感场景

```go
func SelectByLowestLatency(channels []*ChannelWithLatency) *Channel {
    if len(channels) == 0 {
        return nil
    }

    // 按平均延迟排序
    sort.Slice(channels, func(i, j int) bool {
        return channels[i].AvgLatency < channels[j].AvgLatency
    })

    // 在延迟最低的前 3 个中按权重选择
    topN := min(3, len(channels))
    return SelectByWeight(channels[:topN])
}
```

---

#### 策略 5：轮询（Round-Robin）
**适用场景**：简单负载均衡

```go
// 使用 Redis 存储全局计数器
func SelectByRoundRobin(channels []*Channel, cacheKey string) *Channel {
    if len(channels) == 0 {
        return nil
    }

    // Redis INCR 保证原子性
    counter := redis.Incr(ctx, "rr:counter:"+cacheKey)
    index := int(counter) % len(channels)

    return channels[index]
}
```

---

### 4.2 健康检查机制

#### 被动健康检查（Passive Health Check）
每次请求后更新渠道健康状态

```go
type ChannelHealth struct {
    ChannelID      int       `gorm:"primaryKey"`
    SuccessCount   int       `gorm:"default:0"`
    FailureCount   int       `gorm:"default:0"`
    LastSuccessAt  time.Time
    LastFailureAt  time.Time
    AvgLatency     int       // ms
    Status         string    `gorm:"type:varchar(20)"` // healthy/unhealthy/unknown
    ConsecutiveFails int     `gorm:"default:0"`
}

// 请求成功后调用
func OnRequestSuccess(channelID int, latency int) {
    health := GetChannelHealth(channelID)
    health.SuccessCount++
    health.LastSuccessAt = time.Now()
    health.ConsecutiveFails = 0

    // 更新平均延迟（指数移动平均）
    alpha := 0.2
    health.AvgLatency = int(alpha * float64(latency) + (1-alpha) * float64(health.AvgLatency))

    // 判断是否恢复健康
    if health.Status == "unhealthy" {
        successRate := float64(health.SuccessCount) / float64(health.SuccessCount + health.FailureCount)
        if successRate > 0.9 {
            health.Status = "healthy"
        }
    }

    health.Save()
}

// 请求失败后调用
func OnRequestFailure(channelID int, errorCode string) {
    health := GetChannelHealth(channelID)
    health.FailureCount++
    health.LastFailureAt = time.Now()
    health.ConsecutiveFails++

    // 连续失败 3 次，标记为不健康
    if health.ConsecutiveFails >= 3 {
        health.Status = "unhealthy"
        // 触发告警
        sendAlert(channelID, "Channel unhealthy")
    }

    // 失败率超过 10%，标记为不健康
    failureRate := float64(health.FailureCount) / float64(health.SuccessCount + health.FailureCount)
    if failureRate > 0.1 {
        health.Status = "unhealthy"
    }

    health.Save()
}
```

#### 主动健康检查（Active Health Check）
定期主动探测渠道可用性

```go
// 每 30 秒运行一次
func ActiveHealthCheckTask() {
    channels := GetAllEnabledChannels()

    for _, channel := range channels {
        go func(ch *Channel) {
            // 发送测试请求
            err := sendTestRequest(ch)

            if err != nil {
                OnRequestFailure(ch.ID, "health_check_failed")
            } else {
                OnRequestSuccess(ch.ID, 0)
            }
        }(channel)
    }
}

func sendTestRequest(channel *Channel) error {
    // 发送简单的模型请求
    req := &TestRequest{
        Model: "gpt-3.5-turbo",
        Messages: []Message{
            {Role: "user", Content: "test"},
        },
        MaxTokens: 1,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    resp, err := callChannel(ctx, channel, req)
    if err != nil {
        return err
    }

    // 检查响应是否正常
    if resp.StatusCode != 200 {
        return fmt.Errorf("status code: %d", resp.StatusCode)
    }

    return nil
}
```

---

### 4.3 并发控制与限流

#### 并发控制
限制每个渠道的最大并发数

```go
type ConcurrencyController struct {
    redis *redis.Client
}

func (cc *ConcurrencyController) TryAcquire(channelID int, maxConcurrent int) (bool, error) {
    key := fmt.Sprintf("channel:%d:concurrent", channelID)

    // 使用 Lua 脚本保证原子性
    script := `
        local current = redis.call('GET', KEYS[1])
        if not current then
            current = 0
        end
        current = tonumber(current)

        if current < tonumber(ARGV[1]) then
            redis.call('INCR', KEYS[1])
            redis.call('EXPIRE', KEYS[1], 300)
            return 1
        else
            return 0
        end
    `

    result, err := cc.redis.Eval(context.Background(), script, []string{key}, maxConcurrent).Int()
    if err != nil {
        return false, err
    }

    return result == 1, nil
}

func (cc *ConcurrencyController) Release(channelID int) error {
    key := fmt.Sprintf("channel:%d:concurrent", channelID)
    return cc.redis.Decr(context.Background(), key).Err()
}
```

#### QPS 限流
使用滑动窗口算法

```go
func (cc *ConcurrencyController) CheckRateLimit(channelID int, maxQPS int) (bool, error) {
    now := time.Now()
    key := fmt.Sprintf("channel:%d:qps:%d", channelID, now.Unix()/60) // 按分钟

    // 使用 Lua 脚本实现滑动窗口
    script := `
        local count = redis.call('GET', KEYS[1])
        if not count then
            count = 0
        end
        count = tonumber(count)

        if count < tonumber(ARGV[1]) then
            redis.call('INCR', KEYS[1])
            redis.call('EXPIRE', KEYS[1], 60)
            return 1
        else
            return 0
        end
    `

    result, err := cc.redis.Eval(context.Background(), script, []string{key}, maxQPS).Int()
    if err != nil {
        return false, err
    }

    return result == 1, nil
}
```

---

### 4.4 失败重试与回退

```go
type RetryConfig struct {
    MaxRetries      int           // 最大重试次数
    InitialDelay    time.Duration // 初始延迟
    MaxDelay        time.Duration // 最大延迟
    BackoffMultiplier float64     // 退避倍数
}

func RetryWithFallback(ctx context.Context, req *Request, config *RetryConfig) (*Response, error) {
    var lastErr error
    delay := config.InitialDelay

    // 获取所有可用渠道（按优先级排序）
    channels := GetSortedChannels(req.Group, req.Model)

    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        // 选择渠道（跳过已失败的）
        channel := SelectNextChannel(channels, attempt)
        if channel == nil {
            break
        }

        // 尝试请求
        resp, err := CallChannel(ctx, channel, req)
        if err == nil {
            // 成功
            OnRequestSuccess(channel.ID, resp.Latency)
            logDecision(req, channel, "success", attempt)
            return resp, nil
        }

        // 失败
        lastErr = err
        OnRequestFailure(channel.ID, err.Error())
        logDecision(req, channel, fmt.Sprintf("failed: %v", err), attempt)

        // 判断是否应该重试
        if !ShouldRetry(err) {
            break
        }

        // 指数退避
        if attempt < config.MaxRetries {
            time.Sleep(delay)
            delay = time.Duration(float64(delay) * config.BackoffMultiplier)
            if delay > config.MaxDelay {
                delay = config.MaxDelay
            }
        }
    }

    return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

// 判断是否应该重试
func ShouldRetry(err error) bool {
    // 5xx 错误重试
    if statusErr, ok := err.(*StatusError); ok {
        return statusErr.Code >= 500 && statusErr.Code < 600
    }

    // 网络错误重试
    if _, ok := err.(*net.OpError); ok {
        return true
    }

    // 超时错误重试
    if err == context.DeadlineExceeded {
        return true
    }

    // 4xx 错误不重试
    return false
}
```

---

### 4.5 智能缓存

#### 两级缓存架构
1. **L1 缓存（本地内存）**：LRU，容量 1000 条
2. **L2 缓存（Redis）**：TTL 60s

```go
type ChannelCache struct {
    local  *lru.Cache        // 本地 LRU 缓存
    redis  *redis.Client     // Redis 缓存
    db     *gorm.DB          // 数据库
}

func (cc *ChannelCache) GetChannels(group string, model string) ([]*Channel, error) {
    key := fmt.Sprintf("%s:%s", group, model)

    // 1. 尝试本地缓存
    if val, ok := cc.local.Get(key); ok {
        return val.([]*Channel), nil
    }

    // 2. 尝试 Redis
    data, err := cc.redis.Get(context.Background(), "channels:"+key).Result()
    if err == nil {
        var channels []*Channel
        json.Unmarshal([]byte(data), &channels)
        cc.local.Add(key, channels)
        return channels, nil
    }

    // 3. 查询数据库
    channels, err := cc.loadFromDB(group, model)
    if err != nil {
        return nil, err
    }

    // 4. 写入缓存
    cc.writeToCache(key, channels)

    return channels, nil
}

func (cc *ChannelCache) writeToCache(key string, channels []*Channel) {
    // 写入本地缓存
    cc.local.Add(key, channels)

    // 写入 Redis
    data, _ := json.Marshal(channels)
    cc.redis.Set(context.Background(), "channels:"+key, data, 60*time.Second)
}

// 缓存失效（渠道配置变更时）
func (cc *ChannelCache) Invalidate(channelID int) {
    // 清空本地缓存
    cc.local.Purge()

    // 删除 Redis 中相关的 key
    // 需要维护 channel_id -> cache_keys 的映射
    keys := cc.getRelatedKeys(channelID)
    for _, key := range keys {
        cc.redis.Del(context.Background(), "channels:"+key)
    }
}
```

---

### 4.6 路由决策日志

```go
type RoutingDecision struct {
    RequestID       string    `json:"request_id"`
    Timestamp       time.Time `json:"timestamp"`
    UserID          int       `json:"user_id"`
    Group           string    `json:"group"`
    Model           string    `json:"model"`
    Strategy        string    `json:"strategy"`
    CandidateCount  int       `json:"candidate_count"`
    SelectedChannel int       `json:"selected_channel"`
    Reason          string    `json:"reason"`
    Attempt         int       `json:"attempt"`
    Result          string    `json:"result"` // success/failed
}

func logDecision(req *Request, channel *Channel, reason string, attempt int) {
    decision := &RoutingDecision{
        RequestID:       req.ID,
        Timestamp:       time.Now(),
        UserID:          req.UserID,
        Group:           req.Group,
        Model:           req.Model,
        Strategy:        "priority+weight",
        SelectedChannel: channel.ID,
        Reason:          reason,
        Attempt:         attempt,
    }

    // 异步写入日志（避免阻塞）
    go func() {
        data, _ := json.Marshal(decision)
        redis.LPush(context.Background(), "routing_logs", data)
    }()
}
```

---

## 五、数据库表设计

### 5.1 渠道健康表
```sql
CREATE TABLE channel_health (
    channel_id INT PRIMARY KEY,
    success_count INT DEFAULT 0,
    failure_count INT DEFAULT 0,
    consecutive_fails INT DEFAULT 0,
    last_success_at BIGINT,
    last_failure_at BIGINT,
    avg_latency INT DEFAULT 0 COMMENT '平均延迟（ms）',
    status VARCHAR(20) NOT NULL DEFAULT 'unknown' COMMENT 'healthy/unhealthy/unknown',
    updated_at BIGINT,
    INDEX idx_status (status),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 5.2 路由日志表
```sql
CREATE TABLE routing_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    request_id VARCHAR(64) NOT NULL,
    timestamp BIGINT NOT NULL,
    user_id INT NOT NULL,
    group_name VARCHAR(32) NOT NULL,
    model VARCHAR(100) NOT NULL,
    strategy VARCHAR(50) NOT NULL,
    candidate_count INT DEFAULT 0,
    selected_channel_id INT NOT NULL,
    reason TEXT,
    attempt INT DEFAULT 0,
    result VARCHAR(20) COMMENT 'success/failed',
    INDEX idx_request_id (request_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_channel (selected_channel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY RANGE (timestamp) (
    -- 按月分区
);
```

### 5.3 渠道配置增强
```sql
ALTER TABLE channels
ADD COLUMN max_concurrent INT DEFAULT 0 COMMENT '最大并发数，0=无限制',
ADD COLUMN max_qps INT DEFAULT 0 COMMENT '最大 QPS，0=无限制',
ADD COLUMN cost_per_million BIGINT DEFAULT 0 COMMENT '每百万 token 成本（单位：分）',
ADD COLUMN auto_disable_on_error BOOLEAN DEFAULT TRUE COMMENT '错误时是否自动禁用';
```

---

## 六、实现计划

### 第一步：核心路由引擎（1-2天）
- [ ] `pkg/router/engine.go` - 路由引擎主逻辑
- [ ] `pkg/router/strategy.go` - 路由策略接口与实现
- [ ] `pkg/router/scorer.go` - 渠道评分器

### 第二步：健康检查（1天）
- [ ] `pkg/router/health.go` - 健康检查器
- [ ] `model/channel_health.go` - 健康状态数据模型
- [ ] 定时任务：主动健康检查

### 第三步：并发控制与限流（1天）
- [ ] `pkg/router/concurrency.go` - 并发控制器
- [ ] `pkg/router/ratelimit.go` - 限流器
- [ ] Redis Lua 脚本

### 第四步：缓存优化（1天）
- [ ] `pkg/router/cache.go` - 两级缓存
- [ ] 缓存失效机制
- [ ] 热加载配置

### 第五步：重试回退（1天）
- [ ] `pkg/router/retry.go` - 重试逻辑
- [ ] 回退策略
- [ ] 错误分类

### 第六步：日志与监控（1天）
- [ ] `pkg/router/logger.go` - 决策日志
- [ ] Prometheus 指标
- [ ] Grafana 仪表盘

### 第七步：集成测试（1天）
- [ ] 单元测试
- [ ] 集成测试
- [ ] 压力测试

---

## 七、性能优化

### 7.1 优化点
1. **预加载渠道池**：启动时加载所有渠道到内存
2. **批量更新健康状态**：每秒批量写入一次
3. **异步日志**：路由决策日志异步写入
4. **连接池**：复用 HTTP 连接
5. **并发请求**：支持并发调用多个渠道（取最快响应）

### 7.2 基准测试目标
- 单次路由决策 < 5ms
- QPS > 10000
- 内存占用 < 500MB
- CPU 占用 < 50%（10000 QPS 时）

---

## 八、监控指标

### 8.1 路由指标
- `router_request_total` - 路由请求总数
- `router_decision_duration_seconds` - 路由决策耗时
- `router_strategy_usage` - 各策略使用次数
- `router_cache_hit_ratio` - 缓存命中率

### 8.2 渠道指标
- `channel_requests_total{channel_id}` - 渠道请求总数
- `channel_success_rate{channel_id}` - 渠道成功率
- `channel_avg_latency{channel_id}` - 渠道平均延迟
- `channel_concurrent{channel_id}` - 渠道当前并发数
- `channel_health_status{channel_id}` - 渠道健康状态

### 8.3 告警规则
```yaml
groups:
  - name: router_alerts
    rules:
      - alert: ChannelHighFailureRate
        expr: channel_success_rate < 0.9
        for: 5m
        annotations:
          summary: "渠道 {{ $labels.channel_id }} 失败率过高"

      - alert: RouterHighLatency
        expr: router_decision_duration_seconds > 0.01
        for: 1m
        annotations:
          summary: "路由决策延迟过高"

      - alert: ChannelAllDown
        expr: sum(channel_health_status == 1) == 0
        for: 1m
        annotations:
          summary: "所有渠道均不可用"
```

---

## 九、总结

### 核心优势
✅ **多策略支持**：权重、优先级、成本、延迟等
✅ **高可用**：健康检查 + 自动下线 + 失败重试
✅ **高性能**：两级缓存 + 并发控制 + 异步日志
✅ **可观测**：完整的决策日志 + Prometheus 指标
✅ **易扩展**：策略模式，易于添加新策略

### 下一步
开始实现第一步：核心路由引擎代码编写

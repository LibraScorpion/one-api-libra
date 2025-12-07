# OpenRouter 转型技术设计

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
├─────────────────────────────────────────────────────────────────┤
│  首页  │  模型市场  │  排行榜  │  额度管理  │  API Keys  │  活动  │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP/WebSocket
┌────────────────────────────▼────────────────────────────────────┐
│                      API Gateway (Gin)                           │
├─────────────────────────────────────────────────────────────────┤
│   认证中间件  │  限流中间件  │  余额检查中间件  │  日志中间件    │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                     Smart Distributor                            │
├─────────────────────────────────────────────────────────────────┤
│  渠道选择  │  健康检查  │  故障转移  │  负载均衡  │  成本优化    │
└────────────────────────────┬────────────────────────────────────┘
                             │
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌─────────────┐      ┌─────────────┐     ┌─────────────┐
│ OpenRouter  │      │   OpenAI    │     │  Anthropic  │
│   Channel   │      │   Channel   │     │   Channel   │
└─────────────┘      └─────────────┘     └─────────────┘
```

## 核心模块设计

### 1. Smart Distributor（智能分发器）

```go
// middleware/smart_distributor.go

type SmartDistributor struct {
    healthChecker  *HealthChecker
    loadBalancer   *LoadBalancer
    costOptimizer  *CostOptimizer
    failoverPolicy *FailoverPolicy
}

type RoutingStrategy string

const (
    StrategyLowestLatency  RoutingStrategy = "lowest_latency"
    StrategyLowestCost     RoutingStrategy = "lowest_cost"
    StrategyWeightedRandom RoutingStrategy = "weighted_random"
)

func (d *SmartDistributor) SelectChannel(
    ctx context.Context,
    modelName string,
    strategy RoutingStrategy,
) (*Channel, error) {
    // 1. 获取支持该模型的所有渠道
    channels := d.getChannelsForModel(modelName)
    
    // 2. 过滤不健康的渠道
    healthyChannels := d.healthChecker.FilterHealthy(channels)
    
    // 3. 根据策略选择渠道
    switch strategy {
    case StrategyLowestLatency:
        return d.loadBalancer.SelectByLatency(healthyChannels)
    case StrategyLowestCost:
        return d.costOptimizer.SelectByCost(healthyChannels)
    case StrategyWeightedRandom:
        return d.loadBalancer.SelectByWeight(healthyChannels)
    }
    
    return nil, ErrNoAvailableChannel
}
```

### 2. 健康检查器

```go
// model/channel_health.go

type ChannelHealth struct {
    ID                  int64     `gorm:"primaryKey"`
    ChannelID           int       `gorm:"uniqueIndex"`
    Status              string    `gorm:"type:enum('healthy','unhealthy','degraded')"`
    AvgLatencyMs        int
    SuccessRate         float64
    LastCheckAt         time.Time
    LastSuccessAt       time.Time
    ConsecutiveFailures int
    UpdatedAt           time.Time
}

type HealthChecker struct {
    interval         time.Duration
    timeout          time.Duration
    failureThreshold int
}

func (h *HealthChecker) RunCheck(channel *Channel) error {
    // 发送测试请求
    start := time.Now()
    err := h.sendTestRequest(channel)
    latency := time.Since(start)
    
    // 更新健康状态
    health := h.getOrCreateHealth(channel.ID)
    if err != nil {
        health.ConsecutiveFailures++
        if health.ConsecutiveFailures >= h.failureThreshold {
            health.Status = "unhealthy"
        }
    } else {
        health.ConsecutiveFailures = 0
        health.Status = "healthy"
        health.LastSuccessAt = time.Now()
        health.AvgLatencyMs = h.updateAvgLatency(health.AvgLatencyMs, latency)
    }
    
    return h.saveHealth(health)
}
```

### 3. 计费系统

```go
// relay/billing/billing.go

type BillingService struct {
    pricingCache *cache.Cache
}

func (b *BillingService) CalculateCost(
    modelName string,
    inputTokens, outputTokens int,
) (float64, error) {
    pricing, err := b.getPricing(modelName)
    if err != nil {
        return 0, err
    }
    
    inputCost := float64(inputTokens) * pricing.InputPrice / 1_000_000
    outputCost := float64(outputTokens) * pricing.OutputPrice / 1_000_000
    
    return inputCost + outputCost, nil
}

func (b *BillingService) DeductBalance(
    ctx context.Context,
    userID int,
    cost float64,
    requestID string,
) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 1. 获取用户当前余额
        var user User
        if err := tx.Lock().First(&user, userID).Error; err != nil {
            return err
        }
        
        // 2. 检查余额
        if user.Balance < cost {
            return ErrInsufficientBalance
        }
        
        // 3. 扣除余额
        user.Balance -= cost
        if err := tx.Save(&user).Error; err != nil {
            return err
        }
        
        // 4. 记录交易
        transaction := BalanceTransaction{
            UserID:       userID,
            Amount:       -cost,
            BalanceAfter: user.Balance,
            Type:         "usage",
            ReferenceID:  requestID,
        }
        return tx.Create(&transaction).Error
    })
}
```

## 数据库迁移

```sql
-- migrations/003_openrouter_transform.sql

-- 1. 渠道健康表
CREATE TABLE IF NOT EXISTS channel_health (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    channel_id INT NOT NULL,
    status ENUM('healthy','unhealthy','degraded') DEFAULT 'healthy',
    avg_latency_ms INT,
    success_rate DECIMAL(5,2),
    last_check_at TIMESTAMP,
    last_success_at TIMESTAMP,
    consecutive_failures INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_channel (channel_id)
);

-- 2. 模型统计表
CREATE TABLE IF NOT EXISTS model_stats (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    model_name VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    request_count BIGINT DEFAULT 0,
    token_count BIGINT DEFAULT 0,
    total_cost DECIMAL(20,8) DEFAULT 0,
    avg_latency_ms INT,
    error_count INT DEFAULT 0,
    UNIQUE KEY uk_model_date (model_name, date),
    INDEX idx_date (date)
);

-- 3. 用户偏好表
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id BIGINT PRIMARY KEY,
    theme VARCHAR(20) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'zh-CN',
    default_model VARCHAR(100),
    routing_strategy VARCHAR(50) DEFAULT 'weighted_random',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 4. 添加 Google OAuth 支持
ALTER TABLE users ADD COLUMN google_id VARCHAR(255) UNIQUE AFTER email;

-- 5. 渠道表添加健康检查配置
ALTER TABLE channels 
    ADD COLUMN weight INT DEFAULT 100 AFTER priority,
    ADD COLUMN health_check_enabled BOOLEAN DEFAULT TRUE,
    ADD COLUMN health_check_interval INT DEFAULT 60;
```

## 前端组件结构

```
web/src/
├── components/
│   ├── ModelCard/           # 模型卡片组件
│   ├── ModelFilter/         # 筛选器组件
│   ├── BalanceDisplay/      # 余额显示组件
│   ├── TransactionList/     # 交易记录列表
│   └── StatChart/           # 统计图表组件
├── pages/
│   ├── Home/                # 首页（模型市场风格）
│   ├── Models/              # 模型列表页
│   ├── Rankings/            # 排行榜页
│   ├── Credits/             # 额度管理页
│   ├── Activity/            # 活动记录页
│   └── Settings/            # 设置页
├── api/
│   ├── models.ts            # 模型相关 API
│   ├── balance.ts           # 余额相关 API
│   └── activity.ts          # 活动相关 API
└── store/
    ├── models.ts            # 模型状态
    ├── user.ts              # 用户状态
    └── balance.ts           # 余额状态
```

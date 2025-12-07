# 渠道健康与智能路由规格

## Purpose
监控渠道健康状态，实现智能路由和故障转移。

## Requirements

### Requirement: 渠道健康检查
系统 SHALL 定期检查渠道可用性。

#### Scenario: 健康检查执行
- GIVEN 系统配置了健康检查间隔（默认 60 秒）
- WHEN 到达检查时间
- THEN 向每个渠道发送测试请求
- AND 记录响应时间和状态

#### Scenario: 渠道标记为不健康
- GIVEN 渠道连续 3 次检查失败
- WHEN 下一次失败
- THEN 标记渠道为 unhealthy
- AND 停止向该渠道路由请求

#### Scenario: 渠道恢复
- GIVEN 渠道被标记为 unhealthy
- WHEN 健康检查成功
- THEN 标记渠道为 healthy
- AND 恢复路由请求

### Requirement: 智能路由算法
系统 MUST 根据多维度选择最优渠道。

#### Scenario: 基于延迟路由
- GIVEN 多个渠道支持同一模型
- WHEN 路由策略为 "lowest_latency"
- THEN 选择平均延迟最低的渠道

#### Scenario: 基于成本路由
- GIVEN 多个渠道支持同一模型
- WHEN 路由策略为 "lowest_cost"
- THEN 选择价格最低的渠道

#### Scenario: 加权随机路由
- GIVEN 渠道配置了权重
- WHEN 路由策略为 "weighted_random"
- THEN 按权重比例随机选择渠道

### Requirement: 故障转移
系统 SHALL 在渠道失败时自动切换。

#### Scenario: 自动故障转移
- GIVEN 请求发送到渠道 A
- WHEN 渠道 A 返回错误（5xx 或超时）
- THEN 自动重试到渠道 B
- AND 记录故障转移事件

#### Scenario: 最大重试次数
- GIVEN 配置 max_retries=3
- WHEN 连续 3 个渠道都失败
- THEN 返回错误给客户端
- AND 记录所有失败的渠道

## Data Model

```sql
CREATE TABLE channel_health (
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
```

## Configuration

```yaml
routing:
  strategy: weighted_random  # lowest_latency | lowest_cost | weighted_random
  health_check:
    enabled: true
    interval: 60s
    timeout: 10s
    failure_threshold: 3
  failover:
    enabled: true
    max_retries: 3
    retry_delay: 100ms
```

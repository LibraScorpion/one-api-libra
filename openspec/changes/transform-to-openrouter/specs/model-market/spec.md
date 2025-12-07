# 模型市场规格

## Purpose
提供模型浏览、搜索、筛选和排行功能。

## Requirements

### Requirement: 模型定价管理
系统 SHALL 维护所有模型的定价信息。

#### Scenario: 定价存储
- GIVEN 模型 "openai/gpt-4"
- WHEN 存储定价信息
- THEN 包含 input_price、output_price（美元/百万 tokens）
- AND 包含上下文长度、能力标签

#### Scenario: 定价同步
- WHEN 管理员触发定价同步
- THEN 从 OpenRouter API 获取最新定价
- AND 更新本地数据库

### Requirement: 模型列表展示
系统 SHALL 提供模型浏览界面。

#### Scenario: 模型卡片信息
- WHEN 展示模型卡片
- THEN 显示：模型名称、供应商、描述
- AND 显示：输入/输出价格
- AND 显示：上下文长度、能力标签
- AND 显示：速度评分

### Requirement: 模型搜索和筛选
系统 MUST 支持多维度筛选。

#### Scenario: 按供应商筛选
- WHEN 用户选择供应商 "OpenAI"
- THEN 只显示 OpenAI 的模型

#### Scenario: 按能力筛选
- WHEN 用户选择能力 "vision"
- THEN 只显示支持视觉输入的模型

#### Scenario: 按价格范围筛选
- WHEN 用户设置价格范围 $0-$10/M tokens
- THEN 只显示输入价格在该范围内的模型

#### Scenario: 关键词搜索
- WHEN 用户输入 "gpt-4"
- THEN 返回名称或描述包含 "gpt-4" 的模型

### Requirement: 模型排序
系统 SHALL 支持多种排序方式。

#### Scenario: 排序选项
- WHEN 用户选择排序方式
- THEN 支持：最受欢迎、价格低到高、价格高到低
- AND 支持：速度最快、上下文最大、最新添加

### Requirement: 模型排行榜
系统 SHALL 提供模型使用排行。

#### Scenario: 用量排行
- WHEN 访问排行榜页面
- THEN 显示过去 7 天使用量最高的模型

#### Scenario: 性价比排行
- WHEN 选择性价比排行
- THEN 按 (质量评分 / 价格) 排序

## Data Model

```sql
CREATE TABLE model_pricing (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  model_name VARCHAR(100) UNIQUE NOT NULL,
  display_name VARCHAR(200),
  provider VARCHAR(50) NOT NULL,
  description TEXT,
  pricing_input DECIMAL(12,8) NOT NULL,
  pricing_output DECIMAL(12,8) NOT NULL,
  context_length INT,
  max_tokens INT,
  capabilities JSON,
  speed_score INT DEFAULT 5,
  quality_score INT DEFAULT 5,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_provider (provider),
  INDEX idx_is_active (is_active),
  FULLTEXT idx_search (model_name, display_name, description)
);

CREATE TABLE model_stats (
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
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/models | 获取模型列表，支持筛选和分页 |
| GET | /api/models/:name | 获取模型详情 |
| GET | /api/models/search | 搜索模型 |
| GET | /api/models/rankings | 获取排行榜 |
| GET | /api/models/providers | 获取供应商列表 |
| PUT | /api/models/pricing | 更新模型定价（管理员） |

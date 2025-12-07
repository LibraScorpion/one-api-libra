# API 路由规格

## Purpose
定义 OpenRouter 风格的 API 端点和路由规则。

## Requirements

### Requirement: OpenAI 兼容 API
系统 SHALL 提供与 OpenAI API 完全兼容的端点。

#### Scenario: Chat Completions
- WHEN 客户端发送 POST 请求到 `/v1/chat/completions`
- AND 请求包含有效的 API Key
- THEN 系统路由请求到适当的渠道并返回响应

#### Scenario: 模型列表
- WHEN 客户端发送 GET 请求到 `/v1/models`
- THEN 系统返回所有可用模型列表

### Requirement: OpenRouter 风格 API
系统 SHALL 提供模型市场相关的 API。

#### Scenario: 获取模型列表
- WHEN 客户端发送 GET 请求到 `/api/models`
- THEN 系统返回模型列表，包含定价、能力、上下文长度信息

#### Scenario: 模型搜索
- WHEN 客户端发送 GET 请求到 `/api/models?q=gpt&provider=openai`
- THEN 系统返回匹配的模型列表

#### Scenario: 模型排行
- WHEN 客户端发送 GET 请求到 `/api/models/rankings?by=usage&period=week`
- THEN 系统返回按用量排序的模型排行

### Requirement: 余额管理 API
系统 MUST 提供余额查询和交易记录 API。

#### Scenario: 查询余额
- WHEN 已认证用户发送 GET 请求到 `/api/balance`
- THEN 系统返回用户当前余额（美元）

#### Scenario: 交易记录
- WHEN 已认证用户发送 GET 请求到 `/api/balance/transactions`
- THEN 系统返回分页的交易记录列表

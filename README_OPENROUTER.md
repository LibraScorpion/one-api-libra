# 万联APIrouter - OpenRouter 集成版

基于 one-api 的大模型 API 路由平台，第一阶段接入 **OpenRouter.ai** 作为基础渠道商。

## ✨ 新增功能

- ✅ **美元余额系统** - 支持美元 Credits 充值和消费
- ✅ **实时成本计算** - 基于模型定价自动计算 Token 成本
- ✅ **余额交易记录** - 完整的充值/消费流水
- ✅ **模型定价管理** - 支持动态配置模型价格
- ✅ **OpenRouter 集成** - 接入 235+ 模型

## 🚀 快速开始

### 1. 获取 OpenRouter API Key

访问 https://openrouter.ai/keys 注册并获取 API Key

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入你的 OpenRouter API Key
```

### 3. 启动服务

```bash
# 使用 Docker Compose 一键启动
docker-compose up -d

# 等待服务启动（约 30 秒）
docker-compose logs -f backend
```

### 4. 执行数据库迁移

```bash
# 运行迁移脚本
docker-compose exec backend ./migrate -dir=./sql

# 或者手动执行 SQL
docker-compose exec mysql mysql -uroot -proot123 oneapi < sql/001_add_balance_system.sql
docker-compose exec mysql mysql -uroot -proot123 oneapi < sql/002_seed_openrouter_data.sql
```

**⚠️ 重要：修改 OpenRouter API Key**

编辑 `sql/002_seed_openrouter_data.sql`，将 `YOUR_OPENROUTER_API_KEY_HERE` 替换为实际的 API Key，然后重新执行。

或者通过管理后台修改：
1. 登录管理后台（默认账号：root / 123456）
2. 进入 "渠道管理"
3. 编辑 ID 为 1000 的 OpenRouter 渠道
4. 填入你的 API Key

### 5. 访问应用

- **前端**: http://localhost:3000
- **管理后台**: http://localhost:3000 （登录：root / 123456）
- **数据库管理**: http://localhost:8080 （Adminer）

## 📡 API 接口

### 余额管理

```bash
# 查询余额
GET /api/balance
Authorization: Bearer {user_token}

# 查询交易记录
GET /api/balance/transactions?page=1&page_size=20
Authorization: Bearer {user_token}

# 管理员充值
POST /api/balance/admin/add
Authorization: Bearer {admin_token}
{
  "user_id": 1,
  "amount": 10.00,
  "description": "充值测试"
}
```

### 模型定价

```bash
# 获取所有模型定价（公开）
GET /api/models/pricing?provider=openrouter

# 获取单个模型定价
GET /api/models/pricing/openai/gpt-4

# 管理员更新定价
PUT /api/models/pricing
Authorization: Bearer {admin_token}
{
  "model_name": "openai/gpt-4",
  "pricing_input": 0.00003,
  "pricing_output": 0.00006
}
```

### OpenAI 兼容接口

```bash
# Chat Completions（自动路由到 OpenRouter）
POST /v1/chat/completions
Authorization: Bearer {api_key}
{
  "model": "openai/gpt-3.5-turbo",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ]
}
```

## 🔧 测试流程

### 1. 管理员为用户充值

```bash
curl -X POST http://localhost:3000/api/balance/admin/add \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "amount": 10.00,
    "description": "测试充值"
  }'
```

### 2. 用户查询余额

```bash
curl -X GET http://localhost:3000/api/balance \
  -H "Authorization: Bearer {user_token}"
```

### 3. 调用 AI API

```bash
# 创建 API Key（在前端操作或使用 API）
curl -X POST http://localhost:3000/api/token \
  -H "Authorization: Bearer {user_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Key"
  }'

# 使用 API Key 调用模型
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer {api_key}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### 4. 查看消费记录

```bash
curl -X GET "http://localhost:3000/api/balance/transactions?page=1" \
  -H "Authorization: Bearer {user_token}"
```

## 📊 数据库结构

### 新增表

**balance_transactions** - 余额交易记录
```sql
CREATE TABLE balance_transactions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  amount DECIMAL(20,8) NOT NULL,        -- 交易金额（美元）
  balance_after DECIMAL(20,8) NOT NULL, -- 交易后余额
  type ENUM('recharge','usage','refund','adjustment'),
  reference_id VARCHAR(100),
  description VARCHAR(500),
  created_at TIMESTAMP
);
```

**model_pricing** - 模型定价
```sql
CREATE TABLE model_pricing (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  model_name VARCHAR(100) UNIQUE NOT NULL,
  display_name VARCHAR(200),
  provider VARCHAR(50),
  pricing_input DECIMAL(12,8),  -- 输入价格（美元/Token）
  pricing_output DECIMAL(12,8), -- 输出价格（美元/Token）
  context_length INT,
  is_active BOOLEAN DEFAULT TRUE
);
```

### 复用字段

- `users.quota` - 存储余额（单位：分，1美元=100分）
- `logs.cost` - 存储每次请求的成本（美元）

## 🔐 环境变量

关键配置项：

```bash
# OpenRouter API Key（必须）
OPENROUTER_API_KEY=sk-or-v1-xxxxx

# 数据库
SQL_DSN=root:root123@tcp(mysql:3306)/oneapi?charset=utf8mb4

# Redis
REDIS_CONN_STRING=redis://redis:6379

# 服务器
PORT=3000
SESSION_SECRET=change-me
```

## 📝 开发说明

### 项目结构

```
.
├── cmd/
│   └── migrate/           # 数据库迁移工具
├── controller/
│   └── balance.go         # 余额管理接口（新增）
├── model/
│   ├── balance_transaction.go  # 余额交易模型（新增）
│   └── model_pricing.go        # 模型定价模型（新增）
├── relay/
│   └── billing/billing.go      # 扣费逻辑（已修改，添加成本计算）
├── router/
│   └── api.go             # API 路由（已修改，添加余额相关路由）
├── sql/
│   ├── 001_add_balance_system.sql       # 数据表迁移
│   └── 002_seed_openrouter_data.sql     # OpenRouter 初始化数据
└── docker-compose.yml
```

### 核心逻辑

**余额系统**：
- 用户余额以"分"为单位存储在 `users.quota` 字段
- 1 美元 = 100 分
- 所有余额操作都会记录到 `balance_transactions` 表

**成本计算**：
- 调用 `model.CalculateTokenCost(modelName, inputTokens, outputTokens)` 计算实际成本
- 成本记录到 `logs.cost` 字段和 `balance_transactions` 表
- 支持动态更新模型定价

**OpenRouter 集成**：
- 渠道 ID: 1000
- 支持 235+ 模型（详见 `/relay/adaptor/openrouter/constants.go`）
- 自动从 `model_pricing` 表获取价格

## 🐛 常见问题

### Q1: 提示 "模型定价不存在"

A: 执行数据库迁移脚本：
```bash
docker-compose exec mysql mysql -uroot -proot123 oneapi < sql/002_seed_openrouter_data.sql
```

### Q2: OpenRouter 返回 401 Unauthorized

A: 检查 API Key 配置：
```bash
# 查看渠道配置
docker-compose exec mysql mysql -uroot -proot123 oneapi -e "SELECT id, name, `key` FROM channels WHERE id=1000;"

# 更新 API Key
docker-compose exec mysql mysql -uroot -proot123 oneapi -e "UPDATE channels SET \`key\`='sk-or-v1-YOUR_KEY' WHERE id=1000;"
```

### Q3: 余额扣除不正确

A: 检查模型定价是否正确：
```bash
# 查看定价
curl http://localhost:3000/api/models/pricing/openai/gpt-3.5-turbo

# 查看交易记录
curl -H "Authorization: Bearer {token}" http://localhost:3000/api/balance/transactions
```

## 📚 下一步

- [ ] 接入在线支付（微信/支付宝/Stripe）
- [ ] 添加发票管理
- [ ] 完善前端页面（主页、模型列表）
- [ ] 支持多个上游供应商
- [ ] 组织（Organization）功能

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

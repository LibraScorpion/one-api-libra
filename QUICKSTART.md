# 万联APIrouter 快速开始指南

## 前置条件

- Docker & Docker Compose
- OpenRouter API Key（从 https://openrouter.ai/keys 获取）

## 5 分钟快速部署

### 步骤 1：克隆代码

```bash
git clone <repository_url>
cd one-api-libra
```

### 步骤 2：配置 OpenRouter API Key

```bash
# 复制环境变量文件
cp .env.example .env

# 编辑 .env 文件，填入你的 API Key
OPENROUTER_API_KEY=sk-or-v1-xxxxxxxxxxxxxx
```

### 步骤 3：启动服务

```bash
# 启动所有服务（MySQL + Redis + Backend）
docker-compose up -d

# 查看启动日志
docker-compose logs -f backend
```

### 步骤 4：初始化数据库

```bash
# 方式 1：使用迁移工具（推荐）
docker-compose exec backend go run cmd/migrate/main.go -dir=./sql

# 方式 2：手动执行 SQL
docker-compose exec -T mysql mysql -uroot -proot123 oneapi < sql/001_add_balance_system.sql
docker-compose exec -T mysql mysql -uroot -proot123 oneapi < sql/002_seed_openrouter_data.sql
```

**⚠️ 重要：修改 OpenRouter API Key（如果步骤 2 未配置）**

```bash
# 更新渠道的 API Key
docker-compose exec mysql mysql -uroot -proot123 oneapi -e "UPDATE channels SET \`key\`='sk-or-v1-YOUR_ACTUAL_KEY' WHERE id=1000;"
```

### 步骤 5：访问应用

- **前端**: http://localhost:3000
- **管理后台**: http://localhost:3000
  - 默认账号：`root`
  - 默认密码：`123456`

### 步骤 6：测试调用

#### 6.1 管理员充值

```bash
# 1. 登录管理后台获取 Token
curl -X POST http://localhost:3000/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"root","password":"123456"}'

# 2. 为用户充值 $10
curl -X POST http://localhost:3000/api/balance/admin/add \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "amount": 10.00,
    "description": "测试充值"
  }'
```

#### 6.2 创建 API Key

```bash
# 在前端创建，或使用 API：
curl -X POST http://localhost:3000/api/token \
  -H "Authorization: Bearer {user_token}" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Key"}'
```

#### 6.3 调用 OpenAI 兼容接口

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer {api_key}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello! This is a test."}
    ]
  }'
```

#### 6.4 查看余额和消费记录

```bash
# 查询余额
curl http://localhost:3000/api/balance \
  -H "Authorization: Bearer {user_token}"

# 查询交易记录
curl "http://localhost:3000/api/balance/transactions?page=1" \
  -H "Authorization: Bearer {user_token}"
```

## 常用命令

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f backend

# 重启服务
docker-compose restart backend

# 停止所有服务
docker-compose down

# 清理数据（⚠️ 会删除数据库）
docker-compose down -v
```

## 数据库管理

访问 Adminer（Web 数据库管理工具）：http://localhost:8080

- 服务器：`mysql`
- 用户名：`root`
- 密码：`root123`
- 数据库：`oneapi`

## 疑难排查

### 问题 1：服务启动失败

```bash
# 检查日志
docker-compose logs backend

# 常见原因：
# - 端口被占用（修改 .env 中的 SERVER_PORT）
# - 数据库连接失败（检查 MySQL 是否正常启动）
```

### 问题 2：OpenRouter 调用失败（401）

```bash
# 检查 API Key 是否正确配置
docker-compose exec mysql mysql -uroot -proot123 oneapi -e "SELECT id, name, \`key\` FROM channels WHERE id=1000;"

# 更新 API Key
docker-compose exec mysql mysql -uroot -proot123 oneapi -e "UPDATE channels SET \`key\`='sk-or-v1-CORRECT_KEY' WHERE id=1000;"
```

### 问题 3：余额扣除不正确

```bash
# 检查模型定价
curl http://localhost:3000/api/models/pricing

# 查看日志中的成本计算
docker-compose logs backend | grep "成本"
```

## 下一步

- 📚 阅读 [README_OPENROUTER.md](./README_OPENROUTER.md) 了解详细功能
- 🔧 查看 [API 文档](./README_OPENROUTER.md#-api-接口)
- 🎨 自定义前端页面
- 💳 集成在线支付

## 获取帮助

- GitHub Issues: [提交问题](https://github.com/your-repo/issues)
- 文档: [完整文档](./第一阶段_MVP实施文档.md)

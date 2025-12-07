# OpenRouter 页面功能分析与开发实现方案

## 📊 页面功能分析

### 1. 主页 (/)
**功能特性**：
- 产品介绍和核心价值展示
- 快速开始向导
- 模型搜索框
- 热门模型推荐
- 实时统计数据（请求量、模型数量等）
- 用户登录/注册入口

**关键元素**：
- Hero Banner（主视觉区）
- 模型搜索组件
- 统计卡片
- 快速链接导航

### 2. 模型列表页 (/models)
**功能特性**：
- 模型卡片网格展示
- 多维度筛选（供应商、能力、价格、上下文长度）
- 搜索功能
- 排序选项（价格、速度、质量）
- 模型详情弹窗
- 价格对比功能

**数据展示**：
- 模型名称和供应商
- 输入/输出价格
- 上下文窗口大小
- 速度评分
- 功能标签（Chat、Code、Vision等）

### 3. 排行榜页 (/rankings)
**功能特性**：
- 模型性能排名
- 多维度榜单（质量、速度、性价比）
- 时间周期切换（日、周、月）
- 用量趋势图表
- 社区评分展示

**排行维度**：
- 使用量排行
- 性价比排行
- 速度排行
- 用户评分排行

### 4. 额度管理页 (/settings/credits)
**功能特性**：
- 余额显示
- 充值功能
- 消费记录
- 用量预警设置
- 自动充值配置
- 发票管理

**支付集成**：
- 信用卡支付
- PayPal
- 加密货币
- 余额转账

### 5. API Keys 管理页 (/settings/keys)
**功能特性**：
- API Key 列表
- 创建新 Key
- Key 权限设置
- 使用限制配置
- 访问日志
- Key 轮换管理

**安全功能**：
- IP 白名单
- 请求限流
- 模型访问权限
- 过期时间设置

### 6. 活动记录页 (/activity)
**功能特性**：
- 请求历史列表
- 详细日志展示
- 时间范围筛选
- 模型使用统计
- 成本分析
- 错误追踪

**数据分析**：
- Token 用量统计
- 成本趋势图
- 模型使用分布
- 错误率分析

### 7. 偏好设置页 (/settings/preferences)
**功能特性**：
- 界面主题设置
- 语言偏好
- 通知设置
- 默认参数配置
- 隐私设置
- 数据导出

---

## 🏗️ 系统架构设计

### 前端技术栈
```
- React 18 + TypeScript
- Material-UI / Ant Design
- React Router v6
- Redux Toolkit (状态管理)
- Recharts (图表)
- Axios (API请求)
```

### 后端技术栈
```
- Go + Gin Framework
- GORM (ORM)
- Redis (缓存/会话)
- MySQL/PostgreSQL
- JWT (认证)
- Google OAuth2
```

### 页面路由结构
```
/                          # 主页
/login                     # 登录页
/dashboard                 # 用户仪表盘
├── /models               # 模型列表
├── /rankings             # 排行榜
├── /credits              # 额度管理
├── /keys                 # API Keys
├── /activity             # 活动记录
├── /settings             # 设置
│   ├── /preferences      # 偏好设置
│   ├── /profile         # 个人资料
│   └── /security        # 安全设置
└── /admin                # 管理后台
```

---

## 🔌 后端API接口设计

### 认证相关 API
```go
POST   /api/auth/google           # Google OAuth登录
POST   /api/auth/refresh          # 刷新Token
POST   /api/auth/logout           # 登出
GET    /api/auth/user             # 获取当前用户信息
```

### 模型相关 API
```go
GET    /api/models                # 获取模型列表
GET    /api/models/:id            # 获取模型详情
GET    /api/models/search         # 搜索模型
GET    /api/models/rankings       # 获取模型排行
POST   /api/models/compare        # 模型对比
```

### 额度管理 API
```go
GET    /api/credits/balance       # 获取余额
POST   /api/credits/recharge      # 充值
GET    /api/credits/transactions  # 交易记录
POST   /api/credits/alerts        # 设置预警
```

### API Key 管理
```go
GET    /api/keys                  # 获取Key列表
POST   /api/keys                  # 创建新Key
PUT    /api/keys/:id             # 更新Key设置
DELETE /api/keys/:id              # 删除Key
GET    /api/keys/:id/logs        # 获取Key使用日志
```

### 活动记录 API
```go
GET    /api/activity/logs         # 获取活动日志
GET    /api/activity/stats        # 获取统计数据
GET    /api/activity/usage        # 获取用量分析
POST   /api/activity/export       # 导出数据
```

### 设置相关 API
```go
GET    /api/settings/preferences  # 获取偏好设置
PUT    /api/settings/preferences  # 更新偏好设置
GET    /api/settings/profile      # 获取个人资料
PUT    /api/settings/profile      # 更新个人资料
```

---

## 🔐 Google OAuth 登录实现

### 1. 前端实现
```tsx
// components/GoogleLogin.tsx
import { GoogleOAuthProvider, GoogleLogin } from '@react-oauth/google';

const GOOGLE_CLIENT_ID = process.env.REACT_APP_GOOGLE_CLIENT_ID;

export const GoogleLoginButton = () => {
  const handleSuccess = async (credentialResponse) => {
    const { credential } = credentialResponse;

    // 发送到后端验证
    const response = await fetch('/api/auth/google', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: credential })
    });

    const data = await response.json();
    if (data.success) {
      // 保存JWT token
      localStorage.setItem('access_token', data.access_token);
      // 跳转到仪表盘
      window.location.href = '/dashboard';
    }
  };

  return (
    <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
      <GoogleLogin
        onSuccess={handleSuccess}
        onError={() => console.log('Login Failed')}
        theme="outline"
        size="large"
        text="signin_with"
        shape="rectangular"
      />
    </GoogleOAuthProvider>
  );
};
```

### 2. 后端实现
```go
// controller/auth_google.go
package controller

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/gin-gonic/gin"
    "google.golang.org/api/oauth2/v2"
    "google.golang.org/api/option"
)

type GoogleAuthRequest struct {
    Token string `json:"token" binding:"required"`
}

func GoogleAuth(c *gin.Context) {
    var req GoogleAuthRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // 验证Google token
    ctx := context.Background()
    oauth2Service, err := oauth2.NewService(ctx, option.WithHTTPClient(http.DefaultClient))
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create oauth2 service"})
        return
    }

    tokenInfo, err := oauth2Service.Tokeninfo().
        IdToken(req.Token).
        Context(ctx).
        Do()

    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid token"})
        return
    }

    // 检查或创建用户
    user, err := model.GetOrCreateUserByGoogle(tokenInfo.Email, tokenInfo.UserId)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create user"})
        return
    }

    // 生成JWT
    accessToken, _ := GenerateJWT(user.ID)
    refreshToken, _ := GenerateRefreshToken(user.ID)

    c.JSON(200, gin.H{
        "success": true,
        "access_token": accessToken,
        "refresh_token": refreshToken,
        "user": user,
    })
}
```

---

## 📁 页面组件实现示例

### 模型列表页组件
```tsx
// pages/Models.tsx
import React, { useState, useEffect } from 'react';
import {
  Grid, Card, CardContent, Typography, Chip,
  TextField, Select, MenuItem, Slider, Button,
  InputAdornment, Box, Pagination
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { fetchModels } from '../api/models';

interface Model {
  id: string;
  name: string;
  provider: string;
  description: string;
  pricing: {
    input: number;
    output: number;
  };
  contextLength: number;
  capabilities: string[];
  speed: number;
}

export const ModelsPage = () => {
  const [models, setModels] = useState<Model[]>([]);
  const [filteredModels, setFilteredModels] = useState<Model[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [provider, setProvider] = useState('all');
  const [priceRange, setPriceRange] = useState([0, 100]);
  const [capability, setCapability] = useState('all');
  const [sortBy, setSortBy] = useState('popular');
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadModels();
  }, []);

  const loadModels = async () => {
    setLoading(true);
    try {
      const data = await fetchModels();
      setModels(data);
      setFilteredModels(data);
    } catch (error) {
      console.error('Failed to load models:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (event: React.ChangeEvent<HTMLInputElement>) => {
    const term = event.target.value.toLowerCase();
    setSearchTerm(term);
    filterModels(term, provider, capability, priceRange);
  };

  const filterModels = (search: string, prov: string, cap: string, price: number[]) => {
    let filtered = models;

    // 搜索过滤
    if (search) {
      filtered = filtered.filter(m =>
        m.name.toLowerCase().includes(search) ||
        m.description.toLowerCase().includes(search)
      );
    }

    // 供应商过滤
    if (prov !== 'all') {
      filtered = filtered.filter(m => m.provider === prov);
    }

    // 能力过滤
    if (cap !== 'all') {
      filtered = filtered.filter(m => m.capabilities.includes(cap));
    }

    // 价格过滤
    filtered = filtered.filter(m =>
      m.pricing.input >= price[0] && m.pricing.input <= price[1]
    );

    // 排序
    switch (sortBy) {
      case 'price-low':
        filtered.sort((a, b) => a.pricing.input - b.pricing.input);
        break;
      case 'price-high':
        filtered.sort((a, b) => b.pricing.input - a.pricing.input);
        break;
      case 'speed':
        filtered.sort((a, b) => b.speed - a.speed);
        break;
      case 'context':
        filtered.sort((a, b) => b.contextLength - a.contextLength);
        break;
    }

    setFilteredModels(filtered);
    setPage(1);
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Typography variant="h4" gutterBottom>
        AI 模型目录
      </Typography>
      <Typography variant="body1" color="textSecondary" paragraph>
        浏览和比较来自不同供应商的 AI 模型
      </Typography>

      {/* 搜索和筛选栏 */}
      <Grid container spacing={2} sx={{ mb: 4 }}>
        <Grid item xs={12} md={4}>
          <TextField
            fullWidth
            placeholder="搜索模型..."
            value={searchTerm}
            onChange={handleSearch}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
          />
        </Grid>
        <Grid item xs={12} md={2}>
          <Select
            fullWidth
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
            displayEmpty
          >
            <MenuItem value="all">所有供应商</MenuItem>
            <MenuItem value="openai">OpenAI</MenuItem>
            <MenuItem value="anthropic">Anthropic</MenuItem>
            <MenuItem value="google">Google</MenuItem>
            <MenuItem value="meta">Meta</MenuItem>
          </Select>
        </Grid>
        <Grid item xs={12} md={2}>
          <Select
            fullWidth
            value={capability}
            onChange={(e) => setCapability(e.target.value)}
            displayEmpty
          >
            <MenuItem value="all">所有能力</MenuItem>
            <MenuItem value="chat">对话</MenuItem>
            <MenuItem value="code">代码</MenuItem>
            <MenuItem value="vision">视觉</MenuItem>
            <MenuItem value="function">函数调用</MenuItem>
          </Select>
        </Grid>
        <Grid item xs={12} md={2}>
          <Select
            fullWidth
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value)}
          >
            <MenuItem value="popular">最受欢迎</MenuItem>
            <MenuItem value="price-low">价格从低到高</MenuItem>
            <MenuItem value="price-high">价格从高到低</MenuItem>
            <MenuItem value="speed">速度最快</MenuItem>
            <MenuItem value="context">上下文最大</MenuItem>
          </Select>
        </Grid>
      </Grid>

      {/* 价格范围滑块 */}
      <Box sx={{ mb: 4 }}>
        <Typography gutterBottom>
          价格范围: ${priceRange[0]} - ${priceRange[1]} / 1M tokens
        </Typography>
        <Slider
          value={priceRange}
          onChange={(e, value) => setPriceRange(value as number[])}
          valueLabelDisplay="auto"
          min={0}
          max={100}
          marks={[
            { value: 0, label: '$0' },
            { value: 25, label: '$25' },
            { value: 50, label: '$50' },
            { value: 100, label: '$100' }
          ]}
        />
      </Box>

      {/* 模型卡片网格 */}
      <Grid container spacing={3}>
        {filteredModels.slice((page - 1) * 12, page * 12).map((model) => (
          <Grid item xs={12} sm={6} md={4} key={model.id}>
            <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
              <CardContent sx={{ flexGrow: 1 }}>
                <Typography variant="h6" gutterBottom>
                  {model.name}
                </Typography>
                <Chip
                  label={model.provider}
                  size="small"
                  color="primary"
                  sx={{ mb: 1 }}
                />
                <Typography variant="body2" color="textSecondary" paragraph>
                  {model.description}
                </Typography>

                {/* 价格信息 */}
                <Box sx={{ mt: 2 }}>
                  <Typography variant="subtitle2" color="primary">
                    输入: ${model.pricing.input}/1M tokens
                  </Typography>
                  <Typography variant="subtitle2" color="primary">
                    输出: ${model.pricing.output}/1M tokens
                  </Typography>
                </Box>

                {/* 特性标签 */}
                <Box sx={{ mt: 2 }}>
                  {model.capabilities.map((cap) => (
                    <Chip
                      key={cap}
                      label={cap}
                      size="small"
                      variant="outlined"
                      sx={{ mr: 0.5, mb: 0.5 }}
                    />
                  ))}
                </Box>

                {/* 规格信息 */}
                <Box sx={{ mt: 2 }}>
                  <Typography variant="caption" display="block">
                    上下文: {model.contextLength.toLocaleString()} tokens
                  </Typography>
                  <Typography variant="caption" display="block">
                    速度评分: {model.speed}/10
                  </Typography>
                </Box>
              </CardContent>

              <Box sx={{ p: 2 }}>
                <Button fullWidth variant="outlined" size="small">
                  查看详情
                </Button>
              </Box>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* 分页 */}
      <Box sx={{ mt: 4, display: 'flex', justifyContent: 'center' }}>
        <Pagination
          count={Math.ceil(filteredModels.length / 12)}
          page={page}
          onChange={(e, value) => setPage(value)}
          color="primary"
        />
      </Box>
    </Box>
  );
};
```

---

## 🗂️ 数据库设计

### 核心数据表
```sql
-- 用户表（支持Google OAuth）
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,
    google_id VARCHAR(255) UNIQUE,
    name VARCHAR(100),
    avatar VARCHAR(500),
    locale VARCHAR(10) DEFAULT 'zh-CN',
    theme VARCHAR(20) DEFAULT 'light',
    status ENUM('active', 'suspended') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_google_id (google_id),
    INDEX idx_email (email)
);

-- 模型表
CREATE TABLE models (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(200),
    provider VARCHAR(50) NOT NULL,
    description TEXT,
    context_length INT,
    max_tokens INT,
    pricing_input DECIMAL(12, 8),
    pricing_output DECIMAL(12, 8),
    speed_score INT DEFAULT 5,
    quality_score INT DEFAULT 5,
    capabilities JSON,
    parameters JSON,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_provider (provider),
    INDEX idx_is_active (is_active),
    FULLTEXT idx_search (name, display_name, description)
);

-- 模型使用统计表
CREATE TABLE model_stats (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    model_id BIGINT NOT NULL,
    date DATE NOT NULL,
    request_count BIGINT DEFAULT 0,
    token_count BIGINT DEFAULT 0,
    error_count INT DEFAULT 0,
    avg_latency INT,
    total_cost DECIMAL(20, 8),
    UNIQUE KEY unique_model_date (model_id, date),
    FOREIGN KEY (model_id) REFERENCES models(id),
    INDEX idx_date (date)
);

-- 用户偏好设置表
CREATE TABLE user_preferences (
    user_id BIGINT PRIMARY KEY,
    theme VARCHAR(20) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'zh-CN',
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    email_notifications BOOLEAN DEFAULT TRUE,
    webhook_url VARCHAR(500),
    default_model VARCHAR(100),
    default_parameters JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 信用额度交易表
CREATE TABLE credit_transactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    type ENUM('recharge', 'consume', 'refund', 'bonus') NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    balance_after DECIMAL(20, 8) NOT NULL,
    payment_method VARCHAR(50),
    payment_id VARCHAR(255),
    description VARCHAR(500),
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_created (user_id, created_at DESC),
    INDEX idx_payment_id (payment_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

---

## 🚀 实施计划

### 第一阶段：基础功能（1周）
1. ✅ Google OAuth 登录集成
2. ✅ 用户系统和会话管理
3. ✅ 基础页面框架搭建
4. ✅ 模型列表展示

### 第二阶段：核心功能（1周）
1. ⏳ API Key 管理系统
2. ⏳ 额度充值和管理
3. ⏳ 活动记录和日志
4. ⏳ 基础统计功能

### 第三阶段：高级功能（1周）
1. ⏳ 模型排行榜
2. ⏳ 数据分析和图表
3. ⏳ 用户偏好设置
4. ⏳ 批量操作和导出

### 第四阶段：优化完善（3天）
1. ⏳ 性能优化
2. ⏳ 缓存策略
3. ⏳ 错误处理
4. ⏳ 测试和文档

---

这个分析文档包含了OpenRouter所有主要页面的功能分析和完整的开发实现方案，包括前后端代码示例和数据库设计。
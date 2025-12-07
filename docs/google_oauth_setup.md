# Google OAuth 集成配置指南

## 📋 前置准备

### 1. 创建 Google Cloud 项目

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 启用 Google+ API 和 Google Identity API

### 2. 配置 OAuth 2.0 凭据

1. 在 Google Cloud Console 中，导航至 **API 和服务** > **凭据**
2. 点击 **创建凭据** > **OAuth 客户端 ID**
3. 选择应用类型为 **Web 应用**
4. 配置以下信息：
   ```
   名称: One-API Libra OAuth

   已授权的 JavaScript 来源:
   - http://localhost:3000
   - http://localhost:5173
   - https://your-domain.com

   已授权的重定向 URI:
   - http://localhost:3000/api/auth/google/callback
   - https://your-domain.com/api/auth/google/callback
   ```
5. 保存后获取 **Client ID** 和 **Client Secret**

## 🔧 后端配置

### 1. 环境变量配置

编辑 `.env` 或 `.env.production` 文件：

```env
# Google OAuth 配置
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:3000/api/auth/google/callback

# 生产环境
# GOOGLE_REDIRECT_URL=https://your-domain.com/api/auth/google/callback
```

### 2. 数据库迁移

运行以下SQL添加Google OAuth相关字段：

```sql
-- 为users表添加Google OAuth字段
ALTER TABLE users
ADD COLUMN google_id VARCHAR(255) UNIQUE,
ADD COLUMN avatar VARCHAR(500),
ADD COLUMN locale VARCHAR(10) DEFAULT 'zh-CN',
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN last_login_at BIGINT,
ADD COLUMN last_login_ip VARCHAR(45),
ADD INDEX idx_google_id (google_id);

-- 创建用户偏好设置表
CREATE TABLE IF NOT EXISTS user_preferences (
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

-- 创建用户活动日志表
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    action VARCHAR(50),
    resource VARCHAR(50),
    resource_id VARCHAR(100),
    details JSON,
    ip_address VARCHAR(45),
    user_agent VARCHAR(500),
    status VARCHAR(20),
    error_msg TEXT,
    request_time BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_activity (user_id, created_at DESC),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### 3. 后端代码集成

在 `router/main.go` 中添加Google OAuth路由：

```go
// 添加Google OAuth路由
authGroup := router.Group("/api/auth")
{
    authGroup.GET("/google", controller.GoogleLoginHandler)
    authGroup.GET("/google/callback", controller.GoogleCallbackHandler)
    authGroup.POST("/google/token", controller.GoogleTokenLoginHandler)
    authGroup.POST("/google/link", middleware.UserAuth(), controller.LinkGoogleAccount)
    authGroup.DELETE("/google/link", middleware.UserAuth(), controller.UnlinkGoogleAccount)
}
```

## 🎨 前端配置

### 1. 安装依赖

```bash
cd web
npm install @react-oauth/google axios
# 或使用 pnpm
pnpm add @react-oauth/google axios
```

### 2. 环境变量配置

创建 `web/.env` 文件：

```env
REACT_APP_GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
REACT_APP_API_URL=http://localhost:3000
```

### 3. 前端集成

在 `App.tsx` 中包裹Google OAuth Provider：

```tsx
import { GoogleAuthProvider } from './components/GoogleLogin';

function App() {
  return (
    <GoogleAuthProvider>
      {/* 你的应用组件 */}
      <Router>
        <Routes>
          {/* 路由配置 */}
        </Routes>
      </Router>
    </GoogleAuthProvider>
  );
}
```

### 4. 登录页面使用

```tsx
import { GoogleAuthButton } from '../components/GoogleLogin';

function LoginPage() {
  return (
    <div>
      <h1>登录</h1>

      {/* Google登录按钮 */}
      <GoogleAuthButton
        mode="signin"
        onSuccess={(user) => {
          console.log('登录成功', user);
        }}
        onError={(error) => {
          console.error('登录失败', error);
        }}
      />

      {/* 或使用自定义样式按钮 */}
      <CustomGoogleButton mode="signin" />
    </div>
  );
}
```

## 🚀 运行测试

### 1. 启动后端服务

```bash
# 开发模式
./run.sh dev

# 或使用Docker
./run.sh prod
```

### 2. 启动前端开发服务器

```bash
cd web
npm start
# 或
pnpm dev
```

### 3. 测试流程

1. 访问 http://localhost:5173/login
2. 点击 "使用 Google 登录"
3. 选择或输入Google账号
4. 授权应用访问
5. 自动跳转回仪表盘

## 🔒 安全配置

### 1. CORS配置

确保后端允许前端域名的跨域请求：

```go
// middleware/cors.go
func CORS() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins: []string{
            "http://localhost:3000",
            "http://localhost:5173",
            "https://your-domain.com",
        },
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
    })
}
```

### 2. HTTPS配置（生产环境）

使用Nginx反向代理配置SSL：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 📊 功能测试清单

- [x] Google账号登录
- [x] Google账号注册（自动创建用户）
- [x] 关联Google账号到现有账号
- [x] 取消Google账号关联
- [x] 用户信息同步（头像、邮箱）
- [x] JWT令牌生成
- [x] 会话管理
- [x] 活动日志记录

## 🐛 常见问题

### 1. "redirect_uri_mismatch" 错误

**原因**: 回调URL不匹配
**解决**: 在Google Cloud Console中添加正确的重定向URI

### 2. "invalid_client" 错误

**原因**: Client ID或Secret错误
**解决**: 检查环境变量中的配置是否正确

### 3. 跨域问题

**原因**: CORS未正确配置
**解决**: 确保后端CORS中间件包含前端域名

### 4. Token验证失败

**原因**: Google Token过期或无效
**解决**: 确保使用最新的Token，检查网络时间同步

## 📚 参考资源

- [Google OAuth 2.0 文档](https://developers.google.com/identity/protocols/oauth2)
- [Google Sign-In for Web](https://developers.google.com/identity/gsi/web)
- [@react-oauth/google 文档](https://www.npmjs.com/package/@react-oauth/google)
- [JWT 认证最佳实践](https://jwt.io/introduction)

## 🎯 下一步

1. 实现其他OAuth提供商（GitHub、Microsoft等）
2. 添加两步验证（2FA）
3. 实现单点登录（SSO）
4. 添加账号安全设置
5. 实现会话管理功能
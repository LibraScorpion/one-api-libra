#!/bin/bash

# One-API 生产环境部署脚本
# 自动化部署、配置和健康检查

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/backups/$(date +%Y%m%d_%H%M%S)"
LOG_FILE="${SCRIPT_DIR}/deploy_$(date +%Y%m%d).log"

# 日志函数
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

# 系统检查
check_system() {
    print_info "系统环境检查..."

    # 检查操作系统
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
    else
        print_error "不支持的操作系统: $OSTYPE"
        exit 1
    fi

    # 检查必要命令
    local commands=("docker" "docker-compose" "git" "curl")
    for cmd in "${commands[@]}"; do
        if ! command -v $cmd &> /dev/null; then
            print_error "$cmd 未安装，请先安装"
            exit 1
        fi
    done

    # 检查Docker服务
    if ! docker info &> /dev/null; then
        print_error "Docker服务未运行"
        exit 1
    fi

    # 检查端口占用
    local ports=("3000" "3306" "6379")
    for port in "${ports[@]}"; do
        if lsof -i :$port &> /dev/null; then
            print_warning "端口 $port 已被占用"
            echo -n "是否强制停止占用的服务？(y/N): "
            read -r answer
            if [[ "$answer" == "y" || "$answer" == "Y" ]]; then
                fuser -k $port/tcp &> /dev/null || true
            else
                exit 1
            fi
        fi
    done

    print_success "系统检查通过"
}

# 初始化配置
init_config() {
    print_info "初始化配置..."

    # 生成安全的密钥
    local session_secret=$(openssl rand -base64 32)
    local mysql_root_pwd=$(openssl rand -base64 16)
    local mysql_user_pwd=$(openssl rand -base64 16)
    local initial_token=$(openssl rand -base64 16)

    # 创建生产环境配置
    cat > .env.production << EOF
# One-API 生产环境配置
# Generated at $(date)

# 基础配置
PORT=3000
GIN_MODE=release
DEBUG=false

# 数据库配置
SQL_DSN=oneapi:${mysql_user_pwd}@tcp(db:3306)/one-api
MYSQL_ROOT_PASSWORD=${mysql_root_pwd}
MYSQL_USER=oneapi
MYSQL_PASSWORD=${mysql_user_pwd}
MYSQL_DATABASE=one-api

# Redis配置
REDIS_CONN_STRING=redis://redis:6379
REDIS_PASSWORD=

# 安全配置
SESSION_SECRET=${session_secret}
INITIAL_ROOT_TOKEN=${initial_token}

# OpenRouter API配置（请填写您的API Key）
OPENROUTER_API_KEY=

# 系统配置
TZ=Asia/Shanghai
MEMORY_CACHE_ENABLED=true
SYNC_FREQUENCY=60
BATCH_UPDATE_ENABLED=true
BATCH_UPDATE_INTERVAL=60

# 监控配置
ENABLE_METRIC=true
CHANNEL_TEST_FREQUENCY=1800

# 日志配置
LOG_LEVEL=info
LOG_SQL_ENABLED=false
EOF

    print_success "配置文件创建完成"
    print_warning "请编辑 .env.production 文件，填写您的 OpenRouter API Key"

    # 记录管理员信息
    echo "" >> .env.production
    echo "# 管理员信息（首次登录使用）" >> .env.production
    echo "# Initial Admin Token: ${initial_token}" >> .env.production

    print_info "初始管理员令牌: ${initial_token}"
    print_warning "请妥善保存此令牌，用于首次登录"
}

# 备份数据
backup_data() {
    if [ -d "data" ] && [ "$(ls -A data)" ]; then
        print_info "备份现有数据..."
        mkdir -p "$BACKUP_DIR"

        # 备份数据目录
        if [ -d "data/mysql" ]; then
            tar -czf "$BACKUP_DIR/mysql_backup.tar.gz" data/mysql
            print_success "MySQL数据备份完成"
        fi

        if [ -d "data/oneapi" ]; then
            tar -czf "$BACKUP_DIR/oneapi_backup.tar.gz" data/oneapi
            print_success "OneAPI数据备份完成"
        fi

        # 备份配置文件
        cp .env* "$BACKUP_DIR/" 2>/dev/null || true

        print_success "备份完成: $BACKUP_DIR"
    fi
}

# 创建Docker Compose配置
create_docker_compose() {
    print_info "创建Docker Compose配置..."

    cat > docker-compose.production.yml << 'EOF'
version: '3.8'

services:
  one-api:
    image: justsong/one-api:latest
    container_name: one-api-prod
    restart: unless-stopped
    command: --log-dir /app/logs
    ports:
      - "3000:3000"
    volumes:
      - ./data/oneapi:/data
      - ./logs:/app/logs
      - /etc/localtime:/etc/localtime:ro
    env_file:
      - .env.production
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:3000/api/status"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - one-api-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  db:
    image: mysql:8.2.0
    container_name: mysql-prod
    restart: unless-stopped
    volumes:
      - ./data/mysql:/var/lib/mysql
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    env_file:
      - .env.production
    ports:
      - "127.0.0.1:3306:3306"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks:
      - one-api-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  redis:
    image: redis:7-alpine
    container_name: redis-prod
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - ./data/redis:/data
    ports:
      - "127.0.0.1:6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - one-api-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

networks:
  one-api-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
EOF

    print_success "Docker Compose配置创建完成"
}

# 创建初始化SQL
create_init_sql() {
    print_info "创建数据库初始化脚本..."

    cat > init.sql << 'EOF'
-- One-API 数据库初始化脚本

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `one-api` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

USE `one-api`;

-- 设置时区
SET time_zone = '+08:00';

-- 创建索引优化
-- 这里可以添加自定义的索引优化语句

SET FOREIGN_KEY_CHECKS = 1;
EOF

    print_success "初始化脚本创建完成"
}

# 部署服务
deploy_service() {
    print_info "开始部署服务..."

    # 拉取最新镜像
    print_info "拉取Docker镜像..."
    docker-compose -f docker-compose.production.yml pull

    # 启动服务
    print_info "启动服务..."
    docker-compose -f docker-compose.production.yml up -d

    # 等待服务启动
    print_info "等待服务启动..."
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:3000/api/status | grep -q "success"; then
            print_success "服务启动成功！"
            break
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    if [ $attempt -eq $max_attempts ]; then
        print_error "服务启动超时"
        docker-compose -f docker-compose.production.yml logs
        exit 1
    fi
}

# 配置Nginx反向代理
setup_nginx() {
    print_info "配置Nginx反向代理..."

    cat > nginx.conf << 'EOF'
server {
    listen 80;
    server_name your-domain.com;  # 修改为您的域名

    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;  # 修改为您的域名

    # SSL证书配置
    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;

    # SSL优化
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # 安全头部
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # 日志
    access_log /var/log/nginx/one-api-access.log;
    error_log /var/log/nginx/one-api-error.log;

    # 客户端限制
    client_max_body_size 100M;
    client_body_timeout 120s;

    # 反向代理到One-API
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;

        # WebSocket支持
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 代理头部
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时设置
        proxy_connect_timeout 300s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;

        # 缓冲设置
        proxy_buffering off;
        proxy_request_buffering off;
    }

    # API限流
    location /v1/ {
        limit_req zone=api_limit burst=20 nodelay;
        proxy_pass http://localhost:3000;
    }
}

# API限流区域（添加到http块）
# limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
EOF

    print_success "Nginx配置文件已生成: nginx.conf"
    print_info "请将此配置复制到您的Nginx配置目录"
}

# 系统监控设置
setup_monitoring() {
    print_info "设置系统监控..."

    # 创建监控脚本
    cat > monitor.sh << 'EOF'
#!/bin/bash

# 监控脚本
check_service() {
    if ! curl -s http://localhost:3000/api/status | grep -q "success"; then
        echo "[$(date)] Service is down, attempting restart..."
        docker-compose -f docker-compose.production.yml restart one-api

        # 发送告警（这里可以集成邮件或其他通知方式）
        # echo "One-API service is down" | mail -s "Service Alert" admin@example.com
    fi
}

check_disk_space() {
    local usage=$(df -h / | awk 'NR==2 {print $(NF-1)}' | sed 's/%//')
    if [ "$usage" -gt 80 ]; then
        echo "[$(date)] Disk usage is high: ${usage}%"
        # 发送告警
    fi
}

check_memory() {
    local mem_usage=$(free | awk 'NR==2 {printf "%.0f", $3*100/$2}')
    if [ "$mem_usage" -gt 80 ]; then
        echo "[$(date)] Memory usage is high: ${mem_usage}%"
        # 发送告警
    fi
}

# 执行检查
check_service
check_disk_space
check_memory
EOF

    chmod +x monitor.sh

    # 添加到crontab（每5分钟检查一次）
    print_info "配置定时监控任务..."
    (crontab -l 2>/dev/null; echo "*/5 * * * * ${SCRIPT_DIR}/monitor.sh >> ${SCRIPT_DIR}/monitor.log 2>&1") | crontab -

    print_success "监控设置完成"
}

# 性能优化
optimize_performance() {
    print_info "执行性能优化..."

    # 优化Docker
    cat > daemon.json << 'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ]
}
EOF

    print_info "Docker优化配置已生成: daemon.json"
    print_info "请将此文件复制到 /etc/docker/daemon.json"

    # MySQL优化配置
    cat > mysql-optimization.cnf << 'EOF'
[mysqld]
# 基础优化
max_connections = 500
connect_timeout = 10
wait_timeout = 600
interactive_timeout = 600

# 缓冲池优化
innodb_buffer_pool_size = 1G
innodb_buffer_pool_instances = 4

# 日志优化
innodb_log_file_size = 256M
innodb_log_buffer_size = 64M
innodb_flush_log_at_trx_commit = 2

# 查询缓存
query_cache_type = 1
query_cache_size = 128M
query_cache_limit = 4M

# 临时表
tmp_table_size = 128M
max_heap_table_size = 128M

# 慢查询日志
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 2
EOF

    print_success "MySQL优化配置已生成: mysql-optimization.cnf"
}

# 健康检查
health_check() {
    print_info "执行健康检查..."

    local all_healthy=true

    # 检查API服务
    echo -n "API服务: "
    if curl -s http://localhost:3000/api/status | grep -q "success"; then
        echo -e "${GREEN}正常${NC}"
    else
        echo -e "${RED}异常${NC}"
        all_healthy=false
    fi

    # 检查数据库
    echo -n "MySQL数据库: "
    if docker exec mysql-prod mysqladmin ping -h localhost &> /dev/null; then
        echo -e "${GREEN}正常${NC}"
    else
        echo -e "${RED}异常${NC}"
        all_healthy=false
    fi

    # 检查Redis
    echo -n "Redis缓存: "
    if docker exec redis-prod redis-cli ping &> /dev/null; then
        echo -e "${GREEN}正常${NC}"
    else
        echo -e "${RED}异常${NC}"
        all_healthy=false
    fi

    # 检查磁盘空间
    echo -n "磁盘空间: "
    local usage=$(df -h / | awk 'NR==2 {print $(NF-1)}' | sed 's/%//')
    if [ "$usage" -lt 80 ]; then
        echo -e "${GREEN}充足 (${usage}% 使用)${NC}"
    else
        echo -e "${YELLOW}警告 (${usage}% 使用)${NC}"
    fi

    # 检查内存
    echo -n "内存使用: "
    local mem_usage=$(free | awk 'NR==2 {printf "%.0f", $3*100/$2}')
    if [ "$mem_usage" -lt 80 ]; then
        echo -e "${GREEN}正常 (${mem_usage}% 使用)${NC}"
    else
        echo -e "${YELLOW}警告 (${mem_usage}% 使用)${NC}"
    fi

    if [ "$all_healthy" = true ]; then
        print_success "所有服务健康检查通过"
    else
        print_error "部分服务存在问题，请检查日志"
    fi
}

# 显示部署信息
show_info() {
    echo ""
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║           One-API 部署完成                           ║"
    echo "╚═══════════════════════════════════════════════════════╝"
    echo ""
    echo -e "${CYAN}访问地址:${NC} http://localhost:3000"
    echo -e "${CYAN}管理后台:${NC} http://localhost:3000/admin"
    echo -e "${CYAN}API文档:${NC} http://localhost:3000/docs"
    echo ""
    echo -e "${YELLOW}重要提示:${NC}"
    echo "1. 请编辑 .env.production 文件配置 OpenRouter API Key"
    echo "2. 首次登录使用生成的管理员令牌"
    echo "3. 建议配置SSL证书和域名"
    echo "4. 定期备份 data/ 目录"
    echo ""
    echo -e "${GREEN}常用命令:${NC}"
    echo "  查看日志: docker-compose -f docker-compose.production.yml logs -f"
    echo "  重启服务: docker-compose -f docker-compose.production.yml restart"
    echo "  停止服务: docker-compose -f docker-compose.production.yml down"
    echo "  备份数据: ./deploy.sh backup"
    echo "  健康检查: ./deploy.sh health"
    echo ""
}

# 主函数
main() {
    case "$1" in
        install)
            log "开始安装部署 One-API..."
            check_system
            backup_data
            init_config
            create_docker_compose
            create_init_sql
            deploy_service
            setup_monitoring
            optimize_performance
            health_check
            show_info
            ;;
        upgrade)
            log "开始升级 One-API..."
            backup_data
            docker-compose -f docker-compose.production.yml pull
            docker-compose -f docker-compose.production.yml up -d
            health_check
            ;;
        backup)
            backup_data
            ;;
        health)
            health_check
            ;;
        nginx)
            setup_nginx
            ;;
        monitor)
            setup_monitoring
            ;;
        optimize)
            optimize_performance
            ;;
        *)
            echo "使用方法: $0 {install|upgrade|backup|health|nginx|monitor|optimize}"
            echo ""
            echo "命令说明:"
            echo "  install  - 全新安装部署"
            echo "  upgrade  - 升级到最新版本"
            echo "  backup   - 备份数据"
            echo "  health   - 健康检查"
            echo "  nginx    - 生成Nginx配置"
            echo "  monitor  - 设置监控"
            echo "  optimize - 性能优化配置"
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
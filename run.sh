#!/bin/bash

# One-API 一键运行脚本
# 支持开发模式和生产模式

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装，请先安装 $1"
        exit 1
    fi
}

# 显示使用帮助
show_help() {
    echo "使用方法: ./run.sh [选项]"
    echo ""
    echo "选项:"
    echo "  dev          开发模式（本地运行）"
    echo "  prod         生产模式（Docker运行）"
    echo "  stop         停止服务"
    echo "  status       查看服务状态"
    echo "  logs         查看日志"
    echo "  clean        清理数据和日志"
    echo "  help         显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  ./run.sh dev     # 启动开发模式"
    echo "  ./run.sh prod    # 启动生产模式"
    echo "  ./run.sh stop    # 停止服务"
}

# 创建必要的目录
create_directories() {
    print_info "创建必要的目录..."
    mkdir -p data/oneapi
    mkdir -p data/mysql
    mkdir -p logs
    mkdir -p config
    print_success "目录创建完成"
}

# 初始化配置文件
init_config() {
    if [ ! -f .env ]; then
        print_info "创建 .env 配置文件..."
        cat > .env << EOF
# One-API 环境配置
PORT=3000
DEBUG=false
GIN_MODE=release

# 数据库配置（开发模式）
SQL_DSN=oneapi:123456@tcp(localhost:3306)/one-api

# Redis配置（开发模式）
REDIS_CONN_STRING=redis://localhost:6379

# Session密钥（请修改为随机字符串）
SESSION_SECRET=$(openssl rand -base64 32)

# 时区设置
TZ=Asia/Shanghai

# OpenRouter API配置
OPENROUTER_API_KEY=sk-or-v1-xxxxxxxxxxxxx

# 管理员账号（首次登录后请修改密码）
INITIAL_ROOT_TOKEN=admin123

# 开启内存缓存
MEMORY_CACHE_ENABLED=true
SYNC_FREQUENCY=60
EOF
        print_success ".env 文件创建完成，请编辑并配置您的 OpenRouter API Key"
        print_warning "请修改 OPENROUTER_API_KEY 为您的实际 API Key"
    fi
}

# 开发模式
run_dev() {
    print_info "启动开发模式..."

    # 检查Go环境
    check_command go

    # 检查MySQL和Redis是否运行
    print_info "检查MySQL服务..."
    if ! nc -z localhost 3306 2>/dev/null; then
        print_warning "MySQL未运行，尝试启动Docker MySQL..."
        docker run -d \
            --name mysql-dev \
            -p 3306:3306 \
            -e MYSQL_ROOT_PASSWORD='OneAPI@justsong' \
            -e MYSQL_USER=oneapi \
            -e MYSQL_PASSWORD='123456' \
            -e MYSQL_DATABASE=one-api \
            -v $(pwd)/data/mysql:/var/lib/mysql \
            mysql:8.2.0

        print_info "等待MySQL启动..."
        sleep 10
    fi

    print_info "检查Redis服务..."
    if ! nc -z localhost 6379 2>/dev/null; then
        print_warning "Redis未运行，尝试启动Docker Redis..."
        docker run -d \
            --name redis-dev \
            -p 6379:6379 \
            redis:latest
    fi

    # 设置开发环境变量
    export GIN_MODE=debug
    export DEBUG=true

    # 下载依赖
    print_info "下载Go依赖..."
    go mod download

    # 运行服务
    print_info "启动One-API服务..."
    print_success "服务启动中，访问 http://localhost:3000"
    print_info "使用 Ctrl+C 停止服务"

    # 运行主程序
    go run main.go --log-dir ./logs
}

# 生产模式（Docker Compose）
run_prod() {
    print_info "启动生产模式..."

    # 检查Docker和Docker Compose
    check_command docker
    check_command docker-compose

    # 检查配置文件
    if [ ! -f .env ]; then
        init_config
        print_error "请先编辑 .env 文件配置必要参数"
        exit 1
    fi

    # 构建并启动服务
    print_info "构建Docker镜像..."
    docker-compose build

    print_info "启动服务..."
    docker-compose up -d

    # 等待服务启动
    print_info "等待服务启动..."
    sleep 5

    # 检查服务状态
    if docker-compose ps | grep -q "Up"; then
        print_success "服务启动成功！"
        print_info "访问地址: http://localhost:3000"
        print_info "查看日志: ./run.sh logs"
    else
        print_error "服务启动失败，请查看日志"
        docker-compose logs
    fi
}

# 停止服务
stop_service() {
    print_info "停止服务..."

    # 停止开发模式的容器
    if docker ps -a | grep -q mysql-dev; then
        docker stop mysql-dev && docker rm mysql-dev
        print_success "MySQL开发容器已停止"
    fi

    if docker ps -a | grep -q redis-dev; then
        docker stop redis-dev && docker rm redis-dev
        print_success "Redis开发容器已停止"
    fi

    # 停止生产模式
    if [ -f docker-compose.yml ]; then
        docker-compose down
        print_success "Docker Compose服务已停止"
    fi

    # 杀掉Go进程
    if pgrep -f "go run main.go" > /dev/null; then
        pkill -f "go run main.go"
        print_success "Go开发进程已停止"
    fi

    print_success "所有服务已停止"
}

# 查看服务状态
check_status() {
    print_info "服务状态检查..."

    echo ""
    echo "=== Docker容器状态 ==="
    docker ps -a | grep -E "(one-api|mysql|redis)" || echo "没有运行的容器"

    echo ""
    echo "=== 端口占用情况 ==="
    echo "Port 3000 (One-API):"
    lsof -i :3000 2>/dev/null | grep LISTEN || echo "  未占用"
    echo "Port 3306 (MySQL):"
    lsof -i :3306 2>/dev/null | grep LISTEN || echo "  未占用"
    echo "Port 6379 (Redis):"
    lsof -i :6379 2>/dev/null | grep LISTEN || echo "  未占用"

    echo ""
    echo "=== API健康检查 ==="
    if curl -s http://localhost:3000/api/status | grep -q "success"; then
        print_success "API服务运行正常"
    else
        print_warning "API服务未响应或异常"
    fi
}

# 查看日志
view_logs() {
    if [ -f docker-compose.yml ] && docker-compose ps | grep -q "Up"; then
        print_info "显示Docker Compose日志..."
        docker-compose logs -f --tail=100
    elif [ -d logs ] && [ "$(ls -A logs)" ]; then
        print_info "显示本地日志..."
        tail -f logs/*.log
    else
        print_warning "没有找到日志文件"
    fi
}

# 清理数据
clean_data() {
    print_warning "此操作将清理所有数据和日志，是否继续？(y/N)"
    read -r confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        print_info "操作已取消"
        return
    fi

    stop_service

    print_info "清理数据..."
    rm -rf data/oneapi/*
    rm -rf data/mysql/*
    rm -rf logs/*

    print_success "清理完成"
}

# 快速设置（首次运行）
quick_setup() {
    print_info "执行快速设置..."

    # 创建目录
    create_directories

    # 初始化配置
    init_config

    print_success "快速设置完成！"
    print_info "请编辑 .env 文件配置必要参数"
    print_info "然后运行 ./run.sh dev 或 ./run.sh prod 启动服务"
}

# 主函数
main() {
    case "$1" in
        dev)
            create_directories
            init_config
            run_dev
            ;;
        prod)
            create_directories
            init_config
            run_prod
            ;;
        stop)
            stop_service
            ;;
        status)
            check_status
            ;;
        logs)
            view_logs
            ;;
        clean)
            clean_data
            ;;
        setup)
            quick_setup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# ASCII Logo
echo "
╔═══════════════════════════════════════════╗
║     One-API Libra 一键运行脚本           ║
║     万联APIrouter MVP版本                ║
╚═══════════════════════════════════════════╝
"

# 运行主函数
main "$@"
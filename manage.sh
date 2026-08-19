#!/bin/bash
# ============================================================
# API Gateway 管理脚本
# ============================================================

DIR="/root/projects/api-gateway"
cd "$DIR" || { echo "❌ 项目目录不存在: $DIR"; exit 1; }

case "$1" in
    start)
        echo "🚀 启动 API Gateway..."
        docker compose up -d
        echo ""
        sleep 3
        docker compose ps
        echo ""
        echo "✅ 启动完成，访问: http://$(hostname -I | awk "{print \$1}")"
        ;;
    stop)
        echo "🛑 停止 API Gateway..."
        docker compose stop
        echo "✅ 已停止"
        ;;
    down)
        echo "⚠️  停止并移除所有容器..."
        docker compose down
        echo "✅ 已清理"
        ;;
    restart)
        echo "🔄 重启 API Gateway..."
        docker compose restart
        sleep 3
        docker compose ps
        echo "✅ 重启完成"
        ;;
    status)
        echo "📊 容器状态:"
        docker compose ps
        echo ""
        echo "💾 资源占用:"
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" gateway-nginx gateway-new-api gateway-postgres gateway-redis 2>/dev/null || true
        ;;
    logs)
        SERVICE="${2:-new-api}"
        echo "📋 服务日志: $SERVICE (最后 100 行)"
        echo "---"
        docker compose logs --tail 100 "$SERVICE"
        ;;
    update)
        echo "⬆️  更新 new-api 镜像..."
        docker compose pull new-api
        docker compose up -d new-api
        echo "✅ 更新完成"
        ;;
    backup)
        TIMESTAMP=$(date +%Y%m%d_%H%M%S)
        BACKUP_DIR="$DIR/backups"
        mkdir -p "$BACKUP_DIR"
        echo "📦 备份数据库..."
        docker compose exec -T postgres pg_dump -U gateway new-api > "$BACKUP_DIR/db_${TIMESTAMP}.sql"
        echo "📦 备份配置..."
        tar czf "$BACKUP_DIR/config_${TIMESTAMP}.tar.gz" .env docker-compose.yml nginx/
        echo "✅ 备份完成: $BACKUP_DIR/db_${TIMESTAMP}.sql"
        # 清理 7 天前的备份
        find "$BACKUP_DIR" -name "*.sql" -mtime +7 -delete
        find "$BACKUP_DIR" -name "*.tar.gz" -mtime +7 -delete
        ;;
    shell)
        SERVICE="${2:-new-api}"
        echo "🖥️  进入容器: $SERVICE"
        docker compose exec "$SERVICE" /bin/sh
        ;;
    *)
        echo "用法: $0 {start|stop|down|restart|status|logs [service]|update|backup|shell [service]}"
        echo ""
        echo "命令说明:"
        echo "  start          启动所有服务"
        echo "  stop           停止所有服务（保留容器）"
        echo "  down           停止并移除容器（保留数据）"
        echo "  restart        重启所有服务"
        echo "  status         查看运行状态"
        echo "  logs [service] 查看日志（默认: new-api，可选: nginx/postgres/redis）"
        echo "  update         更新 new-api 镜像"
        echo "  backup         备份数据库和配置"
        echo "  shell [service] 进入容器终端"
        ;;
esac

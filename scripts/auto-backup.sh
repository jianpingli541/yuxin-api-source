#!/bin/bash
set -euo pipefail
umask 077  # 所有备份默认 600（防密钥泄漏）
DIR=/root/projects/api-gateway
TS=$(date +%Y%m%d_%H%M%S)
DEST=$DIR/backups/auto
mkdir -p "$DEST"
cd "$DIR"
set -a; . ./.env; set +a
echo "[$TS] 备份开始"
docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" 2>/dev/null | gzip > "$DEST/pg_$TS.sql.gz"
echo "  PG: $(du -h "$DEST/pg_$TS.sql.gz" | cut -f1)"
docker compose exec -T redis redis-cli -a "$REDIS_PASSWORD" BGSAVE 2>/dev/null >/dev/null || true
sleep 3
docker cp gateway-redis:/data/dump.rdb "$DEST/redis_$TS.rdb" 2>/dev/null && echo "  Redis: $(du -h "$DEST/redis_$TS.rdb" | cut -f1)" || echo "  Redis: 跳过"
docker compose exec -T clickhouse clickhouse-client --user default --password "$CH_PASS" --query "BACKUP DATABASE new_api_logs TO Disk('backups','ch_$TS.zip')" 2>"/tmp/ch_backup_err_$TS.log" && echo "  ClickHouse: 完成" || { echo "  ClickHouse: 失败! 详见 /tmp/ch_backup_err_$TS.log"; cat "/tmp/ch_backup_err_$TS.log"; }
tar czf "$DEST/config_$TS.tar.gz" .env docker-compose.yml docker-compose.observability.yml nginx/ 2>/dev/null
echo "  Config: $(du -h "$DEST/config_$TS.tar.gz" | cut -f1)"
python3 - "$DEST" <<'PY'
import os, sys, time
dest = sys.argv[1]
cutoff = time.time() - 7*24*3600
removed = 0
for f in os.listdir(dest):
    p = os.path.join(dest, f)
    if os.path.isfile(p) and f != 'cron.log' and os.path.getmtime(p) < cutoff:
        os.remove(p); removed += 1
print(f"  清理: 删 {removed} 个旧备份")
PY
echo "[$TS] 完成,现存: $(ls "$DEST" | wc -l) 个文件"

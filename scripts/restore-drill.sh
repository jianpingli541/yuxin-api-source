#!/bin/bash
# 每月恢复演练: pg+CH 备份还原到 scratch 容器并断言行数 (R3 常规化)
set -uo pipefail
BASE=/root/projects/api-gateway/backups/auto
OUT=/root/projects/api-gateway/backups/auto/drill_$(date +%Y%m%d).log
{
echo "=== restore drill $(date -Is) ==="
PG_DUMP=$(ls -t $BASE/pg_*.sql.gz | head -1)
docker run -d --name drill-pg -e POSTGRES_PASSWORD=drill -v $BASE:/dumps:ro postgres:16-alpine >/dev/null 2>&1
sleep 8
docker exec drill-pg sh -c "gunzip -c /dumps/$(basename $PG_DUMP) | psql -U postgres -q 2>/dev/null >/dev/null"
U=$(docker exec drill-pg psql -U postgres -t -A -c "SELECT COUNT(*) FROM users;")
T=$(docker exec drill-pg psql -U postgres -t -A -c "SELECT COUNT(*) FROM tokens;")
TB=$(docker exec drill-pg psql -U postgres -t -A -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';")
docker rm -f drill-pg >/dev/null
echo "pg restore: tables=$TB users=$U tokens=$T (source: $(basename $PG_DUMP))"
[ "$TB" -gt 20 ] && [ "$U" -gt 0 ] && echo "PG_DRILL_PASS" || echo "PG_DRILL_FAIL"
CH_ZIP=$(ls -t $BASE/ch_*.zip 2>/dev/null | head -1)
if [ -n "$CH_ZIP" ]; then
  mkdir -p /tmp/drill-ch/backups && cp "$CH_ZIP" /tmp/drill-ch/backups/
  printf '<clickhouse><storage_configuration><disks><backups><type>local</type><path>/var/lib/clickhouse/backups/</path></backups></disks></storage_configuration><backups><allowed_disk>backups</allowed_disk></backups></clickhouse>' > /tmp/drill-ch/backup-disk.xml
  docker run -d --name drill-ch -v /tmp/drill-ch/backups:/var/lib/clickhouse/backups -v /tmp/drill-ch/backup-disk.xml:/etc/clickhouse-server/config.d/backup-disk.xml:ro clickhouse/clickhouse-server:24.8-alpine >/dev/null 2>&1
  sleep 12
  R=$(docker exec drill-ch clickhouse-client --query "RESTORE DATABASE new_api_logs FROM Disk('backups','$(basename $CH_ZIP)')" 2>&1 | tail -1)
  N=$(docker exec drill-ch clickhouse-client --query "SELECT count() FROM new_api_logs.logs" 2>/dev/null || echo 0)
  docker rm -f drill-ch >/dev/null; rm -rf /tmp/drill-ch
  echo "ch restore: logs rows=$N (source: $(basename $CH_ZIP))"
  [ "$N" -gt 0 ] && echo "CH_DRILL_PASS" || echo "CH_DRILL_FAIL"
fi
echo "=== drill done ==="
} >> "$OUT" 2>&1
grep -q DRILL_FAIL "$OUT" && exit 1 || exit 0

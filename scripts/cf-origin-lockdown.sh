#!/bin/bash
# 源站 80/443 仅允许 Cloudflare 官方 IP 段（R3 红队发现: 源站可直连绕过 CF）
set -e
CHAIN=CF-LOCKDOWN
iptables -N $CHAIN 2>/dev/null || true
iptables -F $CHAIN
iptables -A $CHAIN -s 127.0.0.0/8 -j RETURN
iptables -A $CHAIN -s 172.16.0.0/12 -j RETURN
for r in 173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/13 104.24.0.0/14 172.64.0.0/13 131.0.72.0/22; do
  iptables -A $CHAIN -s $r -p tcp -m multiport --dports 80,443 -j RETURN
done
iptables -A $CHAIN -p tcp -m multiport --dports 80,443 -j DROP
iptables -C DOCKER-USER -j $CHAIN 2>/dev/null || iptables -I DOCKER-USER 1 -j $CHAIN
echo "lockdown rules applied"

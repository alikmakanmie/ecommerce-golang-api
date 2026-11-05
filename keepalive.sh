#!/bin/bash
# Watchdog ecommerce-api + cloudflared.
# - API dicek tiap 30 dtk via /health, di-restart bila mati.
# - cloudflared dicek bila file .cloudflared-token (0600) sudah ada,
#   di-restart dengan token dari file itu (token TIDAK tertulis di script ini).
# Jalankan: nohup ./keepalive.sh > keepalive.out 2>&1 &
# Catatan: loop ini mati bila VM diganti; unit systemd adalah jalan yang
# tahan replacement (belum dibuat — minta ke Dhodi bila mau).
cd /home/hatch/workspace/ecommerce-api || exit 1
LOG=keepalive.log
log() { echo "$(date '+%F %T') $*" >> "$LOG"; }
while true; do
  if ! curl -sf -m 3 http://localhost:8080/health >/dev/null 2>&1; then
    pkill -x api 2>/dev/null || true
    sleep 1
    (setsid nohup ./bin/api > server.log 2>&1 < /dev/null &)
    log "api restart"
  fi
  if [ -f .cloudflared-token ]; then
    if ! pgrep -f "[c]loudflared tunnel --no-autoupdate --protocol http2 run" >/dev/null 2>&1; then
      (setsid nohup env TUNNEL_TOKEN="$(cat .cloudflared-token)" /home/hatch/workspace/tools/cloudflared tunnel --no-autoupdate --protocol http2 run > cf_tunnel.log 2>&1 < /dev/null &)
      log "cloudflared restart"
    fi
  fi
  sleep 30
done

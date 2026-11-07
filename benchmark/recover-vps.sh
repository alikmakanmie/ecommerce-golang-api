#!/bin/bash
# Pemulihan VPS pasca-benchmark 6 Nov 2026. Dijalankan DI VPS sebagai root.
# Urutan: matikan sisa benchmark -> cek service -> cleanup selektif
# (HANYA bench-*) -> restore DB dari backup -> swap -> hardening systemd
# -> deploy binary baru (concurrency limiter) -> verifikasi.
set -u
cd /tmp/benchrun 2>/dev/null || true

echo "== 1. Matikan sisa proses benchmark =="
pkill -x bench 2>/dev/null
pkill -f 'run-all.sh' 2>/dev/null
pkill -f 'monitor.sh' 2>/dev/null
sleep 2
ps aux | grep -E 'bench|run-all' | grep -v grep || echo "tidak ada sisa proses benchmark"

echo "== 2. Status service =="
systemctl is-active ecommerce-api ecommerce-web cloudflared
for s in ecommerce-api ecommerce-web cloudflared; do
  if [ "$(systemctl is-active $s)" != "active" ]; then
    echo "restart $s..."
    systemctl restart $s
  fi
done
sleep 3
systemctl is-active ecommerce-api ecommerce-web cloudflared

echo "== 3. Cleanup selektif uploads (HANYA bench-*) =="
echo "isi uploads sebelum cleanup:"
ls /opt/ecommerce-api/uploads/ | head -20
echo "jumlah bench-*: $(ls /opt/ecommerce-api/uploads/bench-* 2>/dev/null | wc -l)"
rm -f /opt/ecommerce-api/uploads/bench-*.jpg
echo "isi uploads sesudah cleanup:"
ls /opt/ecommerce-api/uploads/

echo "== 4. Restore DB dari backup pra-benchmark =="
systemctl stop ecommerce-api
rm -f /opt/ecommerce-api/ecommerce.db-wal /opt/ecommerce-api/ecommerce.db-shm
cp /opt/ecommerce-api/ecommerce.db.bak /opt/ecommerce-api/ecommerce.db
cp /opt/ecommerce-api/ecommerce.db-wal.bak /opt/ecommerce-api/ecommerce.db-wal
cp /opt/ecommerce-api/ecommerce.db-shm.bak /opt/ecommerce-api/ecommerce.db-shm
ls -la /opt/ecommerce-api/ecommerce.db*

echo "== 5. Kembalikan rate limit produksi di .env =="
sed -i 's/^PUBLIC_RATE_LIMIT=.*/PUBLIC_RATE_LIMIT=20/' /opt/ecommerce-api/.env
sed -i 's/^PUBLIC_BURST=.*/PUBLIC_BURST=40/' /opt/ecommerce-api/.env
grep -E "PUBLIC_RATE_LIMIT|PUBLIC_BURST" /opt/ecommerce-api/.env

echo "== 6. Swap 2GB bila belum ada =="
if [ "$(free -m | awk '/^Swap:/ {print $2}')" -eq 0 ]; then
  fallocate -l 2G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=2048 status=none
  chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
  grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
  echo "swap dibuat"
else
  echo "swap sudah ada"
fi
free -h | grep -E "Mem:|Swap:"

echo "== 7. Hardening systemd =="
mkdir -p /etc/systemd/system/cloudflared.service.d /etc/systemd/system/ecommerce-api.service.d
cat > /etc/systemd/system/cloudflared.service.d/override.conf <<'EOF'
[Service]
Nice=-10
OOMScoreAdjust=-500
Restart=always
RestartSec=2
EOF
cat > /etc/systemd/system/ecommerce-api.service.d/override.conf <<'EOF'
[Service]
MemoryMax=512M
Restart=always
RestartSec=2
EOF
systemctl daemon-reload
systemctl restart cloudflared ecommerce-api
sleep 3
systemctl show cloudflared -p Nice,OOMScoreAdjust,Restart,RestartSec | tr '\n' ' '; echo
systemctl show ecommerce-api -p MemoryMax,Restart,RestartSec,LimitNOFILE | tr '\n' ' '; echo

echo "== 8. Deploy binary baru (bila /opt/ecommerce-api/bin/api.new ada) =="
if [ -f /opt/ecommerce-api/bin/api.new ]; then
  systemctl stop ecommerce-api
  mv /opt/ecommerce-api/bin/api.new /opt/ecommerce-api/bin/api
  systemctl start ecommerce-api
  sleep 3
fi
systemctl is-active ecommerce-api ecommerce-web cloudflared
curl -s -m 5 http://127.0.0.1:8080/api/v1/health; echo
echo "RECOVER DONE"

#!/bin/bash
# Runner benchmark SLARAS NUSA FOODLAB di VPS (1 vCPU / 2GB).
# Menjalankan Skenario 1 (read API + web), 2 (guest checkout), 3 (upload)
# dengan monitor resource per tahap. Hasil JSON per tahap -> results.jsonl.
set -u
cd /tmp/benchrun
B=./bench
API=http://127.0.0.1:8080/api/v1
WEB=http://127.0.0.1:3000
JS_BUNDLE=$(ls /opt/ecommerce-web/dist/assets/index-*.js | head -1 | xargs basename)
OUT=results.jsonl
: > "$OUT"
: > run.log

start_mon() { ./monitor.sh "mon-$1.csv" & echo $! > mon.pid; sleep 1; }
stop_mon() { kill "$(cat mon.pid)" 2>/dev/null; sleep 1; }

echo "== Skenario 1a: API katalog (products + categories) ==" | tee -a run.log
for C in 100 500 1000 2500 5000; do
  start_mon "s1-api-$C"
  $B -mode read -url "$API/products,$API/categories" -c "$C" -d 30s >> "$OUT" 2>> run.log
  stop_mon
  echo "s1-api c=$C selesai" | tee -a run.log
done

echo "== Skenario 1b: Website (index + bundle JS) ==" | tee -a run.log
for C in 100 500 1000 2500 5000; do
  start_mon "s1-web-$C"
  $B -mode read -url "$WEB/,$WEB/assets/$JS_BUNDLE" -c "$C" -d 30s >> "$OUT" 2>> run.log
  stop_mon
  echo "s1-web c=$C selesai" | tee -a run.log
done

echo "== Top-up stok sebelum Skenario 2 ==" | tee -a run.log
TOKEN=$(curl -s -X POST "$API/auth/login" -H 'Content-Type: application/json' -d '{"email":"admin@ecommerce.local","password":"Admin123!"}' | node -e "let d='';process.stdin.on('data',c=>d+=c).on('end',()=>console.log(JSON.parse(d).data.access_token))")
node topup.js "$TOKEN" | tee -a run.log

echo "== Skenario 2: Concurrent guest checkout ==" | tee -a run.log
for C in 50 200 500; do
  start_mon "s2-$C"
  $B -mode checkout -url "$API" -c "$C" -d 30s >> "$OUT" 2>> run.log
  stop_mon
  echo "s2 c=$C selesai" | tee -a run.log
done

echo "== Skenario 3: Concurrent upload (700KB) ==" | tee -a run.log
for C in 10 50 100 250 500; do
  start_mon "s3-$C"
  $B -mode upload -url "$API" -file ./dummy-700k.jpg -c "$C" -d 30s >> "$OUT" 2>> run.log
  stop_mon
  echo "s3 c=$C selesai" | tee -a run.log
done

echo "ALL DONE" | tee -a run.log

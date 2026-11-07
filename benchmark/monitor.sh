#!/bin/bash
# Monitor resource VPS per detik -> CSV.
# Pemakaian: ./monitor.sh <outfile.csv> &  (kill untuk berhenti)
OUT="$1"
API_PID=$(pgrep -f '/opt/ecommerce-api/bin/api' | head -1)
echo "ts,cpu_pct,mem_used_mb,mem_total_mb,load1,api_rss_mb,api_cpu_pct,conns_8080,conns_3000" > "$OUT"
prev_total=0
prev_idle=0
while true; do
  read -r _ u n s i w irq si st _ _ < <(grep '^cpu ' /proc/stat)
  total=$((u + n + s + i + w + irq + si + st))
  if [ "$prev_total" -gt 0 ] && [ "$((total - prev_total))" -gt 0 ]; then
    cpu=$((100 * ((total - prev_total) - (i - prev_idle)) / (total - prev_total)))
  else
    cpu=0
  fi
  prev_total=$total
  prev_idle=$i
  read -r _ mem_total mem_used _ <<< "$(free -m | awk '/^Mem:/ {print $2, $3}')"
  load1=$(cut -d' ' -f1 /proc/loadavg)
  api_rss=$(ps -o rss= -p "$API_PID" 2>/dev/null | tr -d ' ')
  api_rss_mb=$((api_rss / 1024))
  api_cpu=$(ps -o %cpu= -p "$API_PID" 2>/dev/null | tr -d ' ')
  c8080=$(ss -tn state established 2>/dev/null | grep -c ':8080')
  c3000=$(ss -tn state established 2>/dev/null | grep -c ':3000')
  echo "$(date +%s),$cpu,$mem_used,$mem_total,$load1,$api_rss_mb,$api_cpu,$c8080,$c3000" >> "$OUT"
  sleep 1
done

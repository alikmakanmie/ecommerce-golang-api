#!/usr/bin/env bash
# Security audit simulation against http://localhost:8080
# Output is appended to security-test.log next to this script.
# Order matters: login-based tests run BEFORE the login rate-limit test.
set -u
BASE=http://localhost:8080
LOG="$(dirname "$0")/security-test.log"
SECRET=$(grep '^WEBHOOK_SECRET=' "$(dirname "$0")/.env" | cut -d= -f2)
JWT_SECRET=$(grep '^JWT_ACCESS_SECRET=' "$(dirname "$0")/.env" | cut -d= -f2)
: > "$LOG"
say() { echo "$*" | tee -a "$LOG"; }
code() { curl -s -o /tmp/sec_body.txt -w "%{http_code}" "$@"; }

say "=== S1: JWT bypass (fake + tampered + expired token) ==="
say "fake token cart: $(code $BASE/api/v1/cart -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAwIiwicm9sZSI6ImFkbWluIn0.fake')"
say "  body: $(cat /tmp/sec_body.txt)"
VALID=$(curl -s -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"customer@ecommerce.local","password":"Customer123!"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
TAMPERED="${VALID%?}x"
say "tampered token cart: $(code $BASE/api/v1/cart -H "Authorization: Bearer $TAMPERED")"
EXPIRED=$(python3 - "$JWT_SECRET" <<'PY'
import base64, hashlib, hmac, json, sys, time
def b64(b): return base64.urlsafe_b64encode(b).rstrip(b'=').decode()
h = b64(json.dumps({"alg":"HS256","typ":"JWT"}, separators=(',',':')).encode())
p = b64(json.dumps({"user_id":"50721186-ca72-4fbd-984f-2826dc2ae8f1","email":"customer@ecommerce.local","role":"customer","exp":int(time.time())-3600,"iat":int(time.time())-7200,"nbf":int(time.time())-7200}, separators=(',',':')).encode())
sig = b64(hmac.new(sys.argv[1].encode(), f"{h}.{p}".encode(), hashlib.sha256).digest())
print(f"{h}.{p}.{sig}")
PY
)
say "expired token cart: $(code $BASE/api/v1/cart -H "Authorization: Bearer $EXPIRED")"
say "no token cart: $(code $BASE/api/v1/cart)"

say "=== S2: role escalation (customer -> admin endpoint) ==="
CTOK=$(curl -s -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"customer@ecommerce.local","password":"Customer123!"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
ATOK=$(curl -s -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@ecommerce.local","password":"Admin123!"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
say "customer POST /products: $(code -X POST $BASE/api/v1/products -H "Authorization: Bearer $CTOK" -H 'Content-Type: application/json' -d '{"category_id":"00000000-0000-0000-0000-000000000000","name":"Hack","price":1,"stock":1,"sku":"HACK-1"}')"
PROD_ID=$(curl -s -X POST $BASE/api/v1/products -H "Authorization: Bearer $ATOK" -H 'Content-Type: application/json' -d "{\"category_id\":$(curl -s $BASE/api/v1/categories | python3 -c "import sys,json; print(repr(json.load(sys.stdin)['data'][0]['id']))"),\"name\":\"Produk Audit\",\"price\":1000,\"stock\":5,\"sku\":\"AUDIT-SEC-1\"}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['id'] if d.get('data') else d)")
say "admin POST /products: created id=$PROD_ID"
say "admin DELETE produk audit: $(code -X DELETE $BASE/api/v1/products/$PROD_ID -H "Authorization: Bearer $ATOK")"

say "=== S3: SQL injection pada ?search= ==="
for q in "' OR 1=1--" "%27%20OR%201%3D1--"; do
  curl -s "$BASE/api/v1/products?search=$q" -o /tmp/sec_body.txt -w "search=$q -> %{http_code} " >> "$LOG"
  python3 -c "import json; d=json.load(open('/tmp/sec_body.txt')); print('products returned:', len(d.get('data',[])), '| success:', d.get('success'))" | tee -a "$LOG"
done

say "=== S4: webhook signature ==="
# Order baru untuk uji webhook
ADDR=$(curl -s $BASE/api/v1/users/me/addresses -H "Authorization: Bearer $CTOK" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d[0]['id'] if d else '')")
if [ -z "$ADDR" ]; then
  ADDR=$(curl -s -X POST $BASE/api/v1/users/me/addresses -H "Authorization: Bearer $CTOK" -H 'Content-Type: application/json' -d '{"label":"Rumah","recipient_name":"Customer","phone":"081234567891","street":"Jl. Test No. 1","city":"Yogyakarta","province":"DIY","postal_code":"55165","is_default":true}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
fi
PROD=$(curl -s "$BASE/api/v1/products?limit=100" | python3 -c "import sys,json; ds=[p for p in json.load(sys.stdin)['data'] if p['stock']>=50]; print(ds[0]['id'])")
curl -s -X POST $BASE/api/v1/cart/items -H "Authorization: Bearer $CTOK" -H 'Content-Type: application/json' -d "{\"product_id\":\"$PROD\",\"quantity\":1}" -o /dev/null
ORDER_JSON=$(curl -s -X POST $BASE/api/v1/orders/checkout -H "Authorization: Bearer $CTOK" -H 'Content-Type: application/json' -d "{\"address_id\":\"$ADDR\",\"payment_method\":\"qris\"}")
TX=$(echo "$ORDER_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['payment']['transaction_id'])")
OID=$(echo "$ORDER_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
say "order uji: $OID tx=$TX"
BODY="{\"transaction_id\":\"$TX\",\"transaction_status\":\"settlement\"}"
say "webhook tanpa signature: $(code -X POST $BASE/api/v1/payments/webhook -H 'Content-Type: application/json' -d "$BODY")"
say "webhook signature salah: $(code -X POST $BASE/api/v1/payments/webhook -H 'Content-Type: application/json' -H 'X-Webhook-Signature: deadbeef' -d "$BODY")"
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)
say "webhook signature benar: $(code -X POST $BASE/api/v1/payments/webhook -H 'Content-Type: application/json' -H "X-Webhook-Signature: $SIG" -d "$BODY")"
say "order setelah webhook benar: $(curl -s $BASE/api/v1/orders/$OID -H "Authorization: Bearer $CTOK" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d['status'], '/', d['payment']['status'])")"

say "=== S5: security headers ==="
curl -s -I $BASE/api/v1/products | grep -iE "x-content-type-options|x-frame-options|x-xss-protection|referrer-policy|access-control-allow-origin" | tee -a "$LOG"
say "CORS origin tidak dikenal: $(curl -s -o /dev/null -w '%{http_code}' -H 'Origin: https://evil.example' $BASE/api/v1/products) (cek header ACAO di bawah)"
curl -s -I -H 'Origin: https://evil.example' $BASE/api/v1/products | grep -i "access-control-allow-origin" | tee -a "$LOG" || echo "(tidak ada header ACAO untuk origin asing = benar)" | tee -a "$LOG"

say "=== S6: rate limit login (7x password salah, IP sama, tanpa XFF) ==="
for i in 1 2 3 4 5 6 7; do
  say "percobaan $i: $(code -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"customer@ecommerce.local","password":"Salah123!"}')"
done
say "=== selesai ==="

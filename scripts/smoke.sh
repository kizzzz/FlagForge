#!/usr/bin/env bash
set -euo pipefail
command -v curl >/dev/null || { echo 'smoke requires curl' >&2; exit 1; }
command -v node >/dev/null || { echo 'smoke requires Node >=18' >&2; exit 1; }
node -e 'if (Number(process.versions.node.split(".")[0]) < 18) process.exit(1)' || {
  echo 'smoke requires Node >=18' >&2; exit 1;
}
base_url="${BASE_URL:-http://localhost:8787}"
base_url="${base_url%/}"
temp_dir="$(mktemp -d)"
trap 'rm -rf -- "$temp_dir"' EXIT
response="$temp_dir/response.json"
flag_key="smoke-$(node -e 'console.log(Date.now().toString(36)+"-"+require("node:crypto").randomBytes(4).toString("hex"))')"

request() {
  local method="$1" path="$2" expected="$3" body="${4:-}" status
  local args=(-sS --connect-timeout 5 --max-time 20 -X "$method" -o "$response" -w '%{http_code}')
  if [[ -n "$body" ]]; then args+=(-H 'Content-Type: application/json' --data "$body"); fi
  status="$(curl "${args[@]}" "$base_url$path")" || {
    echo "FAIL $method $path: connection/request failed" >&2; exit 1;
  }
  if [[ "$status" != "$expected" ]]; then
    echo "FAIL $method $path: expected HTTP $expected, got $status" >&2
    cat "$response" >&2
    exit 1
  fi
  echo "PASS $method $path HTTP $status"
}

assert_json() {
  node - "$response" "$1" <<'NODE'
const fs = require('node:fs');
try {
  const r = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
  const expression = process.argv[3];
  if (!new Function('r', `return (${expression})`)(r)) throw new Error(expression);
} catch (error) {
  console.error('FAIL JSON assertion:', error.message);
  process.exit(1);
}
NODE
}

request GET /healthz 200
assert_json 'r.status === "ok"'
flag_body="{\"name\":\"Smoke rollout\",\"key\":\"$flag_key\",\"environment\":\"staging\",\"enabled\":true,\"default_value\":false}"
request POST /api/v1/flags 201 "$flag_body"
assert_json 'typeof r.id === "string" && r.enabled === true && r.default_value === false'
flag_id="$(node -p 'JSON.parse(require("node:fs").readFileSync(process.argv[1],"utf8")).id' "$response")"
request POST /api/v1/flags 409 "$flag_body"
assert_json 'r.error.code === "DUPLICATE_FLAG_KEY"'
request POST "/api/v1/flags/$flag_id/rules" 201 '{"attribute":"plan","operator":"equals","expected_value":"pro","roll_value":true,"priority":1}'
assert_json 'r.priority === 1 && r.roll_value === true'
request POST /api/v1/evaluate 200 "{\"flag_key\":\"$flag_key\",\"environment\":\"staging\",\"user_attributes\":{\"plan\":\"pro\"}}"
assert_json 'r.value === true && r.reason.type === "rule_hit" && r.reason.priority === 1 && r.trace.length === 1'
request POST /api/v1/evaluate 200 "{\"flag_key\":\"$flag_key\",\"environment\":\"staging\",\"user_attributes\":{\"plan\":\"free\"}}"
assert_json 'r.value === false && r.reason.type === "default"'
request PATCH "/api/v1/flags/$flag_id/status" 200 '{"enabled":false}'
assert_json 'r.enabled === false'
request POST /api/v1/evaluate 200 "{\"flag_key\":\"$flag_key\",\"environment\":\"staging\",\"user_attributes\":{\"plan\":\"pro\"}}"
assert_json 'r.value === false && r.reason.type === "disabled" && r.trace.length === 1 && r.trace[0].cause === "short_circuited_by_disabled_flag"'
request GET "/api/v1/flags/$flag_id/history" 200
assert_json 'r.total >= 3 && r.items.length >= 3 && ["flag_created","rule_created","flag_disabled"].every(action => r.items.some(item => item.action === action))'
request POST /api/v1/evaluate/batch 200 "{\"flag_key\":\"$flag_key\",\"environment\":\"staging\",\"user_attributes_list\":[{\"plan\":\"pro\"},{}]}"
assert_json 'r.summary.total === 2 && r.summary.true_count === 0 && r.summary.false_count === 2 && r.summary.default_count === 0 && r.results.every(item => item.value === false && item.reason.type === "disabled")'
echo "SMOKE PASS; retained test flag $flag_key ($flag_id)"

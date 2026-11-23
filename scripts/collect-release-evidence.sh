#!/usr/bin/env bash
set -euo pipefail

checkout=$(cd "$1" && pwd)
harness=$(cd "$2" && pwd)
mkdir -p "$3"
output=$(cd "$3" && pwd)
variant=$4
revision=$(git -C "$checkout" rev-parse HEAD)
project="antifraud-release-${GITHUB_RUN_ID:?}-${GITHUB_RUN_ATTEMPT:?}-${variant}"

export GATEWAY_HOST_PORT=18080 POSTGRES_HOST_PORT=55434
export TEST_DATABASE_URL='postgres://demo:synthetic-local-only@127.0.0.1:55434/antifraud?sslmode=disable'
cd "$checkout"
if [[ ! -f compose.yaml || ! -d tests ]]; then
    echo 'This revision has no compatible Compose and integration test layout' > "$output/compose.log"
    exit 0
fi
compose() { docker compose -p "$project" "$@"; }
cleanup() { compose down --volumes >> "$output/compose.log" 2>&1; }
trap cleanup EXIT
cp compose.yaml "$output/compose.yaml"

# Older tags may not support the isolated ports. Keep their comparison unverified.
if ! compose config --format json | python -c '
import json, sys
services = json.load(sys.stdin)["services"]
for service, port in (("gateway", "18080"), ("postgres", "55434")):
    bindings = services[service]["ports"]
    assert [(p["host_ip"], str(p["published"])) for p in bindings] == [("127.0.0.1", port)]
' >> "$output/compose.log" 2>&1; then
    echo 'Compose fixture cannot use the isolated release ports' >> "$output/compose.log"
    exit 0
fi

cp "$harness/tests/evaluation_test.go" "$harness/tests/evaluation_burst_test.go" tests/
if compose up -d --wait postgres >> "$output/compose.log" 2>&1; then
    if [[ "$variant" == candidate ]]; then
        if make e2e > "$output/e2e.log" 2>&1; then
            printf '%s\n' PASS > "$output/e2e.status"
        else
            printf '%s\n' FAIL > "$output/e2e.status"
        fi
    fi
    python "$harness/scripts/evaluate.py" --output "$output/evaluation" > "$output/evaluation.log" 2>&1 || true
    if [[ -f "$output/evaluation/metrics.json" ]]; then
        python "$harness/scripts/evaluate.py" --burst --baseline "$output/evaluation/metrics.json" \
            --output "$output/evaluation-burst" > "$output/evaluation-burst.log" 2>&1 || true
    fi
fi
cleanup

# A new project volume for each repetition gives the same history at arrival zero.
for repeat in 1 2 3; do
    directory="$output/load/run-$repeat"
    mkdir -p "$directory"
    build=()
    if [[ "$repeat" == 1 ]]; then build=(--build); fi
    if compose up "${build[@]}" -d --wait >> "$directory/startup.log" 2>&1; then
        python "$harness/scripts/load.py" --profile "$harness/docs/load/gateway-profile.json" \
            --server-ref "$revision" --output "$directory" > "$directory/load.log" 2>&1 || true
    fi
    cleanup
done

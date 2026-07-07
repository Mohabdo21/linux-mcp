#!/usr/bin/env bash
set -euo pipefail

CMD="${1:-get_system_snapshot}"
FIFO=$(mktemp -u)
mkfifo "$FIFO"

(
	echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"demo","version":"1.0"}}}'
	sleep 0.5
	echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
	sleep 0.3
	echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"'"$CMD"'","arguments":{}}}'
	sleep 2
) >"$FIFO" &

timeout 8 ./bin/linux-mcp <"$FIFO" 2>/dev/null |
	jq -C 'select(.id == 2) | .result.content[0].text | fromjson | del(.host_id)' 2>/dev/null |
	head -60

rm -f "$FIFO"

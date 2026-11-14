#!/bin/sh
set -eu

cd /data

fifo=/tmp/services-exit
mkfifo "$fifo"

shutdown() {
  trap - TERM INT
  kill -TERM "$NGINX_PID" "$APP_PID" 2>/dev/null || true
  wait "$NGINX_PID" 2>/dev/null || true
  wait "$APP_PID" 2>/dev/null || true
  rm -f "$fifo"
}

trap 'shutdown; exit 0' TERM INT

# Start Go as an unprivileged user
(
  su-exec app /app serve
  echo app >"$fifo"
) &
APP_PID=$!

# Start Nginx
(
  nginx -g 'daemon off;'
  echo nginx >"$fifo"
) &
NGINX_PID=$!

# Exit if either process terminates
read -r exited_service <"$fifo" || true

echo "$exited_service exited, shutting down..."

shutdown
exit 1

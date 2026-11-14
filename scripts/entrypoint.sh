#!/bin/sh
set -eu

cd /data

# Start Go application as unprivileged user
su-exec app /app serve &
APP_PID=$!

# Start Nginx
nginx -g 'daemon off;' &
NGINX_PID=$!

shutdown() {
  trap - TERM INT

  kill -TERM "$NGINX_PID" "$APP_PID" 2>/dev/null || true

  wait "$NGINX_PID" 2>/dev/null || true
  wait "$APP_PID" 2>/dev/null || true
}

trap 'shutdown; exit 0' TERM INT

# Wait until either service exits
while kill -0 "$APP_PID" 2>/dev/null &&
  kill -0 "$NGINX_PID" 2>/dev/null; do
  sleep 1
done

echo "A service exited. Shutting down container."

shutdown
exit 1

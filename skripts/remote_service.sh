
#!/bin/bash
set -e

# ─── Config ───────────────────────────────────────────────────────────────────
SSH_HOST="tgn"
SERVICE_NAME="poker-bot"
XRAY_SERVICE_NAME="xray"
# ──────────────────────────────────────────────────────────────────────────────

SSH="ssh ${SSH_HOST}"
ACTION="${1}"

if [[ "$ACTION" != "start" && "$ACTION" != "stop" ]]; then
  echo "Usage: $0 start|stop"
  exit 1
fi

if [[ "$ACTION" == "start" ]]; then
  echo "==> Starting services..."
  $SSH "systemctl restart ${XRAY_SERVICE_NAME}"
  $SSH "systemctl restart ${SERVICE_NAME}"
else
  echo "==> Stopping services..."
  $SSH "systemctl stop ${SERVICE_NAME}"
  $SSH "systemctl stop ${XRAY_SERVICE_NAME}"
fi

echo "==> Status:"
$SSH "systemctl status ${SERVICE_NAME} --no-pager -l"

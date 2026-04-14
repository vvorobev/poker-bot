#!/bin/bash
set -e

# ─── Config ───────────────────────────────────────────────────────────────────
SSH_HOST="tgn"
REMOTE_DIR="/opt/poker-bot"
SERVICE_NAME="poker-bot"
BINARY_NAME="poker-bot"
# ──────────────────────────────────────────────────────────────────────────────

SSH="ssh ${SSH_HOST}"

echo "==> Uploading .env..."
scp .env.prod "${SSH_HOST}:${REMOTE_DIR}/.env"
$SSH "chmod 600 ${REMOTE_DIR}/.env"

echo "==> Restarting service..."
$SSH "systemctl restart ${SERVICE_NAME}"

echo "==> Status:"
$SSH "systemctl status ${SERVICE_NAME} --no-pager -l"

echo ""
echo "Done. Logs: ssh ${SSH_HOST} 'journalctl -fu ${SERVICE_NAME}'"


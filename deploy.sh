#!/bin/bash
set -e

# ─── Config ───────────────────────────────────────────────────────────────────
SSH_HOST="tgn"
REMOTE_DIR="/opt/poker-bot"
SERVICE_NAME="poker-bot"
BINARY_NAME="poker-bot"
# ──────────────────────────────────────────────────────────────────────────────

REMOTE_USER=$(ssh -G "$SSH_HOST" | awk '/^user / {print $2}')
SSH="ssh ${SSH_HOST}"
SCP="scp"

echo "==> Building linux/amd64 binary..."
GOOS=linux GOARCH=amd64 go build -o "${BINARY_NAME}-linux" ./cmd/bot/

echo "==> Ensuring remote dir exists..."
$SSH "mkdir -p ${REMOTE_DIR} && chown ${REMOTE_USER}:${REMOTE_USER} ${REMOTE_DIR}"

echo "==> Uploading binary..."
$SCP "${BINARY_NAME}-linux" "${SSH_HOST}:${REMOTE_DIR}/${BINARY_NAME}"
$SSH "chmod +x ${REMOTE_DIR}/${BINARY_NAME}"

echo "==> Uploading .env..."
$SCP .env.prod "${SSH_HOST}:${REMOTE_DIR}/.env"
$SSH "chmod 600 ${REMOTE_DIR}/.env"

# Upload DB only if it doesn't exist remotely yet (don't overwrite prod data)
if ! $SSH "test -f ${REMOTE_DIR}/poker.db" 2>/dev/null; then
    if [[ -f poker.db ]]; then
        echo "==> Uploading initial poker.db (first deploy)..."
        $SCP poker.db "${SSH_HOST}:${REMOTE_DIR}/poker.db"
    fi
else
    echo "==> Skipping poker.db (already exists on server, preserving prod data)"
fi

echo "==> Installing systemd service..."
$SSH "tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null" << EOF
[Unit]
Description=Poker Bot
After=network.target

[Service]
WorkingDirectory=${REMOTE_DIR}
ExecStart=${REMOTE_DIR}/${BINARY_NAME}
EnvironmentFile=${REMOTE_DIR}/.env
Restart=always
RestartSec=5
User=${REMOTE_USER}

[Install]
WantedBy=multi-user.target
EOF

echo "==> Restarting service..."
$SSH "systemctl daemon-reload && systemctl enable ${SERVICE_NAME} && systemctl restart ${SERVICE_NAME}"

echo "==> Status:"
$SSH "systemctl status ${SERVICE_NAME} --no-pager -l"

echo ""
echo "Done. Logs: ssh ${SSH_HOST} 'journalctl -fu ${SERVICE_NAME}'"

rm -f "${BINARY_NAME}-linux"

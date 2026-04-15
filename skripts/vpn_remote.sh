#!/usr/bin/env bash
# Install & configure Xray VLESS+Reality client on remote server via SSH
# Source: vles_config.md — Block 4g-My

set -euo pipefail

# ── SSH Config (same pattern as deploy.sh) ─────────────────────────────────
SSH_HOST="tgn"
SSH="ssh ${SSH_HOST}"

# ── VLESS+Reality Config ────────────────────────────────────────────────────
. ./.env.vpn

# ── Install Xray ────────────────────────────────────────────────────────────
echo "==> Installing Xray on ${SSH_HOST}..."
$SSH "bash -c \"\$(curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh)\" @ install"

# ── Write client config ─────────────────────────────────────────────────────
echo "==> Writing Xray config..."
$SSH "tee /usr/local/etc/xray/config.json > /dev/null" << EOF
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "tag": "socks",
      "port": ${SOCKS_PORT},
      "listen": "127.0.0.1",
      "protocol": "socks",
      "settings": {
        "auth": "noauth",
        "udp": true
      }
    },
    {
      "tag": "http",
      "port": 8118,
      "listen": "127.0.0.1",
      "protocol": "http"
    }
  ],
  "outbounds": [
    {
      "tag": "proxy",
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "${XRAY_ADDRESS}",
            "port": ${XRAY_PORT},
            "users": [
              {
                "id": "${XRAY_UUID}",
                "flow": "xtls-rprx-vision",
                "encryption": "none"
              }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "serverName": "${XRAY_SNI}",
          "fingerprint": "${XRAY_FINGERPRINT}",
          "shortId": "${XRAY_SHORT_ID}",
          "publicKey": "${XRAY_PUBLIC_KEY}",
          "spiderX": "/"
        }
      }
    },
    {
      "tag": "direct",
      "protocol": "freedom"
    }
  ],
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "type": "field",
        "ip": ["geoip:private"],
        "outboundTag": "direct"
      }
    ]
  }
}
EOF

# ── Enable & start service ───────────────────────────────────────────────────
echo "==> Starting Xray service..."
$SSH "systemctl daemon-reload && systemctl enable xray && systemctl restart xray"

# ── Smoke test ───────────────────────────────────────────────────────────────
echo "==> Testing connection..."
$SSH "curl -fsSL --socks5 127.0.0.1:${SOCKS_PORT} --max-time 10 https://t.me > /dev/null && echo '  OK: Telegram reachable' || echo '  WARN: Telegram not reachable — check: journalctl -u xray -f'"

echo "==> Status:"
$SSH "systemctl status xray --no-pager -l"

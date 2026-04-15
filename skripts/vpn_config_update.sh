#!/usr/bin/env bash
# Update Xray VLESS+Reality config on remote server and restart client
# Source: vless://...#Block 4g-poker-bot

set -euo pipefail

# ── SSH Config ──────────────────────────────────────────────────────────────
SSH_HOST="tgn"
SSH="ssh ${SSH_HOST}"

# ── VLESS+Reality Config (Block 4g-poker-bot) ───────────────────────────────
. ./.env.vpn

# ── Write client config ─────────────────────────────────────────────────────
echo "==> Updating Xray config on ${SSH_HOST}..."
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

# ── Restart service ──────────────────────────────────────────────────────────
echo "==> Restarting Xray..."
$SSH "systemctl restart xray"

# ── Smoke test ───────────────────────────────────────────────────────────────
echo "==> Testing connection..."
$SSH "curl -fsSL --socks5 127.0.0.1:${SOCKS_PORT} --max-time 10 https://t.me > /dev/null && echo '  OK: Telegram reachable' || echo '  WARN: Telegram not reachable — check: journalctl -u xray -f'"

echo "==> Status:"
$SSH "systemctl status xray --no-pager -l"

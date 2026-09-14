#!/bin/bash
set -euo pipefail

CERT_DIR="/etc/letsencrypt/live/dot.ramdns.my.id"
TLS_DIR="/etc/ramdns/tls"

install -o ubuntu -g ubuntu -m 0644 \
  "$CERT_DIR/fullchain.pem" \
  "$TLS_DIR/fullchain.pem"

install -o ubuntu -g ubuntu -m 0600 \
  "$CERT_DIR/privkey.pem" \
  "$TLS_DIR/privkey.pem"

systemctl restart ramdns

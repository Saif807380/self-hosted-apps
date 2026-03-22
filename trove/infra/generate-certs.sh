#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/certs"

mkdir -p "$CERT_DIR"

# Install the local CA into the system trust store (only needed once)
mkcert -install

# Generate certs for local dev + hotspot access
# 192.168.137.1 is the default Windows Mobile Hotspot gateway IP
mkcert \
  -cert-file "$CERT_DIR/cert.pem" \
  -key-file "$CERT_DIR/key.pem" \
  trove.local localhost 127.0.0.1 ::1 192.168.137.1

echo "Certificates generated in $CERT_DIR/"
echo ""
echo "To use from a phone on the hotspot:"
echo "  1. Copy $(mkcert -CAROOT)/rootCA.pem to your phone"
echo "  2. Install it in your phone's trusted certificate store"
echo "  3. Access https://192.168.137.1:3443"

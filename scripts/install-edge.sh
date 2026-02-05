#!/usr/bin/env sh
set -eu

BIN_NAME="nkudo-edge"
INSTALL_DIR="/usr/local/bin"
STATE_DIR="/var/lib/nkudo-edge/state"
PKI_DIR="/var/lib/nkudo-edge/pki"
RUNTIME_DIR="/var/lib/nkudo-edge/runtime"
UNIT_SRC_URL_DEFAULT="https://raw.githubusercontent.com/kubedoio/n-kudo/main/deployments/systemd/nkudo-edge.service"

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

if command -v curl >/dev/null 2>&1; then
  FETCH="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  FETCH="wget -qO-"
else
  echo "curl or wget is required" >&2
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) TARGET_ARCH="amd64" ;;
  aarch64|arm64) TARGET_ARCH="arm64" ;;
  *)
    echo "unsupported arch: $ARCH" >&2
    exit 1
    ;;
esac

VERSION="${NKUDO_EDGE_VERSION:-v0.1.0}"
BASE_URL="${NKUDO_EDGE_BASE_URL:-https://github.com/kubedoio/n-kudo/releases/download/${VERSION}}"
BIN_URL="${BASE_URL}/${BIN_NAME}-linux-${TARGET_ARCH}"
UNIT_URL="${NKUDO_EDGE_UNIT_URL:-$UNIT_SRC_URL_DEFAULT}"

mkdir -p "$INSTALL_DIR" "$STATE_DIR" "$PKI_DIR" "$RUNTIME_DIR" /etc/nkudo-edge
chmod 700 /var/lib/nkudo-edge "$STATE_DIR" "$PKI_DIR" "$RUNTIME_DIR"

echo "downloading ${BIN_URL}"
# shellcheck disable=SC2086
$FETCH "$BIN_URL" > "${INSTALL_DIR}/${BIN_NAME}"
chmod 755 "${INSTALL_DIR}/${BIN_NAME}"

if [ -n "${NKUDO_EDGE_SHA256:-}" ]; then
  echo "${NKUDO_EDGE_SHA256}  ${INSTALL_DIR}/${BIN_NAME}" | sha256sum -c -
fi

echo "installing systemd unit"
# shellcheck disable=SC2086
$FETCH "$UNIT_URL" > /etc/systemd/system/nkudo-edge.service

cat >/etc/nkudo-edge/nkudo-edge.env <<EOT
CONTROL_PLANE_URL=${CONTROL_PLANE_URL:-https://control-plane.example.com}
EOT
chmod 600 /etc/nkudo-edge/nkudo-edge.env

systemctl daemon-reload
systemctl enable nkudo-edge.service

echo "installed. next steps:"
echo "1) set CONTROL_PLANE_URL in /etc/nkudo-edge/nkudo-edge.env"
echo "2) enroll: NKUDO_ENROLL_TOKEN=<token> /usr/local/bin/nkudo-edge enroll --control-plane \"${CONTROL_PLANE_URL:-https://control-plane.example.com}\""
echo "3) start: systemctl restart nkudo-edge"

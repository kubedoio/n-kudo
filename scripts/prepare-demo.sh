#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

DEFAULT_OPS_DIR="/ops"
FALLBACK_OPS_DIR="${REPO_ROOT}/ops"
OPS_DIR="${OPS_DIR:-}"
ENV_FILE="${OPS_DIR}/nkudo-demo.env"
SOURCE_PLAN="${SOURCE_PLAN:-${REPO_ROOT}/examples/mvp1-demo-plan.json}"
PREPARED_PLAN="${OPS_DIR}/mvp1-demo-plan.json"

CH_VERSION="${CH_VERSION:-v43.0}"
UBUNTU_RELEASE="${UBUNTU_RELEASE:-noble}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

require_iso_builder() {
  if command -v cloud-localds >/dev/null 2>&1 || command -v genisoimage >/dev/null 2>&1 || command -v mkisofs >/dev/null 2>&1; then
    return
  fi

  echo "missing cloud-init ISO builder: install one of cloud-localds, genisoimage, or mkisofs" >&2
  if command -v apt-get >/dev/null 2>&1; then
    echo "hint (Debian/Ubuntu): sudo apt-get update && sudo apt-get install -y cloud-image-utils genisoimage" >&2
  elif command -v dnf >/dev/null 2>&1; then
    echo "hint (Fedora/RHEL): sudo dnf install -y cloud-utils-growpart genisoimage" >&2
  elif command -v yum >/dev/null 2>&1; then
    echo "hint (CentOS/RHEL): sudo yum install -y cloud-utils-growpart genisoimage" >&2
  elif command -v brew >/dev/null 2>&1; then
    echo "hint (macOS): brew install cloud-image-utils cdrtools" >&2
  fi
  exit 1
}

is_writable_dir() {
  local dir="$1"
  [[ -d "$dir" && -w "$dir" ]]
}

resolve_ops_dir() {
  if [[ -n "$OPS_DIR" ]]; then
    return
  fi

  if is_writable_dir "$DEFAULT_OPS_DIR"; then
    OPS_DIR="$DEFAULT_OPS_DIR"
    return
  fi

  if [[ ! -e "$DEFAULT_OPS_DIR" ]]; then
    if mkdir -p "$DEFAULT_OPS_DIR" 2>/dev/null; then
      chmod 0755 "$DEFAULT_OPS_DIR" || true
      OPS_DIR="$DEFAULT_OPS_DIR"
      return
    fi
  fi

  OPS_DIR="$FALLBACK_OPS_DIR"
}

download() {
  local url="$1"
  local out="$2"
  local tmp
  tmp="${out}.tmp"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --retry-delay 1 "$url" -o "$tmp"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$tmp" "$url"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi

  mv "$tmp" "$out"
}

main() {
  need jq
  require_iso_builder
  resolve_ops_dir
  ENV_FILE="${OPS_DIR}/nkudo-demo.env"
  PREPARED_PLAN="${OPS_DIR}/mvp1-demo-plan.json"

  if [[ ! -f "$SOURCE_PLAN" ]]; then
    echo "source plan not found: ${SOURCE_PLAN}" >&2
    exit 1
  fi

  local arch_raw arch cloud_hypervisor_asset ubuntu_image_name
  arch_raw="$(uname -m)"
  case "$arch_raw" in
    x86_64|amd64)
      arch="amd64"
      cloud_hypervisor_asset="cloud-hypervisor-static"
      ubuntu_image_name="${UBUNTU_RELEASE}-server-cloudimg-amd64.img"
      ;;
    aarch64|arm64)
      arch="arm64"
      cloud_hypervisor_asset="cloud-hypervisor-static-aarch64"
      ubuntu_image_name="${UBUNTU_RELEASE}-server-cloudimg-arm64.img"
      ;;
    *)
      echo "unsupported architecture: ${arch_raw}" >&2
      exit 1
      ;;
  esac

  local ch_bin_path rootfs_path
  ch_bin_path="${OPS_DIR}/cloud-hypervisor"
  rootfs_path="${OPS_DIR}/${ubuntu_image_name}"

  local ch_url rootfs_url
  ch_url="${CLOUD_HYPERVISOR_BIN_URL:-https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/${CH_VERSION}/${cloud_hypervisor_asset}}"
  rootfs_url="${ROOTFS_URL:-https://cloud-images.ubuntu.com/${UBUNTU_RELEASE}/current/${ubuntu_image_name}}"

  install -d -m 0755 "$OPS_DIR"
  if [[ ! -w "$OPS_DIR" ]]; then
    echo "ops directory is not writable: ${OPS_DIR}" >&2
    echo "set OPS_DIR to a writable location or run with elevated permissions" >&2
    exit 1
  fi

  echo "using ops dir: ${OPS_DIR}"

  if [[ ! -x "$ch_bin_path" ]]; then
    echo "downloading cloud-hypervisor from ${ch_url}"
    download "$ch_url" "$ch_bin_path"
    chmod 0755 "$ch_bin_path"
  else
    echo "using existing ${ch_bin_path}"
  fi

  if [[ ! -f "$rootfs_path" ]]; then
    echo "downloading ubuntu cloud image from ${rootfs_url}"
    download "$rootfs_url" "$rootfs_path"
    chmod 0644 "$rootfs_path"
  else
    echo "using existing ${rootfs_path}"
  fi

  jq \
    --arg rootfs "$rootfs_path" \
    '(.actions[] | select(.type == "MicroVMCreate") | .params.rootfs_path) = $rootfs' \
    "$SOURCE_PLAN" > "$PREPARED_PLAN"
  chmod 0644 "$PREPARED_PLAN"

  cat > "$ENV_FILE" <<EOF
export CLOUD_HYPERVISOR_BIN="${ch_bin_path}"
export NKUDO_DEMO_ROOTFS="${rootfs_path}"
export SAMPLE_PLAN="${PREPARED_PLAN}"
EOF
  chmod 0644 "$ENV_FILE"

  cat <<EOF
Prepared demo artifacts:
  cloud_hypervisor_bin=${ch_bin_path}
  rootfs_image=${rootfs_path}
  prepared_plan=${PREPARED_PLAN}
  env_file=${ENV_FILE}

Run:
  source "${ENV_FILE}"
  sudo -E ./demo.sh
EOF

  if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    # shellcheck disable=SC1090
    source "$ENV_FILE"
  fi
}

main "$@"

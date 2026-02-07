#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/deployments/docker-compose.yml}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-https://localhost:8443}"
ADMIN_KEY="${ADMIN_KEY:-dev-admin-key}"
SAMPLE_PLAN="${SAMPLE_PLAN:-${ROOT_DIR}/examples/mvp1-demo-plan.json}"
EDGE_BIN="${EDGE_BIN:-${ROOT_DIR}/bin/edge}"
CLOUD_HYPERVISOR_BIN="${CLOUD_HYPERVISOR_BIN:-cloud-hypervisor}"
WORK_BASE="${WORK_BASE:-${ROOT_DIR}/.demo/mvp1}"
HOSTNAME_OVERRIDE="${HOSTNAME_OVERRIDE:-$(hostname -s)}"

STATE_DIR="${WORK_BASE}/state"
PKI_DIR="${WORK_BASE}/pki"
RUNTIME_DIR="${WORK_BASE}/runtime"
WORK_DIR="${WORK_BASE}/work"
STATE_FILE="${STATE_DIR}/edge-state.json"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

say() {
  printf '\n[%s] %s\n' "$(date -u +%H:%M:%S)" "$*"
}

extract_json_field() {
  local json="$1"
  local field="$2"
  local value
  value="$(printf '%s' "$json" | jq -r "$field // empty")"
  if [[ -z "$value" ]]; then
    echo "failed to extract field $field from JSON:" >&2
    printf '%s\n' "$json" >&2
    exit 1
  fi
  printf '%s' "$value"
}

compose_up() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" up -d --build postgres backend
    return
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f "$COMPOSE_FILE" up -d --build postgres backend
    return
  fi
  echo "docker compose is required (docker compose or docker-compose)" >&2
  exit 1
}

wait_for_control_plane() {
  local retries=60
  local i
  for ((i=1; i<=retries; i++)); do
    if curl -skf "${CONTROL_PLANE_URL}/healthz" >/dev/null; then
      return 0
    fi
    sleep 2
  done
  echo "control-plane did not become ready at ${CONTROL_PLANE_URL}" >&2
  exit 1
}

ensure_edge_binary() {
  if [[ -x "$EDGE_BIN" ]]; then
    return
  fi
  say "Building edge binary at ${EDGE_BIN}"
  mkdir -p "$(dirname "$EDGE_BIN")"
  (cd "$ROOT_DIR" && go build -o "$EDGE_BIN" ./cmd/edge)
}

ensure_root_prereqs_for_microvm() {
  if [[ "$EUID" -ne 0 ]]; then
    cat >&2 <<'MSG'
This demo must run as root to create TAP interfaces and attach them to br0.
Re-run with sudo, for example:
  sudo -E ./demo.sh
MSG
    exit 1
  fi

  need ip
  if ! command -v cloud-localds >/dev/null 2>&1 && ! command -v genisoimage >/dev/null 2>&1 && ! command -v mkisofs >/dev/null 2>&1; then
    echo "need one of cloud-localds, genisoimage, or mkisofs in PATH" >&2
    exit 1
  fi
  if ! command -v "$CLOUD_HYPERVISOR_BIN" >/dev/null 2>&1; then
    echo "cloud-hypervisor binary not found: ${CLOUD_HYPERVISOR_BIN}" >&2
    exit 1
  fi
}

ensure_bridge_br0() {
  if ! ip link show br0 >/dev/null 2>&1; then
    say "Creating bridge br0"
    ip link add name br0 type bridge
  fi
  ip link set br0 up
}

write_edge_plan_create_start() {
  local vm_id="$1"
  local vm_name="$2"
  local tap_iface="$3"
  local out_file="$4"
  local execution_id="$5"

  jq \
    --arg vm_id "$vm_id" \
    --arg vm_name "$vm_name" \
    --arg tap "$tap_iface" \
    --arg execution_id "$execution_id" \
    '
    .execution_id = $execution_id |
    .actions |= map(
      if .type == "MicroVMCreate" then
        .action_id = ("create-" + $vm_id) |
        .params.vm_id = $vm_id |
        .params.name = $vm_name |
        .params.tap_iface = $tap
      elif .type == "MicroVMStart" then
        .action_id = ("start-" + $vm_id) |
        .params.vm_id = $vm_id
      else
        .
      end
    )
    ' "$SAMPLE_PLAN" > "$out_file"
}

write_edge_plan_stop_delete() {
  local vm_id="$1"
  local out_file="$2"
  local execution_id="$3"

  jq -n \
    --arg vm_id "$vm_id" \
    --arg execution_id "$execution_id" \
    '{
      execution_id: $execution_id,
      actions: [
        {
          action_id: ("stop-" + $vm_id),
          type: "MicroVMStop",
          timeout: 30,
          params: {vm_id: $vm_id}
        },
        {
          action_id: ("delete-" + $vm_id),
          type: "MicroVMDelete",
          timeout: 30,
          params: {vm_id: $vm_id}
        }
      ]
    }' > "$out_file"
}

write_cp_plan_payload_from_edge() {
  local edge_plan_file="$1"
  local idempotency_key="$2"
  local client_request_id="$3"
  local out_file="$4"

  jq -n \
    --arg idempotency_key "$idempotency_key" \
    --arg client_request_id "$client_request_id" \
    --slurpfile p "$edge_plan_file" '
    {
      idempotency_key: $idempotency_key,
      client_request_id: $client_request_id,
      actions: (
        $p[0].actions | map(
          if .type == "MicroVMCreate" then
            {
              operation_id: .action_id,
              operation: "CREATE",
              vm_id: .params.vm_id,
              name: (.params.name // .params.vm_id),
              vcpu_count: (.params.vcpu // 1),
              memory_mib: (.params.memory_mib // 256)
            }
          elif .type == "MicroVMStart" then
            {
              operation_id: .action_id,
              operation: "START",
              vm_id: .params.vm_id
            }
          elif .type == "MicroVMStop" then
            {
              operation_id: .action_id,
              operation: "STOP",
              vm_id: .params.vm_id
            }
          elif .type == "MicroVMDelete" then
            {
              operation_id: .action_id,
              operation: "DELETE",
              vm_id: .params.vm_id
            }
          else
            empty
          end
        )
      )
    }
    ' > "$out_file"
}

build_execution_updates() {
  local cp_response_file="$1"
  local edge_result_file="$2"

  jq -n \
    --slurpfile cp "$cp_response_file" \
    --slurpfile er "$edge_result_file" '
    ($cp[0].executions | map({(.operation_id): .id}) | add) as $exec_map
    | [
        $er[0].results[]
        | select($exec_map[.action_id] != null)
        | {
            execution_id: $exec_map[.action_id],
            state: (if .ok then "SUCCEEDED" else "FAILED" end),
            error_code: (if .ok then "" else (.error_code // "ACTION_FAILED") end),
            error_message: (if .ok then "" else (.message // "action failed") end)
          }
      ]
    '
}

build_log_entries_payload() {
  local cp_response_file="$1"
  local edge_result_file="$2"
  local agent_id="$3"

  jq -n \
    --arg agent_id "$agent_id" \
    --slurpfile cp "$cp_response_file" \
    --slurpfile er "$edge_result_file" '
    ($cp[0].executions | map({(.operation_id): .id}) | add) as $exec_map
    | {
        agent_id: $agent_id,
        entries: (
          $er[0].results
          | to_entries
          | map(
              select($exec_map[.value.action_id] != null)
              | {
                  execution_id: $exec_map[.value.action_id],
                  sequence: (.key + 1),
                  severity: (if .value.ok then "INFO" else "ERROR" end),
                  message: .value.message,
                  emitted_at: .value.finished_at
                }
            )
        )
      }
    '
}

hostfacts_to_flat_fields() {
  local hostfacts_file="$1"
  jq -n --slurpfile hf "$hostfacts_file" '
    {
      cpu_cores_total: ($hf[0].cpu_cores // 0),
      memory_bytes_total: ($hf[0].memory_total_bytes // 0),
      storage_bytes_total: (($hf[0].disks // []) | map(.total_bytes // 0) | add // 0),
      kvm_available: (($hf[0].kvm.present // false) and ($hf[0].kvm.readable // false) and ($hf[0].kvm.writable // false)),
      os: ($hf[0].os // "linux"),
      arch: ($hf[0].arch // "amd64"),
      kernel: ($hf[0].kernel // "")
    }
  '
}

state_to_microvms_payload() {
  local state_file="$1"
  if [[ ! -f "$state_file" ]]; then
    echo '[]'
    return
  fi

  jq -c '[
    .microvms[]? |
    {
      id: .id,
      name: .name,
      state: .status,
      vcpu_count: 1,
      memory_mib: 256,
      updated_at: .updated_at
    }
  ]' "$state_file"
}

post_heartbeat_with_updates() {
  local agent_id="$1"
  local hostname="$2"
  local hostfacts_file="$3"
  local microvms_json="$4"
  local updates_json="$5"
  local sequence="$6"

  local flat_json
  flat_json="$(hostfacts_to_flat_fields "$hostfacts_file")"

  local hb_payload
  hb_payload="$(jq -n \
    --arg agent_id "$agent_id" \
    --arg hostname "$hostname" \
    --argjson seq "$sequence" \
    --argjson flat "$flat_json" \
    --argjson microvms "$microvms_json" \
    --argjson updates "$updates_json" \
    --arg ch_available "$(if command -v "$CLOUD_HYPERVISOR_BIN" >/dev/null 2>&1; then echo true; else echo false; fi)" '
    {
      agent_id: $agent_id,
      heartbeat_seq: $seq,
      hostname: $hostname,
      agent_version: "demo-script",
      os: $flat.os,
      arch: $flat.arch,
      kernel_version: $flat.kernel,
      cpu_cores_total: $flat.cpu_cores_total,
      memory_bytes_total: $flat.memory_bytes_total,
      storage_bytes_total: $flat.storage_bytes_total,
      kvm_available: $flat.kvm_available,
      cloud_hypervisor_available: ($ch_available == "true"),
      microvms: $microvms,
      execution_updates: $updates
    }
  ')"

  curl -skf \
    --cert "${PKI_DIR}/client.crt" \
    --key "${PKI_DIR}/client.key" \
    -X POST "${CONTROL_PLANE_URL}/agents/heartbeat" \
    -H "Content-Type: application/json" \
    -d "$hb_payload" >/dev/null
}

submit_plan() {
  local site_id="$1"
  local api_key="$2"
  local payload_file="$3"
  local out_file="$4"

  curl -skf \
    -X POST "${CONTROL_PLANE_URL}/sites/${site_id}/plans" \
    -H "X-API-Key: ${api_key}" \
    -H "Content-Type: application/json" \
    --data-binary "@${payload_file}" > "$out_file"
}

run_edge_plan() {
  local plan_file="$1"
  local out_file="$2"
  "$EDGE_BIN" apply \
    --plan "$plan_file" \
    --state-dir "$STATE_DIR" \
    --runtime-dir "$RUNTIME_DIR" \
    --cloud-hypervisor-bin "$CLOUD_HYPERVISOR_BIN" \
    | tee "$out_file" >/dev/null
}

main() {
  need curl
  need jq
  need docker
  need go

  if [[ ! -f "$SAMPLE_PLAN" ]]; then
    echo "sample plan not found: ${SAMPLE_PLAN}" >&2
    exit 1
  fi

  ensure_root_prereqs_for_microvm

  mkdir -p "$STATE_DIR" "$PKI_DIR" "$RUNTIME_DIR" "$WORK_DIR"
  ensure_edge_binary

  say "Starting control-plane via docker compose"
  compose_up
  wait_for_control_plane

  say "Bootstrapping tenant/site"
  TENANT_SLUG="mvp1-demo-$(date -u +%Y%m%d%H%M%S)"
  TENANT_JSON="$(curl -skf -X POST "${CONTROL_PLANE_URL}/tenants" \
    -H "X-Admin-Key: ${ADMIN_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"slug\":\"${TENANT_SLUG}\",\"name\":\"MVP-1 Demo Tenant\",\"primary_region\":\"local\"}")"
  TENANT_ID="$(extract_json_field "$TENANT_JSON" '.id')"

  API_KEY_JSON="$(curl -skf -X POST "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/api-keys" \
    -H "X-Admin-Key: ${ADMIN_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"name":"mvp1-demo-key"}')"
  API_KEY="$(extract_json_field "$API_KEY_JSON" '.api_key')"

  SITE_JSON="$(curl -skf -X POST "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/sites" \
    -H "X-API-Key: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"name":"single-host-lab","location_country_code":"US"}')"
  SITE_ID="$(extract_json_field "$SITE_JSON" '.id')"

  TOKEN_JSON="$(curl -skf -X POST "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/enrollment-tokens" \
    -H "X-API-Key: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"site_id\":\"${SITE_ID}\",\"expires_in_seconds\":1800}")"
  ENROLL_TOKEN="$(extract_json_field "$TOKEN_JSON" '.token')"

  say "Enrolling edge agent"
  NKUDO_ENROLL_TOKEN="$ENROLL_TOKEN" "$EDGE_BIN" enroll \
    --control-plane "$CONTROL_PLANE_URL" \
    --state-dir "$STATE_DIR" \
    --pki-dir "$PKI_DIR" \
    --hostname "$HOSTNAME_OVERRIDE" \
    --insecure-skip-verify

  AGENT_ID="$(jq -r '.identity.agent_id // empty' "$STATE_FILE")"
  if [[ -z "$AGENT_ID" ]]; then
    echo "could not read agent_id from ${STATE_FILE}" >&2
    exit 1
  fi

  say "Sending initial heartbeat from n-kudo-edge (hostfacts included)"
  "$EDGE_BIN" verify-heartbeat \
    --control-plane "$CONTROL_PLANE_URL" \
    --state-dir "$STATE_DIR" \
    --pki-dir "$PKI_DIR" \
    --runtime-dir "$RUNTIME_DIR" \
    --netbird-enabled=false \
    --insecure-skip-verify \
    --cloud-hypervisor-bin "$CLOUD_HYPERVISOR_BIN"

  HOSTFACTS_JSON_FILE="${WORK_DIR}/hostfacts.json"
  "$EDGE_BIN" hostfacts > "$HOSTFACTS_JSON_FILE"

  RUN_ID="$(date -u +%Y%m%d%H%M%S)"
  RAND_HEX="$(openssl rand -hex 6)"
  VM_ID="00000000-0000-4000-8000-${RAND_HEX}"
  VM_NAME="mvp1-vm-${RUN_ID}"
  TAP_IFACE="nkd${RAND_HEX:0:8}"

  EDGE_CREATE_PLAN="${WORK_DIR}/edge-create-start.json"
  EDGE_CLEAN_PLAN="${WORK_DIR}/edge-stop-delete.json"
  CP_CREATE_PLAN="${WORK_DIR}/cp-create-start.json"
  CP_CLEAN_PLAN="${WORK_DIR}/cp-stop-delete.json"

  write_edge_plan_create_start "$VM_ID" "$VM_NAME" "$TAP_IFACE" "$EDGE_CREATE_PLAN" "exec-${RUN_ID}-create-start"
  write_edge_plan_stop_delete "$VM_ID" "$EDGE_CLEAN_PLAN" "exec-${RUN_ID}-stop-delete"

  write_cp_plan_payload_from_edge "$EDGE_CREATE_PLAN" "idem-${RUN_ID}-create" "req-${RUN_ID}-create" "$CP_CREATE_PLAN"
  write_cp_plan_payload_from_edge "$EDGE_CLEAN_PLAN" "idem-${RUN_ID}-clean" "req-${RUN_ID}-clean" "$CP_CLEAN_PLAN"

  ensure_bridge_br0

  say "Submitting CREATE/START plan to control-plane"
  CP_CREATE_RESP="${WORK_DIR}/cp-create-response.json"
  submit_plan "$SITE_ID" "$API_KEY" "$CP_CREATE_PLAN" "$CP_CREATE_RESP"
  cat "$CP_CREATE_RESP" | jq

  say "Executing CREATE/START locally with n-kudo-edge"
  EDGE_CREATE_RESULT="${WORK_DIR}/edge-create-result.json"
  run_edge_plan "$EDGE_CREATE_PLAN" "$EDGE_CREATE_RESULT"

  CREATE_UPDATES="$(build_execution_updates "$CP_CREATE_RESP" "$EDGE_CREATE_RESULT")"
  CREATE_LOGS_PAYLOAD="$(build_log_entries_payload "$CP_CREATE_RESP" "$EDGE_CREATE_RESULT" "$AGENT_ID")"
  MICROVMS_JSON="$(state_to_microvms_payload "$STATE_FILE")"

  say "Reporting execution updates and logs"
  post_heartbeat_with_updates "$AGENT_ID" "$HOSTNAME_OVERRIDE" "$HOSTFACTS_JSON_FILE" "$MICROVMS_JSON" "$CREATE_UPDATES" 2
  curl -skf \
    --cert "${PKI_DIR}/client.crt" \
    --key "${PKI_DIR}/client.key" \
    -X POST "${CONTROL_PLANE_URL}/agents/logs" \
    -H "Content-Type: application/json" \
    -d "$CREATE_LOGS_PAYLOAD" | jq

  say "Querying status, logs, and VM state"
  curl -skf "${CONTROL_PLANE_URL}/sites/${SITE_ID}/hosts" -H "X-API-Key: ${API_KEY}" | jq
  curl -skf "${CONTROL_PLANE_URL}/sites/${SITE_ID}/vms" -H "X-API-Key: ${API_KEY}" | jq

  while IFS= read -r exec_id; do
    if [[ -z "$exec_id" ]]; then
      continue
    fi
    echo "execution logs for ${exec_id}:"
    curl -skf "${CONTROL_PLANE_URL}/executions/${exec_id}/logs" -H "X-API-Key: ${API_KEY}" | jq
  done < <(jq -r '.executions[].id' "$CP_CREATE_RESP")

  VM_RUNTIME_DIR="${RUNTIME_DIR}/${VM_ID}"
  if [[ -f "${VM_RUNTIME_DIR}/state.json" ]]; then
    echo "local VM runtime state (${VM_RUNTIME_DIR}/state.json):"
    cat "${VM_RUNTIME_DIR}/state.json" | jq
  fi
  if [[ -f "${VM_RUNTIME_DIR}/commands.log" ]]; then
    echo "local VM command log (${VM_RUNTIME_DIR}/commands.log):"
    tail -n 20 "${VM_RUNTIME_DIR}/commands.log"
  fi

  say "Submitting STOP/DELETE plan to control-plane"
  CP_CLEAN_RESP="${WORK_DIR}/cp-clean-response.json"
  submit_plan "$SITE_ID" "$API_KEY" "$CP_CLEAN_PLAN" "$CP_CLEAN_RESP"
  cat "$CP_CLEAN_RESP" | jq

  say "Executing STOP/DELETE locally with n-kudo-edge"
  EDGE_CLEAN_RESULT="${WORK_DIR}/edge-clean-result.json"
  run_edge_plan "$EDGE_CLEAN_PLAN" "$EDGE_CLEAN_RESULT"

  CLEAN_UPDATES="$(build_execution_updates "$CP_CLEAN_RESP" "$EDGE_CLEAN_RESULT")"
  CLEAN_LOGS_PAYLOAD="$(build_log_entries_payload "$CP_CLEAN_RESP" "$EDGE_CLEAN_RESULT" "$AGENT_ID")"

  # Deletion removes local VM state; explicitly publish terminal VM state for the control-plane view.
  DELETE_MARKER="$(jq -n --arg vm_id "$VM_ID" --arg vm_name "$VM_NAME" --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '[{id:$vm_id,name:$vm_name,state:"DELETING",vcpu_count:1,memory_mib:256,updated_at:$ts}]')"

  say "Reporting cleanup execution updates and logs"
  post_heartbeat_with_updates "$AGENT_ID" "$HOSTNAME_OVERRIDE" "$HOSTFACTS_JSON_FILE" "$DELETE_MARKER" "$CLEAN_UPDATES" 3
  curl -skf \
    --cert "${PKI_DIR}/client.crt" \
    --key "${PKI_DIR}/client.key" \
    -X POST "${CONTROL_PLANE_URL}/agents/logs" \
    -H "Content-Type: application/json" \
    -d "$CLEAN_LOGS_PAYLOAD" | jq

  say "Final VM view"
  curl -skf "${CONTROL_PLANE_URL}/sites/${SITE_ID}/vms" -H "X-API-Key: ${API_KEY}" | jq

  if [[ -d "$VM_RUNTIME_DIR" ]]; then
    echo "warning: expected VM runtime dir to be deleted but found ${VM_RUNTIME_DIR}" >&2
  else
    echo "local runtime cleanup confirmed: ${VM_RUNTIME_DIR} removed"
  fi

  cat <<SUMMARY

Demo complete.

Artifacts:
  tenant_id=${TENANT_ID}
  site_id=${SITE_ID}
  agent_id=${AGENT_ID}
  vm_id=${VM_ID}
  work_dir=${WORK_BASE}

Control-plane is still running via docker compose.
Stop it with:
  docker compose -f ${COMPOSE_FILE} down
SUMMARY
}

main "$@"

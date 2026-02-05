# n-kudo-edge (MVP-1)

Single-binary edge agent for Debian/Ubuntu hosts.

Features in this implementation:
- enrollment with one-time token
- mTLS client credential generation + secure persistence
- heartbeat + plan fetch/report over mTLS HTTPS
- basic NetBird join/status verification
- host facts collection (CPU/RAM/disk/OS/kernel/arch/KVM/interfaces/bridges)
- Cloud Hypervisor microVM actions: create/start/stop/delete
- local idempotency cache in BoltDB
- best-effort log streaming per execution

Note: transport uses HTTPS + mTLS with typed JSON contracts to keep bootstrap/build minimal. The package layout is gRPC-ready and can be switched to generated protobuf stubs later.

## Repo tree

```text
.
├── cmd
│   └── nkudo-edge
│       └── main.go
├── deployments
│   └── systemd
│       └── nkudo-edge.service
├── examples
│   └── sample-plan.json
├── pkg
│   ├── controlplane
│   │   └── client.go
│   ├── enroll
│   │   ├── client.go
│   │   ├── token.go
│   │   └── token_test.go
│   ├── executor
│   │   ├── executor.go
│   │   ├── executor_test.go
│   │   └── types.go
│   ├── hostfacts
│   │   ├── hostfacts.go
│   │   └── hostfacts_test.go
│   ├── logstream
│   │   └── logstream.go
│   ├── mtls
│   │   └── mtls.go
│   ├── netbird
│   │   └── netbird.go
│   ├── providers
│   │   └── cloudhypervisor
│   │       └── provider.go
│   └── state
│       └── state.go
├── scripts
│   └── install.sh
├── tests
│   └── integration
│       └── fake_controlplane_test.go
└── go.mod
```

## Build

```bash
go mod tidy
go build -o nkudo-edge ./cmd/nkudo-edge
```

Static Linux binary:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/nkudo-edge-linux-amd64 ./cmd/nkudo-edge
```

## Install (curl | sh)

```bash
curl -fsSL https://raw.githubusercontent.com/n-kudo/n-kudo/main/scripts/install.sh | sudo CONTROL_PLANE_URL=https://cp.example.com sh
```

## Quickstart

### 1) Enroll

Token via env:

```bash
sudo NKUDO_ENROLL_TOKEN="<one-time-token>" \
  /usr/local/bin/nkudo-edge enroll \
  --control-plane https://cp.example.com
```

Token via file:

```bash
sudo /usr/local/bin/nkudo-edge enroll \
  --control-plane https://cp.example.com \
  --token-file /root/enroll.token
```

### 2) Run as service

```bash
sudo sed -i 's#^CONTROL_PLANE_URL=.*#CONTROL_PLANE_URL=https://cp.example.com#' /etc/nkudo-edge/nkudo-edge.env
sudo systemctl daemon-reload
sudo systemctl enable --now nkudo-edge
sudo systemctl status nkudo-edge --no-pager
```

### 3) Verify heartbeat

```bash
sudo /usr/local/bin/nkudo-edge verify-heartbeat \
  --control-plane https://cp.example.com
```

### 4) Create/start/stop/delete microVM (sample plan)

```bash
sudo /usr/local/bin/nkudo-edge apply --plan ./examples/sample-plan.json
```

## Tests

```bash
go test ./...
```

Integration test only:

```bash
go test ./tests/integration -run TestEnrollThenMutualTLSHeartbeatAndLogs -v
```

## Runtime directories
- certs/keys: `/var/lib/nkudo-edge/pki`
- state DB: `/var/lib/nkudo-edge/state/edge.db`
- VM runtime files: `/var/lib/nkudo-edge/runtime`

## Systemd unit
- file: `deployments/systemd/nkudo-edge.service`
- env file: `/etc/nkudo-edge/nkudo-edge.env`

# n-kudo

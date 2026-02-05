# NetBird MVP-1 Integration Strategy

## 1) Minimal strategy for MVP-1

### What runs where
- `netbird` client + daemon run on the customer Linux host where `nkudo-agent` runs.
- `nkudo-agent` calls local NetBird CLI (`netbird`) to:
  - validate install + daemon/service health
  - join peer (optional, setup-key flow)
  - read peer connectivity status
  - run a mesh endpoint probe (`ping` or `HTTP GET`)
- Control-plane receives normalized readiness state from agent:
  - `Connected`
  - `Degraded`
  - `NotConfigured`

### IDs/keys needed
- Required identity for status reporting:
  - `peer_id`
  - peer `ipv4` (NetBird IP)
  - optional `management_url`
  - optional `network_id`
- Join methods:
  - Setup key (non-interactive, recommended for automation): `netbird up --setup-key <key>`
  - Interactive login (not recommended for headless MVP-1 automation): `netbird up` + browser/device auth flow

### Approach A: customer already runs NetBird
- Agent config:
  - `enabled=true`
  - `auto_join=false`
  - no setup key in agent config
- Agent behavior:
  - validate CLI/service
  - read status
  - probe known mesh endpoint
  - report readiness

### Approach B: agent joins with setup key
- Agent config:
  - `enabled=true`
  - `auto_join=true`
  - setup key supplied at runtime (env/secret store)
  - optional install command if CLI missing
- Agent behavior:
  - install CLI if configured and missing
  - run `netbird up --setup-key ... --hostname ...`
  - verify status + probe endpoint
  - report readiness

## 2) Agent module implementation

Implemented in `internal/edge/netbird/netbird.go`:
- `Client.Evaluate(ctx, cfg)`:
  - detects netbird CLI
  - optionally installs CLI via configured command
  - checks netbird service/process
  - optionally joins with setup key
  - verifies peer status via `netbird status --json` (fallback plain status)
  - runs optional `http` or `ping` probe to mesh endpoint
  - returns `Snapshot` with normalized `Connected/Degraded/NotConfigured`
- `Client.DetectService(ctx)`:
  - checks `systemctl` service (`netbird` / `netbird.service`)
  - falls back to `pgrep` (`netbird`/`netbirdd`)
- `Snapshot.ControlPlaneConnected()`:
  - maps readiness to control-plane `connected` boolean
- `Snapshot.ToControlPlaneStatus()`:
  - returns a normalized payload (`state`, `connected`, `peer_id`, `ipv4`, `network_id`, `reason`) ready for heartbeat serialization

Related tests in `internal/edge/netbird/netbird_test.go` cover:
- JSON status parsing variants
- URL normalization for HTTP probe targets
- probe type normalization
- control-plane connected mapping

## 3) Security notes
- Setup key handling:
  - do not persist setup key to disk
  - inject via env var/secret manager at runtime only
  - clear config value immediately after join attempt in runtime flow
- Least retention:
  - store only peer metadata (`peer_id`, `ipv4`, connectivity state), not secrets
- Peer rotation/revoke:
  - rotate by issuing a new setup key in NetBird and re-running join
  - revoke compromised peers in NetBird management and force re-enroll
- Auditability:
  - log high-level outcomes only (joined/failed/connected), never log setup key value

## 4) Runbook (Debian/Ubuntu)

### Install NetBird client

Option 1 (quick install script):
```bash
curl -fsSL https://pkgs.netbird.io/install.sh | sudo sh
```

Option 2 (APT repository):
```bash
curl -fsSL https://pkgs.netbird.io/debian/public.key | sudo gpg --dearmor -o /usr/share/keyrings/netbird-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/netbird-archive-keyring.gpg] https://pkgs.netbird.io/debian stable main" | sudo tee /etc/apt/sources.list.d/netbird.list >/dev/null
sudo apt-get update
sudo apt-get install -y netbird
```

Start service:
```bash
sudo systemctl enable --now netbird
sudo systemctl status netbird --no-pager
```

### Join with setup key (automated)
```bash
sudo netbird up --setup-key "<NETBIRD_SETUP_KEY>" --hostname "site-a-edge-01"
```

Optional self-hosted management URL:
```bash
sudo netbird up --management-url "https://netbird.example.com" --setup-key "<NETBIRD_SETUP_KEY>" --hostname "site-a-edge-01"
```

### Verify connectivity
```bash
netbird status
netbird status --json
ping -c 1 100.90.0.10
curl -fsS http://100.90.0.10:8080/readyz
```

### Troubleshooting
- CLI missing:
```bash
which netbird
```
- Service not running:
```bash
sudo systemctl restart netbird
sudo journalctl -u netbird -n 200 --no-pager
```
- Peer disconnected:
```bash
netbird down
sudo netbird up --setup-key "<NETBIRD_SETUP_KEY>" --hostname "site-a-edge-01"
netbird status --json
```
- DNS/routing issues inside mesh:
```bash
ip route
ip addr show
netbird status
```

## Exact sample onboarding steps for customers

1. Install `netbird` on the host (runbook above).
2. Install/start `nkudo-agent`.
3. Choose onboarding mode:
   - A: customer-managed peer
     - connect peer manually with their own process
     - set agent config `auto_join=false`
   - B: agent-managed peer
     - create setup key in NetBird admin
     - export key as runtime secret for agent
     - set `auto_join=true` and `hostname`
4. Set mesh probe endpoint in agent config (`probe.target`).
5. Start/restart agent.
6. Confirm agent reports `Connected` when peer is connected and probe succeeds.

### Exact agent command examples (this repository)

Approach A (customer-managed NetBird; validation only):
```bash
go run ./cmd/edge run \
  --control-plane "https://cp.example.com" \
  --netbird-enabled=true \
  --netbird-auto-join=false \
  --netbird-probe-type=http \
  --netbird-probe-target="http://100.90.0.10:8080/readyz"
```

Approach B (agent-managed join with setup key):
```bash
export NETBIRD_SETUP_KEY="<NETBIRD_SETUP_KEY>"
go run ./cmd/edge run \
  --control-plane "https://cp.example.com" \
  --netbird-enabled=true \
  --netbird-auto-join=true \
  --netbird-setup-key "$NETBIRD_SETUP_KEY" \
  --netbird-hostname "site-a-edge-01" \
  --netbird-probe-type=http \
  --netbird-probe-target="http://100.90.0.10:8080/readyz"
```

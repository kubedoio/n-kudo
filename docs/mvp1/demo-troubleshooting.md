# MVP-1 Demo Troubleshooting

## Must run as root

Symptom:
- `This demo must run as root ...`

Fix:
- Run with `sudo -E ./demo.sh`.

## Control-plane does not become healthy

Symptom:
- `control-plane did not become ready at https://localhost:8443`

Checks:
- `docker compose -f deployments/docker-compose.yml ps`
- `docker compose -f deployments/docker-compose.yml logs backend`
- Ensure ports `8443` and `5432` are free.

## Enrollment fails with 400/401

Symptom:
- `enroll failed status=400` or `invalid or expired enrollment token`

Checks:
- The script creates one-time tokens; rerun script to issue a fresh token.
- Verify system clock is correct (expired tokens can fail immediately on skewed clocks).

## TLS errors (`certificate` / `x509`)

Symptom:
- Curl/edge TLS handshake errors.

Notes:
- Local demo intentionally uses self-signed control-plane certs.
- Script and edge commands use insecure TLS (`-k` and `--insecure-skip-verify`) for local-only testing.

## Cloud Hypervisor prerequisites missing

Symptom:
- `cloud-hypervisor binary not found`
- `cloud-init ISO builder not found (need cloud-localds or genisoimage)`

Fix:
- Install `cloud-hypervisor`.
- Install either `cloud-localds` (cloud-image-utils) or `genisoimage`.

## Network bridge/TAP errors

Symptom:
- `ip ... master br0` / `setup tap` failures.

Fix:
- Ensure `iproute2` is installed.
- Ensure the script can create and bring up `br0`.
- Check host policy does not block TAP creation.

## VM state not visible in control-plane

Symptom:
- `/sites/{site}/vms` does not show expected transitions.

Checks:
- Confirm heartbeat was posted with agent mTLS cert:
  - `${WORK_BASE:-.demo/mvp1}/pki/client.crt`
  - `${WORK_BASE:-.demo/mvp1}/pki/client.key`
- Re-run `./demo.sh` and inspect `${WORK_BASE:-.demo/mvp1}/work` JSON payloads.

## How to reset demo environment

```bash
sudo rm -rf .demo/mvp1
docker compose -f deployments/docker-compose.yml down -v
```

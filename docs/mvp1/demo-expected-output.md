# MVP-1 Demo Expected Output

Run:

```bash
sudo -E ./demo.sh
```

Typical output (IDs and timestamps will differ):

```text
[12:00:01] Starting control-plane via docker compose
[12:00:14] Bootstrapping tenant/site
[12:00:15] Enrolling edge agent
enrolled agent_id=... site_id=... host_id=...
pki written under .../.demo/mvp1/pki

[12:00:16] Sending initial heartbeat from n-kudo-edge (hostfacts included)

[12:00:18] Submitting CREATE/START plan to control-plane
{
  "plan_id": "...",
  "plan_version": 1,
  "plan_status": "PENDING",
  "deduplicated": false,
  "executions": [
    {"operation_type": "CREATE", "state": "PENDING", ...},
    {"operation_type": "START", "state": "PENDING", ...}
  ]
}

[12:00:18] Executing CREATE/START locally with n-kudo-edge
[12:00:22] Reporting execution updates and logs
{
  "accepted_frames": 2,
  "dropped_frames": 0
}

[12:00:23] Querying status, logs, and VM state
{ "hosts": [ ... ] }
{ "vms": [ { "state": "RUNNING", ... } ] }
execution logs for <execution-id>:
{ "logs": [ { "severity": "INFO", "message": "...", ... } ] }

[12:00:24] Submitting STOP/DELETE plan to control-plane
[12:00:24] Executing STOP/DELETE locally with n-kudo-edge
[12:00:26] Reporting cleanup execution updates and logs

[12:00:27] Final VM view
{ "vms": [ { "state": "DELETING", ... } ] }
local runtime cleanup confirmed: .../.demo/mvp1/runtime/<vm-id> removed

Demo complete.
Artifacts:
  tenant_id=...
  site_id=...
  agent_id=...
  vm_id=...
  work_dir=.../.demo/mvp1
```

# Cloud Hypervisor Provider (MVP-1)

## Approach
- MVP-1 provider uses host-installed binaries by default:
  - `cloud-hypervisor`
  - `ip`
  - `cloud-localds` (preferred) or `genisoimage`
- A static `cloud-hypervisor` binary is also supported by setting `Provider.Binary` to an absolute path.

## API Surface
- `CreateVM(spec) -> vm_id`
- `StartVM(vm_id)`
- `StopVM(vm_id)`
- `DeleteVM(vm_id)`
- `GetVMStatus(vm_id)`
- `CollectConsoleLog(vm_id)` (best effort; aggregates `console.log`, `stdout.log`, `stderr.log`)

## VM Spec Schema
```yaml
name: string
vcpu: int
mem_mb: int
disk_path: string
cloud_init_iso_path: string
tap_name: string
bridge_name: string
mac: string # optional
hostname: string # optional
ssh_authorized_keys: [] # optional
user_data: string # optional cloud-config override
disk_size_mb: int # optional, used when disk_path is empty
```

## Disk Handling
- Base image cache directory: `/var/lib/nkudo-edge/images`
- Per-VM runtime directory: `/var/lib/nkudo-edge/vms/<vm_id>`
- Per-VM disk path:
  - `/var/lib/nkudo-edge/vms/<vm_id>/disk.raw` by default
  - `/var/lib/nkudo-edge/vms/<vm_id>/disk.qcow2` if source image extension is `.qcow2`
- Behavior:
  - If `disk_path` points to an existing file, provider caches it in `/var/lib/nkudo-edge/images` and clones it into the VM directory.
  - If `disk_path` is empty, provider creates a sparse disk (`disk_size_mb`, default 10240 MB).

## Networking
- Tap creation and bridge attach:
```bash
ip tuntap add dev <tap_name> mode tap
ip link set <tap_name> master <bridge_name>
ip link set <tap_name> up
```
- Tap cleanup on delete:
```bash
ip link del <tap_name>
```

## Cloud-init Seed ISO
- Metadata generated under `/var/lib/nkudo-edge/vms/<vm_id>/seed/`:
  - `meta-data`
  - `user-data` (includes hostname + SSH authorized keys, unless custom `user_data` is provided)
- ISO build commands:
```bash
cloud-localds <cloud_init_iso_path> <user-data> <meta-data>
```
or fallback:
```bash
genisoimage -output <cloud_init_iso_path> -volid cidata -joliet -rock <user-data> <meta-data>
```

## Runtime Control and Process Tracking
- Start command:
```bash
cloud-hypervisor \
  --api-socket <vm_dir>/api.sock \
  --cpus boot=<vcpu> \
  --memory size=<mem_mb>M \
  --disk path=<vm_disk> \
  --disk path=<cloud_init_iso_path>,readonly=on \
  --serial file=<vm_dir>/console.log \
  --console off \
  --net tap=<tap_name>[,mac=<mac>]
```
- PID tracking:
  - PID saved in `/var/lib/nkudo-edge/vms/<vm_id>/ch.pid`
  - Provider metadata in `/var/lib/nkudo-edge/vms/<vm_id>/state.json`
- Stop behavior:
  1. Attempt API shutdown on unix socket:
     - `PUT /api/v1/vm.shutdown` via `<vm_dir>/api.sock`
  2. Fallback to `SIGTERM`
  3. Final fallback to `SIGKILL`

## Observability
- Command trace: `/var/lib/nkudo-edge/vms/<vm_id>/commands.log`
- Cloud Hypervisor logs:
  - `/var/lib/nkudo-edge/vms/<vm_id>/stdout.log`
  - `/var/lib/nkudo-edge/vms/<vm_id>/stderr.log`
  - `/var/lib/nkudo-edge/vms/<vm_id>/console.log`
- `CollectConsoleLog(vm_id)` returns concatenated contents of these files (best effort).

## Idempotent Delete
- `DeleteVM(vm_id)` is idempotent:
  - If VM metadata is already gone, it still removes `/var/lib/nkudo-edge/vms/<vm_id>` if present.
  - Missing tap interfaces are ignored during cleanup.
  - State DB entry deletion is best effort and treated as non-fatal for repeated deletes.

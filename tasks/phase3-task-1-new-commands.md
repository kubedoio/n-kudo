# Phase 3 Task 1: New Edge Agent Commands

## Task Description
Implement new CLI commands for the edge agent: status, check, unenroll, and renew.

## New Commands

### 1. `nkudo status`
**Purpose:** Show agent enrollment status, connection health, and certificate expiry

**Output:**
```
$ nkudo status
Agent Status: enrolled
Tenant ID:    550e8400-e29b-41d4-a716-446655440000
Site ID:      550e8400-e29b-41d4-a716-446655440001
Host ID:      550e8400-e29b-41d4-a716-446655440002
Agent ID:     550e8400-e29b-41d4-a716-446655440003

Certificate:
  Serial:     1234567890abcdef
  Expires:    2024-12-31 (89 days remaining)
  Valid:      yes

Connection:
  Control Plane: https://control-plane:8443
  Last Heartbeat: 2024-01-15 10:30:00 UTC (30s ago)
  Status:        connected
```

**Implementation:**
- Read state from `/var/lib/nkudo-edge/state/edge-state.json`
- Parse client certificate for expiry info
- Show enrollment details

### 2. `nkudo check`
**Purpose:** Pre-flight check for requirements

**Checks:**
- KVM availability (`/dev/kvm` accessible)
- Cloud Hypervisor binary exists and executable
- Network bridges configured (`br0` exists)
- Required directories writable
- NetBird installed (if enabled)
- Sufficient disk space
- Memory available

**Output:**
```
$ nkudo check
✓ KVM available
✓ Cloud Hypervisor binary found (/usr/bin/cloud-hypervisor)
✓ Bridge br0 exists
✓ State directory writable (/var/lib/nkudo-edge)
✓ PKI directory writable (/var/lib/nkudo-edge/pki)
✓ Runtime directory writable (/var/lib/nkudo-edge/vms)
✓ NetBird installed
✓ Disk space sufficient (100GB available)
✓ Memory sufficient (16GB total)

All checks passed! System is ready to run the edge agent.
```

**Exit codes:**
- 0: All checks passed
- 1: One or more checks failed

### 3. `nkudo unenroll`
**Purpose:** Cleanly remove agent from site

**Steps:**
1. Stop any running VMs (gracefully)
2. Send unenrollment request to control plane
3. Revoke certificate
4. Clear local state
5. Optionally remove directories

**Flags:**
- `--force` - Skip graceful VM shutdown
- `--keep-dirs` - Keep state directories (just clear contents)

**Output:**
```
$ nkudo unenroll
Stopping running VMs... done (2 VMs stopped)
Sending unenrollment request... done
Revoking certificate... done
Clearing local state... done

Agent successfully unenrolled from site.
```

### 4. `nkudo renew`
**Purpose:** Manual certificate renewal

**Process:**
1. Generate new CSR
2. Send to control plane with refresh token
3. Store new certificate
4. Verify new cert works with heartbeat

**Output:**
```
$ nkudo renew
Current certificate expires: 2024-12-31 (89 days remaining)
Generating new CSR... done
Requesting certificate renewal... done
New certificate received (expires: 2025-03-31)
Testing new certificate... done

Certificate successfully renewed!
```

## Files to Modify

### Main Command File
**`cmd/edge/main.go`**
- Add new subcommands to CLI
- Wire up command handlers

### New Files

**`internal/edge/cmd/status.go`**
```go
package cmd

func StatusCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show agent status",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
        },
    }
}
```

**`internal/edge/cmd/check.go`**
- Implement all check functions
- Return appropriate exit codes

**`internal/edge/cmd/unenroll.go`**
- Implement graceful VM shutdown
- Send unenrollment request
- Clean up state

**`internal/edge/cmd/renew.go`**
- Generate CSR
- Exchange refresh token for new cert
- Update state store

### API Client Extension

**`internal/edge/enroll/client.go`**
Add methods:
```go
func (c *Client) Unenroll(ctx context.Context, agentID string) error
func (c *Client) RenewCertificate(ctx context.Context, csrPEM string) (*RenewResponse, error)
```

## API Endpoints Needed (Backend)

### POST /v1/unenroll
Request:
```json
{
  "agent_id": "uuid",
  "reason": "string"
}
```

Response: 204 No Content

### POST /v1/renew
Request:
```json
{
  "agent_id": "uuid",
  "csr_pem": "string",
  "refresh_token": "string"
}
```

Response:
```json
{
  "client_certificate_pem": "string",
  "expires_at": "2025-03-31T00:00:00Z"
}
```

## Testing

Add tests in `cmd/edge/main_test.go`:
- Test status command output
- Test check command with all pass/fail scenarios
- Test unenroll flow
- Test certificate renewal

## Definition of Done
- [ ] All 4 commands implemented
- [ ] Commands have proper error handling
- [ ] Help text is descriptive
- [ ] Exit codes are correct
- [ ] Tests pass

## Estimated Effort
8-10 hours

# Secret Management

This document describes the secret management features available in n-kudo.

## Table of Contents

- [Overview](#overview)
- [Edge Agent - Encrypted Local State](#edge-agent---encrypted-local-state)
  - [Configuration](#configuration)
  - [Key Formats](#key-formats)
  - [Migration](#migration)
- [Control Plane - External Secret Store](#control-plane---external-secret-store)
  - [Environment Variables (Default)](#environment-variables-default)
  - [HashiCorp Vault](#hashicorp-vault)
  - [AWS Secrets Manager](#aws-secrets-manager)
- [Security Best Practices](#security-best-practices)

## Overview

n-kudo provides two main secret management features:

1. **Edge Agent**: Encrypted local state storage for protecting sensitive data at rest
2. **Control Plane**: Pluggable secret store integration for managing application secrets

## Edge Agent - Encrypted Local State

The edge agent can optionally encrypt its local state files using AES-256-GCM encryption. This protects sensitive data like identity tokens and credentials when stored on disk.

### Configuration

To enable encryption, set the `NKUDO_STATE_KEY` environment variable:

```bash
export NKUDO_STATE_KEY="your-32-byte-encryption-key-here"
nkudo-edge enroll --control-plane https://cp.example.com --token $TOKEN
nkudo-edge run --control-plane https://cp.example.com
```

When encryption is enabled:
- State is stored in `edge-state-encrypted.json`
- The file format is: version byte (1) + nonce (12 bytes) + ciphertext + tag (16 bytes)
- Backward compatibility: if no key is set, the agent uses unencrypted storage

### Key Formats

The encryption key can be provided in two formats:

#### 1. Raw 32-byte string

```bash
# Generate a random 32-byte key
export NKUDO_STATE_KEY=$(openssl rand -base64 32 | head -c 32)
```

#### 2. Base64-encoded key (recommended)

```bash
# Generate a base64-encoded key
export NKUDO_STATE_KEY=$(openssl rand -base64 32)
```

### Generating Keys

Use the following methods to generate secure keys:

```bash
# Using openssl (recommended)
export NKUDO_STATE_KEY=$(openssl rand -base64 32)

# Using Go (if you have Go installed)
go run -e 'package main; import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"
); func main() {
    key := make([]byte, 32)
    io.ReadFull(rand.Reader, key)
    fmt.Println(base64.StdEncoding.EncodeToString(key))
}'
```

### Migration

To migrate from unencrypted to encrypted storage:

1. Stop the edge agent
2. Set the `NKUDO_STATE_KEY` environment variable
3. Start the edge agent - it will create a new encrypted state file
4. Re-enroll the agent to populate the new state

For automated migration, use the `MigrateFromUnencrypted` function:

```go
key, _ := securestate.DeriveKey()
encryptedStore, _ := securestate.OpenWithKey("/var/lib/nkudo-edge/state", key)
securestate.MigrateFromUnencrypted(encryptedStore, "/var/lib/nkudo-edge/state/edge-state.json")
```

### Important Notes

- **Backup your key**: If you lose the encryption key, the state cannot be recovered
- **Key rotation**: To rotate keys, decrypt with the old key and re-encrypt with the new key
- **File permissions**: State files are created with 0600 permissions (readable only by owner)

## Control Plane - External Secret Store

The control plane supports pluggable secret stores for managing sensitive configuration like database credentials, admin keys, and SMTP passwords.

### Configuration

Set the `SECRET_STORE_TYPE` environment variable to choose the store:

```bash
export SECRET_STORE_TYPE=env    # Use environment variables (default)
export SECRET_STORE_TYPE=vault  # Use HashiCorp Vault
export SECRET_STORE_TYPE=aws    # Use AWS Secrets Manager
```

### Environment Variables (Default)

When using environment variables (default), secrets are read from:

- `DATABASE_URL` / `NKUDO_DATABASE_URL`
- `ADMIN_KEY` / `NKUDO_ADMIN_KEY`
- `SMTP_PASSWORD` / `NKUDO_SMTP_PASSWORD`

Example:

```bash
export SECRET_STORE_TYPE=env
export DATABASE_URL="postgres://user:pass@localhost/nkudo"
export ADMIN_KEY="your-secure-admin-key"
export SMTP_PASSWORD="your-smtp-password"
```

### HashiCorp Vault

To use HashiCorp Vault for secrets:

```bash
export SECRET_STORE_TYPE=vault
export VAULT_ADDR="https://vault.example.com:8200"
export VAULT_TOKEN="your-vault-token"
export VAULT_PATH="nkudo"  # Base path for secrets
```

#### Secret Structure

Secrets are stored in Vault's KV v2 store at the following paths:

```
secret/data/nkudo/database/url      -> Database URL
secret/data/nkudo/admin/key         -> Admin API key
secret/data/nkudo/smtp/password     -> SMTP password
secret/data/nkudo/ca/key            -> CA private key (if external)
```

Each secret should have a `value` field containing the secret string:

```bash
# Store database URL in Vault
vault kv put secret/nkudo/database/url value="postgres://user:pass@localhost/nkudo"

# Store admin key
vault kv put secret/nkudo/admin/key value="your-secure-admin-key"
```

#### Vault Policy

Create a Vault policy for n-kudo:

```hcl
# nkudo-policy.hcl
path "secret/data/nkudo/*" {
  capabilities = ["read"]
}

path "secret/metadata/nkudo/*" {
  capabilities = ["list"]
}
```

Apply the policy:

```bash
vault policy write nkudo nkudo-policy.hcl
vault token create -policy=nkudo
```

### AWS Secrets Manager

To use AWS Secrets Manager:

```bash
export SECRET_STORE_TYPE=aws
export AWS_REGION="us-east-1"
# AWS credentials via standard methods (IAM role, env vars, ~/.aws/credentials)
```

#### Secret Structure

Secrets should be created with the following names:

```
nkudo/database/url      -> Database URL
nkudo/admin/key         -> Admin API key
nkudo/smtp/password     -> SMTP password
nkudo/ca/key            -> CA private key (if external)
```

Create secrets using AWS CLI:

```bash
aws secretsmanager create-secret \
    --name nkudo/database/url \
    --secret-string "postgres://user:pass@localhost/nkudo"

aws secretsmanager create-secret \
    --name nkudo/admin/key \
    --secret-string "your-secure-admin-key"
```

Or create a JSON secret:

```bash
aws secretsmanager create-secret \
    --name nkudo/database/credentials \
    --secret-string '{"username":"nkudo","password":"secret123"}'
```

#### IAM Permissions

The control plane needs these IAM permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret"
            ],
            "Resource": "arn:aws:secretsmanager:*:*:secret:nkudo/*"
        }
    ]
}
```

## Security Best Practices

### Key Management

1. **Never commit keys to version control**
   - Use environment variables or secret stores
   - Rotate keys regularly

2. **Use strong encryption keys**
   - Generate keys using cryptographically secure random number generators
   - Use the full 32 bytes for AES-256

3. **Limit key access**
   - Use separate keys for different environments (dev, staging, prod)
   - Implement key rotation policies

4. **Backup and recovery**
   - Store backup keys in a secure location (e.g., separate Vault or password manager)
   - Test recovery procedures regularly

### Production Recommendations

1. **Edge Agent**:
   - Always enable state encryption in production
   - Use a key management service or hardware security module (HSM) for key storage
   - Consider using TPM or similar hardware-backed key derivation

2. **Control Plane**:
   - Use HashiCorp Vault or AWS Secrets Manager in production
   - Enable audit logging for all secret access
   - Use short-lived tokens/credentials where possible
   - Enable TLS for all Vault communications

3. **General**:
   - Regularly rotate all secrets (API keys, database passwords, etc.)
   - Monitor access logs for suspicious activity
   - Implement least-privilege access controls

### Example Production Setup

```bash
# Edge Agent - use systemd environment or environment file
# /etc/systemd/system/nkudo-edge.service.d/override.conf
[Service]
Environment="NKUDO_STATE_KEY_FILE=/etc/nkudo/state.key"
# Read key from file in wrapper script

# Control Plane with Vault
export SECRET_STORE_TYPE=vault
export VAULT_ADDR="https://vault.internal:8200"
export VAULT_TOKEN_FILE=/etc/nkudo/vault-token
export VAULT_PATH="production/nkudo"
```

## Troubleshooting

### Edge Agent

**Problem**: "encrypted state file exists but NKUDO_STATE_KEY is not set"

**Solution**: The agent previously ran with encryption enabled. Either:
- Set the `NKUDO_STATE_KEY` environment variable with the correct key
- Remove the encrypted state file (will require re-enrollment)

**Problem**: "invalid encryption key"

**Solution**: The key must be exactly 32 bytes (raw) or base64-encoded 32 bytes. Check:
```bash
# Check key length (should be 32)
echo -n "$NKUDO_STATE_KEY" | wc -c

# If using base64, verify it decodes to 32 bytes
echo "$NKUDO_STATE_KEY" | base64 -d | wc -c
```

### Control Plane

**Problem**: "vault error (status 403)"

**Solution**: The Vault token doesn't have permission to read secrets. Check:
- Token is valid and not expired
- Policy allows read access to `secret/data/nkudo/*`

**Problem**: "secret not found"

**Solution**: The secret doesn't exist in the store. Check:
- Secret path matches the expected format
- For Vault: `vault kv get secret/nkudo/admin/key`
- For AWS: `aws secretsmanager get-secret-value --secret-id nkudo/admin/key`

## API Reference

### Edge Agent (securestate package)

```go
// Open opens or creates a secure state store
func Open(dir string) (*Store, error)

// OpenWithKey opens with an explicit key
func OpenWithKey(dir string, key []byte) (*Store, error)

// DeriveKey derives a key from NKUDO_STATE_KEY
func DeriveKey() ([]byte, error)

// GenerateKey generates a random 32-byte key
func GenerateKey() ([]byte, error)

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(key, plaintext []byte) ([]byte, error)

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(key, ciphertext []byte) ([]byte, error)
```

### Control Plane (secrets package)

```go
// NewStoreFromEnv creates a store from environment configuration
func NewStoreFromEnv() (SecretStore, error)

// SecretStore interface
type SecretStore interface {
    Get(key string) (string, error)
    Set(key string, value string) error
    Delete(key string) error
}

// GetWithFallback retrieves with environment fallback
func GetWithFallback(store SecretStore, key, envKey, defaultValue string) string
```

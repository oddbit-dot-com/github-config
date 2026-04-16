# Vault-Backed GitHub Actions Secrets

**Date:** 2026-04-16
**Status:** Approved

## Overview

Add the ability to manage GitHub Actions secrets (repository, organization, and deployment environment) using values read from HashiCorp Vault KV v2, via the Pulumi Vault provider.

## New Types (`api/vault_provider.go`)

```go
type VaultProviderConfig struct {
    Address    *string
    Token      *string
    MountPoint string  // e.g. "secrets/"
}

// VaultSecretRef is a reference to a specific value in a Vault KV v2 secret.
type VaultSecretRef struct {
    Path string  // path within the KV v2 mount
    Key  string  // key within that secret
}

// OrgSecretRef is a vault secret reference for org-level GitHub Actions secrets.
type OrgSecretRef struct {
    VaultSecretRef
    Visibility string  // "all" | "private" | "selected", defaults to "all"
}

type ActionsSecrets    map[string]VaultSecretRef  // secret name → vault ref
type OrgActionsSecrets map[string]OrgSecretRef    // secret name → vault ref + visibility
```

### NewVaultProviderConfig

```go
func NewVaultProviderConfig(address, token string, mountpoint ...string) *VaultProviderConfig
```

Takes literal strings for `address` and `token` (callers use `os.Getenv(...)` directly at the call site for env-var-driven values). If either is empty, falls back to `VAULT_ADDR` / `VAULT_TOKEN` respectively. `mountpoint` is variadic and defaults to `"secret"` if not provided. Stores `nil` for address/token if still empty after env var resolution (vault provider plugin picks them up at runtime).

### CreateVaultProvider

```go
func CreateVaultProvider(ctx *pulumi.Context, config *VaultProviderConfig, orgName string) (*vault.Provider, error)
```

Creates a Pulumi Vault provider resource. Returns `nil, nil` if config is nil. Resource name: `vault-provider.{slugified-org}`.

## Organization Changes

New fields on `Organization`:

```go
VaultProviderConfig *VaultProviderConfig
Secrets             OrgActionsSecrets
vaultProvider       *vault.Provider  // private, cached
```

`Ensure()` gains two steps after `createProvider()`:
1. `createVaultProvider()` → stores in `o.vaultProvider`
2. `ensureOrgSecrets()` → provisions org-level secrets

`ensureOrgSecrets()`:
- Hard error if `o.Secrets != nil` and `o.VaultProviderConfig == nil`
- For each entry: calls `kv.LookupSecretV2` (mount = MountPoint stripped of leading `/`, path = `VaultSecretRef.Path`), extracts the key from `DataJson` via `ApplyT` + `json.Unmarshal`, creates `github.ActionsOrganizationSecret`
- Visibility defaults to `"all"` when `OrgSecretRef.Visibility` is empty
- Resource name: `github_actions_organization_secret.{org}.{secret_name}`

## Repository Changes

New fields on `Repository`:

```go
Secrets            ActionsSecrets
Environments       map[string]*github.RepositoryEnvironmentArgs
EnvironmentSecrets map[string]ActionsSecrets  // env name → secrets
```

`Ensure()` gains three steps after existing resources:
1. `ensureEnvironments()` → creates `github.RepositoryEnvironment` per entry; resource name: `github_repository_environment.{repo}.{env}`
2. `ensureRepoSecrets()` → hard error if `Secrets != nil` and org has no vault provider; creates `github.ActionsSecret` per entry; resource name: `github_actions_secret.{repo}.{secret_name}`
3. `ensureEnvironmentSecrets()` → hard error if any key in `EnvironmentSecrets` is absent from `Environments`; reads from vault and creates `github.ActionsEnvironmentSecret` per entry; resource name: `github_actions_environment_secret.{repo}.{env}.{secret_name}`

## Validation Rules

| Condition | Result |
|-----------|--------|
| `Secrets` set on org, `VaultProviderConfig` nil | Hard error |
| `Secrets` set on repo, org has no vault provider | Hard error |
| `Secrets` set on repo in standalone mode (no org) | Hard error |
| `EnvironmentSecrets` key not in `Environments` | Hard error |
| `EnvironmentSecrets` non-nil but `Environments` nil | Hard error (same rule) |

## Vault Data Source

Each secret value is read using `kv.LookupSecretV2`:
- Mount: `VaultProviderConfig.MountPoint` with leading `/` stripped
- Name: `VaultSecretRef.Path`
- Value extracted: `DataJson` → `json.Unmarshal` → `data[VaultSecretRef.Key]`

## Example Usage

```go
var Organization = api.Organization{
    Name:           "baystateradio",
    ProviderConfig: api.ProviderFromEnv("GITHUB_TOKEN_BAYSTATERADIO"),
    VaultProviderConfig: api.NewVaultProviderConfig(
        os.Getenv("VAULT_ADDR_BAYSTATERADIO"),
        os.Getenv("VAULT_TOKEN_BAYSTATERADIO"),
        "secrets/"),
    Secrets: api.OrgActionsSecrets{
        "DEPLOY_KEY": {VaultSecretRef: api.VaultSecretRef{Path: "github/baystateradio", Key: "deploy_key"}},
        "PRIVATE_SECRET": {
            VaultSecretRef: api.VaultSecretRef{Path: "github/baystateradio", Key: "private"},
            Visibility:     "private",
        },
    },
    Repositories: []*api.Repository{
        {
            Name:           "my-repo",
            RepositoryArgs: &github.RepositoryArgs{},
            Secrets: api.ActionsSecrets{
                "DB_PASSWORD": {Path: "github/my-repo", Key: "db_password"},
            },
            Environments: map[string]*github.RepositoryEnvironmentArgs{
                "production": {},
            },
            EnvironmentSecrets: map[string]api.ActionsSecrets{
                "production": {
                    "PROD_API_KEY": {Path: "github/my-repo/prod", Key: "api_key"},
                },
            },
        },
    },
}
```

## Files Changed

| File | Change |
|------|--------|
| `api/vault_provider.go` | New — types, `NewVaultProviderConfig`, `CreateVaultProvider` |
| `api/organization.go` | Add `VaultProviderConfig`, `Secrets`, `vaultProvider`; update `Ensure()` |
| `api/organization_secrets.go` | New — `ensureOrgSecrets()` |
| `api/repository.go` | Add `Secrets`, `Environments`, `EnvironmentSecrets`; update `Ensure()` |
| `api/repository_secrets.go` | New — `ensureRepoSecrets()`, `ensureEnvironments()`, `ensureEnvironmentSecrets()` |
| `api/secrets_test.go` | New — all secret-related tests |

## Testing

`mockMonitor` gains a `vaultSecrets map[string]map[string]string` field (path → key → value). `Call` handles the vault KV v2 invoke token and returns `dataJson`-encoded payloads.

| Test | Verifies |
|------|----------|
| `TestOrgSecretProvisioned` | `ActionsOrganizationSecret` created with correct name and value |
| `TestOrgSecretVisibilityDefault` | Empty `Visibility` defaults to `"all"` |
| `TestOrgSecretCustomVisibility` | Explicit visibility passed through |
| `TestOrgSecretNoVaultProvider` | Hard error when `Secrets` set but no `VaultProviderConfig` |
| `TestRepoSecretProvisioned` | `ActionsSecret` created correctly |
| `TestRepoSecretNoVaultProvider` | Hard error when repo has secrets but org has no vault provider |
| `TestRepoEnvironmentCreated` | `RepositoryEnvironment` created for each entry |
| `TestEnvSecretProvisioned` | `ActionsEnvironmentSecret` created with correct env and name |
| `TestEnvSecretMissingEnvironment` | Hard error when env secret references undeclared environment |

## Dependencies

Add `github.com/pulumi/pulumi-vault/sdk/v6` to `go.mod`.

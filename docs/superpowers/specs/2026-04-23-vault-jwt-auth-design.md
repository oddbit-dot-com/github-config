# Vault JWT Authentication Support

**Date:** 2026-04-23
**Status:** Approved

## Overview

Extend `VaultProviderConfig` to support JWT-based authentication alongside the existing token-based auth. This enables GitHub Actions workflows to authenticate to Vault using GitHub-minted OIDC tokens, while keeping local development working with `VAULT_TOKEN` — without any code change between environments.

## Motivation

The existing implementation authenticates to Vault using a static token (`VAULT_TOKEN`). In GitHub Actions, a more secure alternative is to use a short-lived JWT minted by GitHub's OIDC provider. Vault's JWT auth method validates these tokens against a configured role. The same codebase should work in both contexts by reading whichever credential is present in the environment at runtime.

## Prerequisite: restore `api/secrets_test.go`

The file `api/secrets_test.go` was added in commit `bd52a5c` and accidentally deleted in commit `026e38a` ("Rename Provider -> GithubProvider") — it was not updated to reflect the `Provider` → `GithubProvider` rename before being dropped. It must be restored and updated before new JWT tests are added.

## Changes

### `api/vault_provider.go`

#### Updated struct

```go
type VaultProviderConfig struct {
    Address    *string
    Token      *string
    JWT        *string  // explicit JWT override; normally read from VAULT_JWT env var
    JWTRole    *string  // Vault JWT auth role; when set, enables JWT auth path
    MountPoint string
}
```

#### New builder methods

```go
// WithJWTRole enables JWT auth with the given Vault role name.
// The JWT value is read from the VAULT_JWT env var at runtime,
// or from WithJWT if set explicitly.
func (c *VaultProviderConfig) WithJWTRole(role string) *VaultProviderConfig

// WithJWT sets an explicit JWT value, overriding the VAULT_JWT env var.
// Primarily useful in tests.
func (c *VaultProviderConfig) WithJWT(jwt string) *VaultProviderConfig
```

#### Updated `CreateVaultProvider` resolution logic

```
resolve jwt:   config.JWT (explicit) → os.Getenv("VAULT_JWT")
resolve token: config.Token (explicit) → os.Getenv("VAULT_TOKEN")

if JWTRole set AND jwt non-empty:
    use vault.ProviderArgs{AuthLoginJwt: &vault.ProviderAuthLoginJwtArgs{Jwt: jwt, Role: role}}
elif token non-empty:
    use vault.ProviderArgs{Token: token}
else:
    return error: "vault auth not configured for <org>: set VAULT_JWT or VAULT_TOKEN"
```

`JWTRole` being set signals intent to use JWT auth when a JWT is available. If no JWT is present (e.g. local dev), the provider falls back to token auth transparently. This allows one config to serve both environments.

#### Typical usage

```go
// Works in GitHub Actions (VAULT_JWT set) and locally (MY_ORG_TOKEN set),
// with no code change between environments:
api.NewVaultProviderConfig().
    WithJWTRole("my-github-role").
    WithToken(os.Getenv("MY_ORG_VAULT_TOKEN"))
```

```go
// For tests — explicit JWT value, no env var dependency:
api.NewVaultProviderConfig().
    WithJWTRole("my-role").
    WithJWT("test-jwt-value")
```

#### Auth method resolution precedence

| JWTRole set | JWT resolved | Token resolved | Result     |
|-------------|--------------|----------------|------------|
| yes         | yes          | any            | JWT auth   |
| yes         | no           | yes            | Token auth |
| yes         | no           | no             | Error      |
| no          | any          | yes            | Token auth |
| no          | any          | no             | Error      |

JWT value resolution order: `config.JWT` (explicit) → `VAULT_JWT` env var.
Token value resolution order: `config.Token` (explicit) → `VAULT_TOKEN` env var.

### `api/secrets_test.go`

Restore from commit `bd52a5c`, update all references from `Provider` to `GithubProvider`, then add the following new tests:

| Test | Verifies |
|------|---------|
| `TestJWTAuthUsed` | When JWTRole and JWT are configured, `AuthLoginJwt` is passed to the provider |
| `TestJWTFallsBackToToken` | When JWTRole is set but no JWT is available, falls back to token auth |
| `TestJWTFromEnvVar` | JWT value is read from `VAULT_JWT` env var when not set explicitly |
| `TestJWTIgnoredWithoutRole` | `VAULT_JWT` set but no JWTRole configured → token auth used, JWT ignored |
| `TestNoCredentials` | Neither JWT nor token available → hard error |

The existing `mockMonitor` handles resource registration rather than HTTP calls, so JWT auth args are passed through as provider inputs and can be asserted directly without mocking a Vault HTTP exchange.

## Vault-side setup (out of scope)

For reference, the Vault/OpenBao side requires:
- A JWT auth method enabled (e.g. `vault auth enable jwt`)
- A role configured with `bound_audiences` matching the JWT's `aud` claim and appropriate policies
- The JWT audience is embedded in the token at mint time; Vault validates it server-side

## Files Changed

| File | Change |
|------|--------|
| `api/vault_provider.go` | Add `JWT`, `JWTRole` fields; add `WithJWT`, `WithJWTRole` builders; update `CreateVaultProvider` |
| `api/secrets_test.go` | Restore from `bd52a5c`, fix `Provider` → `GithubProvider` rename, add four JWT auth tests |

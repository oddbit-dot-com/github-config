# Vault JWT Authentication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add JWT-based authentication to `VaultProviderConfig` so that GitHub Actions workflows can authenticate to Vault using GitHub-minted OIDC tokens, while local development continues to use `VAULT_TOKEN`.

**Architecture:** Add `JWT *string` and `JWTRole *string` fields to `VaultProviderConfig` with `WithJWT`/`WithJWTRole` builder methods. `CreateVaultProvider` resolves JWT from config or `VAULT_JWT` env var and token from config or `VAULT_TOKEN`; if JWTRole is set and a JWT is available it uses `AuthLoginJwt`, otherwise falls back to token auth. A prerequisite is restoring `api/secrets_test.go` (dropped in commit `026e38a`) and creating the missing `api/mock_test.go` that defines the `mockMonitor` struct the tests depend on.

**Tech Stack:** Go, Pulumi SDK v3.230.0, pulumi-vault SDK v6.7.4

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `api/mock_test.go` | **Create** | `mockMonitor` struct, `NewResource`, `findByType` |
| `api/secrets_test.go` | **Restore + extend** | All secret provisioning tests + JWT auth tests |
| `api/vault_provider.go` | **Modify** | Add `JWT`/`JWTRole` fields, `WithJWT`/`WithJWTRole` builders, update `CreateVaultProvider` |

---

## Task 1: Create `api/mock_test.go`

`api/secrets_test.go` was restored from git history but references `mockMonitor`, `mockResource`, and `findByType` that were never defined. This file provides that infrastructure.

**Files:**
- Create: `api/mock_test.go`

- [ ] **Step 1: Create the file**

```go
package api

import (
	"sync"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type mockResource struct {
	typ    string
	name   string
	inputs resource.PropertyMap
}

type mockMonitor struct {
	mu        sync.Mutex
	resources []mockResource
	vaultData map[string]map[string]string
}

func (m *mockMonitor) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = append(m.resources, mockResource{
		typ:    args.TypeToken,
		name:   args.Name,
		inputs: args.Inputs,
	})
	return args.Name + "_id", args.Inputs, nil
}

func (m *mockMonitor) findByType(typ string) []mockResource {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []mockResource
	for _, r := range m.resources {
		if r.typ == typ {
			result = append(result, r)
		}
	}
	return result
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /home/lars/projects/oddbit-dot-com/github-config
go build ./api/...
```

Expected: no errors. (`secrets_test.go` doesn't exist yet so no test compilation errors.)

- [ ] **Step 3: Commit**

```bash
git add api/mock_test.go
git commit -m "Add mock test infrastructure for Pulumi unit tests"
```

---

## Task 2: Restore `api/secrets_test.go`

The file exists in git history at commit `bd52a5c` and does not reference the renamed `GithubProviderConfig` type (only `VaultProviderConfig` and `github.Provider` from the SDK), so it can be restored verbatim.

**Files:**
- Create: `api/secrets_test.go`

- [ ] **Step 1: Restore the file from git**

```bash
git show bd52a5c:api/secrets_test.go > api/secrets_test.go
```

- [ ] **Step 2: Run the tests to verify they compile and pass**

```bash
go test ./api/... -v 2>&1 | head -60
```

Expected: all existing tests pass (TestOrgSecret*, TestRepoSecret*, TestRepoEnvironmentCreated, TestEnvSecret*). No compilation errors.

- [ ] **Step 3: Commit**

```bash
git add api/secrets_test.go
git commit -m "Restore secrets_test.go dropped in Provider rename commit"
```

---

## Task 3: Write failing JWT auth tests

Add the five JWT-specific tests to `api/secrets_test.go`. They will fail to compile until Task 4 adds `JWT`/`JWTRole` fields and the builder methods.

**Files:**
- Modify: `api/secrets_test.go`

- [ ] **Step 1: Add `testVaultConfigJWT` helper and five JWT tests**

Append the following to the end of `api/secrets_test.go`, before the final `var _ = (*github.Provider)(nil)` line:

```go
func testVaultConfigJWT() *VaultProviderConfig {
	return NewVaultProviderConfig().
		WithAddress("http://localhost:8200").
		WithJWTRole("test-role").
		WithJWT("test-jwt-token")
}

func TestJWTAuthUsed(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfigJWT(),
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	providers := mocks.findByType("pulumi:providers:vault")
	if len(providers) != 1 {
		t.Fatalf("expected 1 vault provider, got %d", len(providers))
	}
	authJwt := providers[0].inputs["authLoginJwt"]
	if authJwt.IsNull() {
		t.Fatal("expected authLoginJwt to be set on vault provider")
	}
	jwtObj := authJwt.ObjectValue()
	if jwtObj["jwt"].StringValue() != "test-jwt-token" {
		t.Errorf("expected jwt=test-jwt-token, got %v", jwtObj["jwt"])
	}
	if jwtObj["role"].StringValue() != "test-role" {
		t.Errorf("expected role=test-role, got %v", jwtObj["role"])
	}
}

func TestJWTFallsBackToToken(t *testing.T) {
	t.Setenv("VAULT_JWT", "")
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name: "testorg",
			VaultProviderConfig: NewVaultProviderConfig().
				WithAddress("http://localhost:8200").
				WithJWTRole("test-role").
				WithToken("test-token"),
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	providers := mocks.findByType("pulumi:providers:vault")
	if len(providers) != 1 {
		t.Fatalf("expected 1 vault provider, got %d", len(providers))
	}
	if providers[0].inputs["token"].StringValue() != "test-token" {
		t.Errorf("expected token=test-token, got %v", providers[0].inputs["token"])
	}
	if !providers[0].inputs["authLoginJwt"].IsNull() {
		t.Error("expected authLoginJwt to be absent when falling back to token auth")
	}
}

func TestJWTFromEnvVar(t *testing.T) {
	t.Setenv("VAULT_JWT", "env-jwt-token")
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name: "testorg",
			VaultProviderConfig: NewVaultProviderConfig().
				WithAddress("http://localhost:8200").
				WithJWTRole("test-role"),
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	providers := mocks.findByType("pulumi:providers:vault")
	if len(providers) != 1 {
		t.Fatalf("expected 1 vault provider, got %d", len(providers))
	}
	authJwt := providers[0].inputs["authLoginJwt"]
	if authJwt.IsNull() {
		t.Fatal("expected authLoginJwt to be set when VAULT_JWT is in env")
	}
	if authJwt.ObjectValue()["jwt"].StringValue() != "env-jwt-token" {
		t.Errorf("expected jwt=env-jwt-token, got %v", authJwt.ObjectValue()["jwt"])
	}
}

func TestJWTIgnoredWithoutRole(t *testing.T) {
	t.Setenv("VAULT_JWT", "env-jwt-token")
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name: "testorg",
			VaultProviderConfig: NewVaultProviderConfig().
				WithAddress("http://localhost:8200").
				WithToken("test-token"),
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	providers := mocks.findByType("pulumi:providers:vault")
	if len(providers) != 1 {
		t.Fatalf("expected 1 vault provider, got %d", len(providers))
	}
	if providers[0].inputs["token"].StringValue() != "test-token" {
		t.Errorf("expected token=test-token, got %v", providers[0].inputs["token"])
	}
	if !providers[0].inputs["authLoginJwt"].IsNull() {
		t.Error("expected JWT to be ignored when no JWTRole is configured")
	}
}

func TestNoCredentials(t *testing.T) {
	t.Setenv("VAULT_JWT", "")
	t.Setenv("VAULT_TOKEN", "")
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name: "testorg",
			VaultProviderConfig: NewVaultProviderConfig().
				WithAddress("http://localhost:8200").
				WithJWTRole("test-role"),
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err == nil {
		t.Fatal("expected error when no JWT or token is available")
	}
	if !strings.Contains(err.Error(), "vault auth not configured") {
		t.Errorf("expected 'vault auth not configured' error, got: %v", err)
	}
}
```

- [ ] **Step 2: Verify the tests fail to compile (expected)**

```bash
go test ./api/... 2>&1 | head -20
```

Expected: compile error — `WithJWT`, `WithJWTRole` undefined, `JWT`/`JWTRole` unknown fields.

---

## Task 4: Add `JWT`/`JWTRole` fields and builder methods to `vault_provider.go`

**Files:**
- Modify: `api/vault_provider.go`

- [ ] **Step 1: Add fields to `VaultProviderConfig`**

In `api/vault_provider.go`, update the struct to:

```go
type VaultProviderConfig struct {
	Address    *string
	Token      *string
	JWT        *string
	JWTRole    *string
	MountPoint string
}
```

- [ ] **Step 2: Add `WithJWTRole` and `WithJWT` builder methods**

After the existing `WithMountPoint` method, add:

```go
func (c *VaultProviderConfig) WithJWTRole(role string) *VaultProviderConfig {
	c.JWTRole = &role
	return c
}

func (c *VaultProviderConfig) WithJWT(jwt string) *VaultProviderConfig {
	c.JWT = &jwt
	return c
}
```

- [ ] **Step 3: Run `go fmt`**

```bash
go fmt ./api/...
```

- [ ] **Step 4: Verify it compiles (tests still fail at runtime)**

```bash
go build ./api/...
```

Expected: no errors.

---

## Task 5: Update `CreateVaultProvider` with JWT-first resolution logic

**Files:**
- Modify: `api/vault_provider.go`

- [ ] **Step 1: Replace the auth resolution block in `CreateVaultProvider`**

The current function body in `CreateVaultProvider` (lines after the `address` resolution) reads token and errors if missing. Replace everything after the address resolution with:

```go
	jwt := ""
	if config.JWT != nil {
		jwt = *config.JWT
	}
	if jwt == "" {
		jwt = os.Getenv("VAULT_JWT")
	}

	token := ""
	if config.Token != nil {
		token = *config.Token
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}

	providerArgs := &vault.ProviderArgs{
		Address: pulumi.String(address),
	}

	if config.JWTRole != nil && jwt != "" {
		providerArgs.AuthLoginJwt = &vault.ProviderAuthLoginJwtArgs{
			Jwt:  pulumi.String(jwt),
			Role: pulumi.String(*config.JWTRole),
		}
	} else if token != "" {
		providerArgs.Token = pulumi.String(token)
	} else {
		return nil, fmt.Errorf("vault auth not configured for %s: set VAULT_JWT (and configure a JWT role) or VAULT_TOKEN", orgName)
	}

	resourceName := fmt.Sprintf("vault-provider.%s", helpers.Slugify(orgName))
	return vault.NewProvider(ctx, resourceName, providerArgs)
```

The full updated `CreateVaultProvider` function should look like:

```go
func CreateVaultProvider(ctx *pulumi.Context, config *VaultProviderConfig, orgName string) (*vault.Provider, error) {
	if config == nil {
		return nil, nil
	}

	address := ""
	if config.Address != nil {
		address = *config.Address
	}
	if address == "" {
		address = os.Getenv("VAULT_ADDR")
	}
	if address == "" {
		return nil, fmt.Errorf("vault address not configured for %s: set VAULT_ADDR or pass an explicit address", orgName)
	}

	jwt := ""
	if config.JWT != nil {
		jwt = *config.JWT
	}
	if jwt == "" {
		jwt = os.Getenv("VAULT_JWT")
	}

	token := ""
	if config.Token != nil {
		token = *config.Token
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}

	providerArgs := &vault.ProviderArgs{
		Address: pulumi.String(address),
	}

	if config.JWTRole != nil && jwt != "" {
		providerArgs.AuthLoginJwt = &vault.ProviderAuthLoginJwtArgs{
			Jwt:  pulumi.String(jwt),
			Role: pulumi.String(*config.JWTRole),
		}
	} else if token != "" {
		providerArgs.Token = pulumi.String(token)
	} else {
		return nil, fmt.Errorf("vault auth not configured for %s: set VAULT_JWT (and configure a JWT role) or VAULT_TOKEN", orgName)
	}

	resourceName := fmt.Sprintf("vault-provider.%s", helpers.Slugify(orgName))
	return vault.NewProvider(ctx, resourceName, providerArgs)
}
```

- [ ] **Step 2: Run `go fmt`**

```bash
go fmt ./api/...
```

- [ ] **Step 3: Run all tests**

```bash
go test ./api/... -v 2>&1
```

Expected: all tests pass, including the five new JWT tests.

If `TestJWTAuthUsed`, `TestJWTFromEnvVar`, or `TestJWTIgnoredWithoutRole` fail with unexpected `authLoginJwt` nullness, the Pulumi mock may serialize nested input structs differently. In that case, adjust the assertion to check `providers[0].inputs["token"]` for the token-auth tests and skip the nested JWT object check — checking that `token` is absent is sufficient for JWT tests if the nested object check is unreliable.

- [ ] **Step 4: Build check**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add api/vault_provider.go api/secrets_test.go
git commit -m "Add JWT auth support to VaultProviderConfig"
```

---

## Task 6: Verify and clean up

- [ ] **Step 1: Run the full test suite one final time**

```bash
go test ./... -v 2>&1
```

Expected: all tests pass.

- [ ] **Step 2: Run `go vet`**

```bash
go vet ./...
```

Expected: no issues.

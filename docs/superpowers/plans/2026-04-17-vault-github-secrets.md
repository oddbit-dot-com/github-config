# Vault-Backed GitHub Actions Secrets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Vault KV v2-backed GitHub Actions secrets (repository, organization, and deployment environment) to the Pulumi GitHub config codebase.

**Architecture:** A new `api/vault_provider.go` holds all vault-specific types and the `NewVaultProviderConfig`/`CreateVaultProvider` functions, mirroring the existing `api/provider.go` pattern. Secrets provisioning lives in `api/organization_secrets.go` and `api/repository_secrets.go` as methods on the existing `Organization` and `Repository` types. Secret values are read at Pulumi apply time using `kv.LookupSecretV2` (a Pulumi data source) and passed as `pulumi.StringPtrInput` into GitHub Actions secret resources.

**Tech Stack:** Go, Pulumi SDK v3, `pulumi-vault/sdk/v6` (v6.7.4, already added to go.mod), `pulumi-github/sdk/v6`.

---

## File Map

| File | Role |
|------|------|
| `api/vault_provider.go` | **New.** `VaultProviderConfig`, `VaultSecretRef`, `OrgSecretRef`, `ActionsSecrets`, `OrgActionsSecrets`, `NewVaultProviderConfig`, `CreateVaultProvider` |
| `api/organization.go` | **Modify.** Add `VaultProviderConfig`, `Secrets`, `vaultProvider` fields; call `createVaultProvider` and `ensureOrgSecrets` from `Ensure` |
| `api/organization_secrets.go` | **New.** `ensureOrgSecrets` method on `Organization` |
| `api/repository.go` | **Modify.** Add `Secrets`, `Environments`, `EnvironmentSecrets` fields; call three new methods from `Ensure` |
| `api/repository_secrets.go` | **New.** `ensureEnvironments`, `ensureRepoSecrets`, `ensureEnvironmentSecrets` methods on `Repository` |
| `api/secrets_test.go` | **New.** All secret-related tests; extends `mockMonitor` to handle vault invoke calls |

---

## Task 1: Add vault_provider.go with types and constructor

**Files:**
- Create: `api/vault_provider.go`

- [ ] **Step 1: Create the file**

```go
package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/oddbit-dot-com/github-config/helpers"
	vault "github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VaultProviderConfig struct {
	Address    *string
	Token      *string
	MountPoint string
}

type VaultSecretRef struct {
	Path string
	Key  string
}

type OrgSecretRef struct {
	VaultSecretRef
	Visibility string
}

type ActionsSecrets map[string]VaultSecretRef
type OrgActionsSecrets map[string]OrgSecretRef

func NewVaultProviderConfig(address, token string, mountpoint ...string) *VaultProviderConfig {
	if address == "" {
		address = os.Getenv("VAULT_ADDR")
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	mount := "secret"
	if len(mountpoint) > 0 && mountpoint[0] != "" {
		mount = mountpoint[0]
	}
	cfg := &VaultProviderConfig{MountPoint: strings.TrimPrefix(strings.TrimSuffix(mount, "/"), "/")}
	if address != "" {
		cfg.Address = &address
	}
	if token != "" {
		cfg.Token = &token
	}
	return cfg
}

func CreateVaultProvider(ctx *pulumi.Context, config *VaultProviderConfig, orgName string) (*vault.Provider, error) {
	if config == nil {
		return nil, nil
	}
	if config.Address == nil {
		return nil, fmt.Errorf("vault address not configured for %s: set VAULT_ADDR or pass an explicit address", orgName)
	}
	if config.Token == nil {
		return nil, fmt.Errorf("vault token not configured for %s: set VAULT_TOKEN or pass an explicit token", orgName)
	}

	providerArgs := &vault.ProviderArgs{
		Address:      pulumi.String(*config.Address),
		Token:        pulumi.String(*config.Token),
		SkipTlsVerify: pulumi.Bool(false),
	}

	resourceName := fmt.Sprintf("vault-provider.%s", helpers.Slugify(orgName))
	return vault.NewProvider(ctx, resourceName, providerArgs)
}
```

- [ ] **Step 2: Run `go fmt` and build check**

```bash
cd /home/lars/projects/oddbit-dot-com/github-config
go fmt ./api/...
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add api/vault_provider.go go.mod go.sum
git commit -m "feat: add VaultProviderConfig types and NewVaultProviderConfig/CreateVaultProvider"
```

---

## Task 2: Update Organization struct and Ensure()

**Files:**
- Modify: `api/organization.go`

- [ ] **Step 1: Add imports for vault provider**

In `api/organization.go`, add the vault import alongside existing imports:

```go
import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	vault "github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)
```

- [ ] **Step 2: Add new fields to Organization struct**

After the `provider *github.Provider` field, add:

```go
// Optional Vault provider configuration for secrets management
VaultProviderConfig *VaultProviderConfig

// Org-level GitHub Actions secrets (secret name → vault KV v2 ref + visibility)
Secrets OrgActionsSecrets

// Cached vault provider instance (created in Ensure)
vaultProvider *vault.Provider
```

- [ ] **Step 3: Update Ensure() to create vault provider and provision org secrets**

After the existing `o.provider = provider` line in `Ensure()`, add:

```go
// Create vault provider if configured
vaultProvider, err := CreateVaultProvider(ctx, o.VaultProviderConfig, o.Name)
if err != nil {
    return fmt.Errorf("failed to create vault provider for %s: %w", o.Name, err)
}
o.vaultProvider = vaultProvider

// Provision org-level secrets
if err := o.ensureOrgSecrets(ctx, provider); err != nil {
    return fmt.Errorf("failed to ensure org secrets for %s: %w", o.Name, err)
}
```

- [ ] **Step 4: Build check**

```bash
go build ./...
```

Expected: compilation error about undefined `ensureOrgSecrets` — that's fine, it will be added in Task 3.

- [ ] **Step 5: Commit**

```bash
git add api/organization.go
git commit -m "feat: add VaultProviderConfig and Secrets fields to Organization"
```

---

## Task 3: Add organization_secrets.go with ensureOrgSecrets

**Files:**
- Create: `api/organization_secrets.go`

- [ ] **Step 1: Write the failing test first**

Create `api/secrets_test.go` with the org secrets tests:

```go
package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// vaultSecrets maps vault path → key → value for mock responses
func (m *mockMonitor) withVaultSecrets(secrets map[string]map[string]string) *mockMonitor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vaultData = secrets
	return m
}

// extend mockMonitor with vault data field
// NOTE: Add vaultData field to the mockMonitor struct in repository_permissions_test.go:
//   vaultData map[string]map[string]string

// Call handles vault:kv/getSecretV2:getSecretV2 invokes by returning mock data.
// It replaces the empty implementation in repository_permissions_test.go.
func (m *mockMonitor) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token == "vault:kv/getSecretV2:getSecretV2" {
		m.mu.Lock()
		data := m.vaultData
		m.mu.Unlock()
		path, _ := args.Args["name"].(string)
		if kv, ok := data[path]; ok {
			jsonBytes, _ := json.Marshal(kv)
			return resource.PropertyMap{
				"dataJson": resource.NewStringProperty(string(jsonBytes)),
				"mount":    resource.NewStringProperty("secret"),
				"name":     resource.NewStringProperty(path),
				"id":       resource.NewStringProperty(path),
			}, nil
		}
	}
	return resource.PropertyMap{}, nil
}

func testVaultConfig() *VaultProviderConfig {
	addr := "http://localhost:8200"
	token := "test-token"
	return &VaultProviderConfig{Address: &addr, Token: &token, MountPoint: "secret"}
}

func TestOrgSecretNoVaultProvider(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name: "testorg",
			Secrets: OrgActionsSecrets{
				"MY_SECRET": {VaultSecretRef: VaultSecretRef{Path: "mypath", Key: "mykey"}},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err == nil {
		t.Fatal("expected error when Secrets set but no VaultProviderConfig")
	}
	if !strings.Contains(err.Error(), "vault") {
		t.Errorf("expected vault-related error, got: %v", err)
	}
}

func TestOrgSecretProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	mocks.withVaultSecrets(map[string]map[string]string{
		"mypath": {"mykey": "supersecret"},
	})
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfig(),
			Secrets: OrgActionsSecrets{
				"MY_SECRET": {VaultSecretRef: VaultSecretRef{Path: "mypath", Key: "mykey"}},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	secrets := mocks.findByType("github:index/actionsOrganizationSecret:ActionsOrganizationSecret")
	if len(secrets) != 1 {
		t.Fatalf("expected 1 ActionsOrganizationSecret, got %d", len(secrets))
	}
	if secrets[0].inputs["secretName"].StringValue() != "MY_SECRET" {
		t.Errorf("expected secretName=MY_SECRET, got %v", secrets[0].inputs["secretName"])
	}
}

func TestOrgSecretVisibilityDefault(t *testing.T) {
	mocks := &mockMonitor{}
	mocks.withVaultSecrets(map[string]map[string]string{
		"mypath": {"mykey": "val"},
	})
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfig(),
			Secrets: OrgActionsSecrets{
				"S": {VaultSecretRef: VaultSecretRef{Path: "mypath", Key: "mykey"}},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	secrets := mocks.findByType("github:index/actionsOrganizationSecret:ActionsOrganizationSecret")
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].inputs["visibility"].StringValue() != "all" {
		t.Errorf("expected visibility=all, got %v", secrets[0].inputs["visibility"])
	}
}

func TestOrgSecretCustomVisibility(t *testing.T) {
	mocks := &mockMonitor{}
	mocks.withVaultSecrets(map[string]map[string]string{
		"mypath": {"mykey": "val"},
	})
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfig(),
			Secrets: OrgActionsSecrets{
				"S": {VaultSecretRef: VaultSecretRef{Path: "mypath", Key: "mykey"}, Visibility: "private"},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	secrets := mocks.findByType("github:index/actionsOrganizationSecret:ActionsOrganizationSecret")
	if secrets[0].inputs["visibility"].StringValue() != "private" {
		t.Errorf("expected visibility=private, got %v", secrets[0].inputs["visibility"])
	}
}
```

- [ ] **Step 2: Add `vaultData` field to `mockMonitor`**

In `api/repository_permissions_test.go`, add the `vaultData` field to the struct and remove the old `Call` method body (since `secrets_test.go` provides the full implementation):

```go
type mockMonitor struct {
	mu        sync.Mutex
	resources []mockResource
	vaultData map[string]map[string]string
}
```

Remove the existing `Call` method from `repository_permissions_test.go` entirely (the full implementation is in `secrets_test.go`).

- [ ] **Step 3: Run failing tests**

```bash
go test ./api/... -run "TestOrgSecret" -v 2>&1 | head -30
```

Expected: compile errors about `ensureOrgSecrets` undefined.

- [ ] **Step 4: Create `api/organization_secrets.go`**

```go
package api

import (
	"encoding/json"
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi-vault/sdk/v6/go/vault/kv"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (o *Organization) ensureOrgSecrets(ctx *pulumi.Context, githubProvider *github.Provider) error {
	if len(o.Secrets) == 0 {
		return nil
	}
	if o.VaultProviderConfig == nil {
		return fmt.Errorf("organization %s has Secrets but no VaultProviderConfig", o.Name)
	}

	var githubOpts []pulumi.ResourceOption
	if githubProvider != nil {
		githubOpts = append(githubOpts, pulumi.Provider(githubProvider))
	}

	for secretName, ref := range o.Secrets {
		value, err := o.readVaultSecret(ctx, ref.VaultSecretRef)
		if err != nil {
			return fmt.Errorf("failed to read vault secret for %s/%s: %w", o.Name, secretName, err)
		}

		visibility := ref.Visibility
		if visibility == "" {
			visibility = "all"
		}

		resourceName := fmt.Sprintf("github_actions_organization_secret.%s.%s",
			helpers.Slugify(o.Name), helpers.Slugify(secretName))
		_, err = github.NewActionsOrganizationSecret(ctx, resourceName, &github.ActionsOrganizationSecretArgs{
			SecretName:     pulumi.String(secretName),
			Visibility:     pulumi.String(visibility),
			PlaintextValue: value,
		}, githubOpts...)
		if err != nil {
			return fmt.Errorf("failed to create org secret %s/%s: %w", o.Name, secretName, err)
		}
	}
	return nil
}

func (o *Organization) readVaultSecret(ctx *pulumi.Context, ref VaultSecretRef) (pulumi.StringPtrInput, error) {
	result, err := kv.LookupSecretV2(ctx, &kv.LookupSecretV2Args{
		Mount: o.VaultProviderConfig.MountPoint,
		Name:  ref.Path,
	}, pulumi.Provider(o.vaultProvider))
	if err != nil {
		return nil, err
	}
	// LookupSecretV2 is synchronous: result.DataJson is a plain Go string.
	var data map[string]any
	if err := json.Unmarshal([]byte(result.DataJson), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON from vault at %s: %w", ref.Path, err)
	}
	v, ok := data[ref.Key].(string)
	if !ok {
		return nil, fmt.Errorf("key %q not found or not a string in vault secret at %s", ref.Key, ref.Path)
	}
	return pulumi.String(v).ToStringPtrOutput(), nil
}
```

- [ ] **Step 5: Run the org secret tests**

```bash
go test ./api/... -run "TestOrgSecret" -v
```

Expected: all four org secret tests pass.

- [ ] **Step 6: Run full test suite to check for regressions**

```bash
go test ./api/... -v 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add api/organization_secrets.go api/secrets_test.go api/repository_permissions_test.go
git commit -m "feat: add org-level vault-backed GitHub Actions secrets"
```

---

## Task 4: Add Secrets, Environments, EnvironmentSecrets to Repository

**Files:**
- Modify: `api/repository.go`

- [ ] **Step 1: Add new fields to Repository struct**

After the `Collaborators` field, add:

```go
// Secrets maps GitHub Actions secret names to Vault KV v2 references.
// Requires the parent Organization to have a VaultProviderConfig.
Secrets ActionsSecrets

// Environments maps GitHub deployment environment names to their configuration.
// An empty *github.RepositoryEnvironmentArgs ({}) creates a bare environment.
Environments map[string]*github.RepositoryEnvironmentArgs

// EnvironmentSecrets maps deployment environment names to their Actions secrets.
// Every key must also appear in Environments — referencing an undeclared environment is an error.
EnvironmentSecrets map[string]ActionsSecrets
```

- [ ] **Step 2: Add three method calls to Ensure()**

At the end of `Ensure()`, after the `NewIssueLabels` call, add:

```go
if err := r.ensureEnvironments(ctx, repo, opts); err != nil {
    return err
}

if err := r.ensureRepoSecrets(ctx, opts); err != nil {
    return err
}

if err := r.ensureEnvironmentSecrets(ctx, repo, opts); err != nil {
    return err
}
```

- [ ] **Step 3: Build check (expected failure)**

```bash
go build ./...
```

Expected: undefined `ensureEnvironments`, `ensureRepoSecrets`, `ensureEnvironmentSecrets` — that's fine.

- [ ] **Step 4: Commit**

```bash
git add api/repository.go
git commit -m "feat: add Secrets, Environments, EnvironmentSecrets fields to Repository"
```

---

## Task 5: Add repository_secrets.go with all three methods

**Files:**
- Create: `api/repository_secrets.go`

- [ ] **Step 1: Write the failing tests**

Append to `api/secrets_test.go`:

```go
func TestRepoSecretNoVaultProvider(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
			Secrets: ActionsSecrets{
				"MY_SECRET": {Path: "mypath", Key: "mykey"},
			},
		}
		return repo.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err == nil {
		t.Fatal("expected error when repo has Secrets but no org vault provider")
	}
	if !strings.Contains(err.Error(), "vault") {
		t.Errorf("expected vault-related error, got: %v", err)
	}
}

func TestRepoSecretProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	mocks.withVaultSecrets(map[string]map[string]string{
		"mypath": {"mykey": "reposecret"},
	})
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfig(),
			Repositories: []*Repository{
				{
					Name:           "test-repo",
					RepositoryArgs: &github.RepositoryArgs{},
					Secrets: ActionsSecrets{
						"DB_PASSWORD": {Path: "mypath", Key: "mykey"},
					},
				},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	secrets := mocks.findByType("github:index/actionsSecret:ActionsSecret")
	if len(secrets) != 1 {
		t.Fatalf("expected 1 ActionsSecret, got %d", len(secrets))
	}
	if secrets[0].inputs["secretName"].StringValue() != "DB_PASSWORD" {
		t.Errorf("expected secretName=DB_PASSWORD, got %v", secrets[0].inputs["secretName"])
	}
}

func TestRepoEnvironmentCreated(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
			Environments: map[string]*github.RepositoryEnvironmentArgs{
				"production": {},
			},
		}
		return repo.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	envs := mocks.findByType("github:index/repositoryEnvironment:RepositoryEnvironment")
	if len(envs) != 1 {
		t.Fatalf("expected 1 RepositoryEnvironment, got %d", len(envs))
	}
	if envs[0].inputs["environment"].StringValue() != "production" {
		t.Errorf("expected environment=production, got %v", envs[0].inputs["environment"])
	}
}

func TestEnvSecretMissingEnvironment(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfig(),
			Repositories: []*Repository{
				{
					Name:           "test-repo",
					RepositoryArgs: &github.RepositoryArgs{},
					// No Environments declared
					EnvironmentSecrets: map[string]ActionsSecrets{
						"production": {
							"API_KEY": {Path: "p", Key: "k"},
						},
					},
				},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err == nil {
		t.Fatal("expected error when EnvironmentSecrets references undeclared environment")
	}
	if !strings.Contains(err.Error(), "production") {
		t.Errorf("expected error to mention environment name, got: %v", err)
	}
}

func TestEnvSecretProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	mocks.withVaultSecrets(map[string]map[string]string{
		"prod/apikey": {"key": "thevalue"},
	})
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Name:                "testorg",
			VaultProviderConfig: testVaultConfig(),
			Repositories: []*Repository{
				{
					Name:           "test-repo",
					RepositoryArgs: &github.RepositoryArgs{},
					Environments: map[string]*github.RepositoryEnvironmentArgs{
						"production": {},
					},
					EnvironmentSecrets: map[string]ActionsSecrets{
						"production": {
							"PROD_API_KEY": {Path: "prod/apikey", Key: "key"},
						},
					},
				},
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	envSecrets := mocks.findByType("github:index/actionsEnvironmentSecret:ActionsEnvironmentSecret")
	if len(envSecrets) != 1 {
		t.Fatalf("expected 1 ActionsEnvironmentSecret, got %d", len(envSecrets))
	}
	if envSecrets[0].inputs["secretName"].StringValue() != "PROD_API_KEY" {
		t.Errorf("expected secretName=PROD_API_KEY, got %v", envSecrets[0].inputs["secretName"])
	}
	if envSecrets[0].inputs["environment"].StringValue() != "production" {
		t.Errorf("expected environment=production, got %v", envSecrets[0].inputs["environment"])
	}
}
```

- [ ] **Step 2: Run failing tests to confirm compilation issues**

```bash
go test ./api/... -run "TestRepo|TestEnv" -v 2>&1 | head -20
```

Expected: compile errors (undefined methods).

- [ ] **Step 3: Create `api/repository_secrets.go`**

```go
package api

import (
	"encoding/json"
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi-vault/sdk/v6/go/vault/kv"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (r *Repository) ensureEnvironments(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	for envName, args := range r.Environments {
		if args == nil {
			args = &github.RepositoryEnvironmentArgs{}
		}
		argsCopy := *args
		argsCopy.Repository = repo.Name
		argsCopy.Environment = pulumi.String(envName)

		resourceName := fmt.Sprintf("github_repository_environment.%s.%s",
			helpers.Slugify(r.Name), helpers.Slugify(envName))
		if _, err := github.NewRepositoryEnvironment(ctx, resourceName, &argsCopy, opts...); err != nil {
			return fmt.Errorf("failed to create environment %s for %s: %w", envName, r.Name, err)
		}
	}
	return nil
}

func (r *Repository) ensureRepoSecrets(ctx *pulumi.Context, opts []pulumi.ResourceOption) error {
	if len(r.Secrets) == 0 {
		return nil
	}
	vaultProvider, err := r.getVaultProvider()
	if err != nil {
		return err
	}

	for secretName, ref := range r.Secrets {
		value, err := r.readVaultSecret(ctx, ref, vaultProvider)
		if err != nil {
			return fmt.Errorf("failed to read vault secret for %s/%s: %w", r.Name, secretName, err)
		}

		resourceName := fmt.Sprintf("github_actions_secret.%s.%s",
			helpers.Slugify(r.Name), helpers.Slugify(secretName))
		_, err = github.NewActionsSecret(ctx, resourceName, &github.ActionsSecretArgs{
			Repository:     pulumi.String(r.Name),
			SecretName:     pulumi.String(secretName),
			PlaintextValue: value,
		}, opts...)
		if err != nil {
			return fmt.Errorf("failed to create repo secret %s/%s: %w", r.Name, secretName, err)
		}
	}
	return nil
}

func (r *Repository) ensureEnvironmentSecrets(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	if len(r.EnvironmentSecrets) == 0 {
		return nil
	}
	vaultProvider, err := r.getVaultProvider()
	if err != nil {
		return err
	}

	for envName, secrets := range r.EnvironmentSecrets {
		if _, declared := r.Environments[envName]; !declared {
			return fmt.Errorf("environment %q referenced in EnvironmentSecrets of %s is not declared in Environments", envName, r.Name)
		}
		for secretName, ref := range secrets {
			value, err := r.readVaultSecret(ctx, ref, vaultProvider)
			if err != nil {
				return fmt.Errorf("failed to read vault secret for %s/%s/%s: %w", r.Name, envName, secretName, err)
			}

			resourceName := fmt.Sprintf("github_actions_environment_secret.%s.%s.%s",
				helpers.Slugify(r.Name), helpers.Slugify(envName), helpers.Slugify(secretName))
			_, err = github.NewActionsEnvironmentSecret(ctx, resourceName, &github.ActionsEnvironmentSecretArgs{
				Repository:     repo.Name,
				Environment:    pulumi.String(envName),
				SecretName:     pulumi.String(secretName),
				PlaintextValue: value,
			}, opts...)
			if err != nil {
				return fmt.Errorf("failed to create env secret %s/%s/%s: %w", r.Name, envName, secretName, err)
			}
		}
	}
	return nil
}

func (r *Repository) getVaultProvider() (*vault.Provider, error) {
	if r.organization == nil || r.organization.vaultProvider == nil {
		return nil, fmt.Errorf("repository %s has secrets but its organization has no vault provider configured", r.Name)
	}
	return r.organization.vaultProvider, nil
}

func (r *Repository) readVaultSecret(ctx *pulumi.Context, ref VaultSecretRef, vaultProvider *vault.Provider) (pulumi.StringPtrInput, error) {
	cfg := r.organization.VaultProviderConfig
	result, err := kv.LookupSecretV2(ctx, &kv.LookupSecretV2Args{
		Mount: cfg.MountPoint,
		Name:  ref.Path,
	}, pulumi.Provider(vaultProvider))
	if err != nil {
		return nil, err
	}
	// LookupSecretV2 is synchronous: result.DataJson is a plain Go string.
	var data map[string]any
	if err := json.Unmarshal([]byte(result.DataJson), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON from vault at %s: %w", ref.Path, err)
	}
	v, ok := data[ref.Key].(string)
	if !ok {
		return nil, fmt.Errorf("key %q not found or not a string in vault secret at %s", ref.Key, ref.Path)
	}
	return pulumi.String(v).ToStringPtrOutput(), nil
}
```

Note: `vault.Provider` in `getVaultProvider` and `readVaultSecret` needs the vault import. Add to imports:
```go
vault "github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
```

- [ ] **Step 4: Run failing tests**

```bash
go test ./api/... -run "TestRepo|TestEnv" -v
```

Expected: all 5 repo/env tests pass.

- [ ] **Step 5: Run full test suite**

```bash
go test ./api/... -v 2>&1 | tail -30
```

Expected: all tests pass.

- [ ] **Step 6: Run `go fmt`**

```bash
go fmt ./api/...
```

- [ ] **Step 7: Commit**

```bash
git add api/repository_secrets.go api/secrets_test.go
git commit -m "feat: add repo and environment vault-backed GitHub Actions secrets"
```

---

## Task 6: Final build, format, and full test verification

**Files:** None new.

- [ ] **Step 1: Full format pass**

```bash
go fmt ./...
```

- [ ] **Step 2: Full build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Full test suite**

```bash
go test ./... -v 2>&1 | tail -40
```

Expected: all tests pass.

- [ ] **Step 4: Final commit if any formatting changes**

```bash
git diff --stat
# If any changes:
git add -u
git commit -m "style: apply go fmt across all packages"
```

---

## Key SDK Notes

- **Vault provider args:** `vault.ProviderArgs{Address: pulumi.StringInput, Token: pulumi.StringInput}` — both required.
- **KV v2 lookup:** `kv.LookupSecretV2(ctx, &kv.LookupSecretV2Args{Mount: string, Name: string}, opts...)` returns `(*LookupSecretV2Result, error)`. `result.DataJson` is a `string` containing JSON of the KV secret's data map.
- **Repo secret:** `github.ActionsSecretArgs{Repository: pulumi.StringInput, SecretName: pulumi.StringInput, PlaintextValue: pulumi.StringPtrInput}`
- **Org secret:** `github.ActionsOrganizationSecretArgs{SecretName: pulumi.StringInput, Visibility: pulumi.StringInput, PlaintextValue: pulumi.StringPtrInput}`
- **Env secret:** `github.ActionsEnvironmentSecretArgs{Repository: pulumi.StringInput, Environment: pulumi.StringInput, SecretName: pulumi.StringInput, PlaintextValue: pulumi.StringPtrInput}`
- **Env resource:** `github.RepositoryEnvironmentArgs{Repository: pulumi.StringInput, Environment: pulumi.StringInput, ...optional fields}`
- **Mock invoke token for vault KV v2:** `"vault:kv/getSecretV2:getSecretV2"`
- **GitHub type tokens for mock assertions:**
  - `"github:index/actionsSecret:ActionsSecret"`
  - `"github:index/actionsOrganizationSecret:ActionsOrganizationSecret"`
  - `"github:index/actionsEnvironmentSecret:ActionsEnvironmentSecret"`
  - `"github:index/repositoryEnvironment:RepositoryEnvironment"`

package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (m *mockMonitor) withVaultSecrets(secrets map[string]map[string]string) *mockMonitor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vaultData = secrets
	return m
}

func (m *mockMonitor) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token == "vault:kv/getSecretV2:getSecretV2" {
		m.mu.Lock()
		data := m.vaultData
		m.mu.Unlock()
		nameVal := args.Args["name"]
		path := ""
		if nameVal.IsString() {
			path = nameVal.StringValue()
		}
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

// Ensure github import is used (for type reference in other test files)
var _ = (*github.Provider)(nil)

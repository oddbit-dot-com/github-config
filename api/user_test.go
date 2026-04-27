package api

import (
	"testing"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestUserSshKeyProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{Name: "testuser"},
			SshKeys: map[string]*github.UserSshKeyArgs{
				"my-key": {Key: pulumi.String("ssh-rsa AAAA...")},
			},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/userSshKey:UserSshKey")
	if len(keys) != 1 {
		t.Fatalf("expected 1 UserSshKey, got %d", len(keys))
	}
	if keys[0].inputs["title"].StringValue() != "my-key" {
		t.Errorf("expected title=my-key, got %v", keys[0].inputs["title"])
	}
}

func TestUserSshKeyExplicitTitle(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{Name: "testuser"},
			SshKeys: map[string]*github.UserSshKeyArgs{
				"my-key": {Title: pulumi.String("Custom Title"), Key: pulumi.String("ssh-rsa AAAA...")},
			},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/userSshKey:UserSshKey")
	if len(keys) != 1 {
		t.Fatalf("expected 1 UserSshKey, got %d", len(keys))
	}
	if keys[0].inputs["title"].StringValue() != "Custom Title" {
		t.Errorf("expected title=Custom Title, got %v", keys[0].inputs["title"])
	}
}

func TestUserMultipleSshKeysProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{Name: "testuser"},
			SshKeys: map[string]*github.UserSshKeyArgs{
				"key-one": {Key: pulumi.String("ssh-rsa AAAA...")},
				"key-two": {Key: pulumi.String("ssh-rsa BBBB...")},
			},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/userSshKey:UserSshKey")
	if len(keys) != 2 {
		t.Fatalf("expected 2 UserSshKeys, got %d", len(keys))
	}
}

func TestUserGpgKeyProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{Name: "testuser"},
			GpgKeys: map[string]*github.UserGpgKeyArgs{
				"primary": {ArmoredPublicKey: pulumi.String("-----BEGIN PGP PUBLIC KEY BLOCK-----\n...\n-----END PGP PUBLIC KEY BLOCK-----")},
			},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/userGpgKey:UserGpgKey")
	if len(keys) != 1 {
		t.Fatalf("expected 1 UserGpgKey, got %d", len(keys))
	}
}

func TestUserRepositoryProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{Name: "testuser"},
			Repositories: []*Repository{
				{
					Name:           "my-repo",
					RepositoryArgs: &github.RepositoryArgs{},
				},
			},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	repos := mocks.findByType("github:index/repository:Repository")
	if len(repos) != 1 {
		t.Fatalf("expected 1 Repository, got %d", len(repos))
	}
}

func TestUserRepoSecretProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	mocks.withVaultSecrets(map[string]map[string]string{
		"mypath": {"mykey": "reposecret"},
	})
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{
				Name:                "testuser",
				VaultProviderConfig: testVaultConfig(),
			},
			Repositories: []*Repository{
				{
					Name:           "my-repo",
					RepositoryArgs: &github.RepositoryArgs{},
					Secrets: ActionsSecrets{
						"DB_PASSWORD": &VaultSecretRef{Path: "mypath", Key: "mykey"},
					},
				},
			},
		}
		return user.Ensure(ctx)
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

func TestUserNoSshKeysOrGpgKeys(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{Name: "testuser"},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	if len(mocks.findByType("github:index/userSshKey:UserSshKey")) != 0 {
		t.Error("expected no SSH keys when SshKeys is nil")
	}
	if len(mocks.findByType("github:index/userGpgKey:UserGpgKey")) != 0 {
		t.Error("expected no GPG keys when GpgKeys is nil")
	}
}

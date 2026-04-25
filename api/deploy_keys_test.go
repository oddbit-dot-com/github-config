package api

import (
	"testing"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestDeployKeyProvisioned(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
			DeployKeys: DeployKeys{
				"my-deploy-key": &DeployKey{
					Key:      &LiteralSecretRef{Value: "ssh-rsa AAAA..."},
					ReadOnly: pulumi.BoolRef(true),
				},
			},
		}
		return repo.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/repositoryDeployKey:RepositoryDeployKey")
	if len(keys) != 1 {
		t.Fatalf("expected 1 RepositoryDeployKey, got %d", len(keys))
	}
	if keys[0].inputs["title"].StringValue() != "my-deploy-key" {
		t.Errorf("expected title=my-deploy-key, got %v", keys[0].inputs["title"])
	}
	if keys[0].inputs["readOnly"].BoolValue() != true {
		t.Errorf("expected readOnly=true, got %v", keys[0].inputs["readOnly"])
	}
}

func TestDeployKeyDefaultReadOnly(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
			DeployKeys: DeployKeys{
				"my-deploy-key": &DeployKey{
					Key: &LiteralSecretRef{Value: "ssh-rsa AAAA..."},
				},
			},
		}
		return repo.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/repositoryDeployKey:RepositoryDeployKey")
	if len(keys) != 1 {
		t.Fatalf("expected 1 RepositoryDeployKey, got %d", len(keys))
	}
	if keys[0].inputs["readOnly"].BoolValue() != true {
		t.Errorf("expected readOnly=true by default, got %v", keys[0].inputs["readOnly"])
	}
}

func TestMultipleDeployKeys(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
			DeployKeys: DeployKeys{
				"read-key": &DeployKey{
					Key: &LiteralSecretRef{Value: "ssh-rsa AAAA1..."},
				},
				"write-key": &DeployKey{
					Key:      &LiteralSecretRef{Value: "ssh-rsa AAAA2..."},
					ReadOnly: pulumi.BoolRef(false),
				},
			},
		}
		return repo.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/repositoryDeployKey:RepositoryDeployKey")
	if len(keys) != 2 {
		t.Fatalf("expected 2 RepositoryDeployKeys, got %d", len(keys))
	}
}

func TestNoDeployKeys(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
		}
		return repo.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}
	keys := mocks.findByType("github:index/repositoryDeployKey:RepositoryDeployKey")
	if len(keys) != 0 {
		t.Fatalf("expected 0 RepositoryDeployKeys, got %d", len(keys))
	}
}

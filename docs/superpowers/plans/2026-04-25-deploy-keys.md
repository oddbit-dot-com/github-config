# Deploy Keys Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add deploy key provisioning support to `api.Repository`.

**Architecture:** Add a `DeployKeys` field to the `Repository` struct using the existing map-based convention (`map[string]*github.RepositoryDeployKeyArgs`, keyed by title). Add an `ensureDeployKeys` method following the same pattern as `ensureEnvironments`, and wire it into `Ensure()`. Test with the existing `mockMonitor` infrastructure.

**Tech Stack:** Go, Pulumi GitHub provider v6 (`github.NewRepositoryDeployKey`)

**Note:** The user mentioned `github.RepositoryKey` but the Pulumi GitHub SDK v6 type is `github.RepositoryDeployKeyArgs`. This plan uses the correct SDK type. The user mentioned "a list" but the codebase convention is maps keyed by a logical name (see `Environments`, `Teams`, `Collaborators`); this plan follows the existing convention.

---

### Task 1: Add DeployKeys field to Repository struct

**Files:**
- Modify: `api/repository.go:64` (add field after `Environments`)

- [ ] **Step 1: Add the DeployKeys field**

Add the field after the `Environments` field (line 65) in the `Repository` struct:

```go
// DeployKeys maps deploy key titles to their Pulumi configuration.
DeployKeys map[string]*github.RepositoryDeployKeyArgs
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: success (field is defined but not yet used anywhere)

- [ ] **Step 3: Commit**

```bash
git add api/repository.go
git commit -m "Add DeployKeys field to Repository struct"
```

---

### Task 2: Write failing test for deploy key provisioning

**Files:**
- Create: `api/deploy_keys_test.go`

- [ ] **Step 1: Write the test**

```go
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
			DeployKeys: map[string]*github.RepositoryDeployKeyArgs{
				"my-deploy-key": {
					Key:      pulumi.String("ssh-rsa AAAA..."),
					ReadOnly: pulumi.Bool(true),
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

func TestMultipleDeployKeys(t *testing.T) {
	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repo := &Repository{
			Name:           "test-repo",
			RepositoryArgs: &github.RepositoryArgs{},
			DeployKeys: map[string]*github.RepositoryDeployKeyArgs{
				"read-key": {
					Key:      pulumi.String("ssh-rsa AAAA1..."),
					ReadOnly: pulumi.Bool(true),
				},
				"write-key": {
					Key:      pulumi.String("ssh-rsa AAAA2..."),
					ReadOnly: pulumi.Bool(false),
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./api/ -run TestDeployKeyProvisioned -v`
Expected: FAIL — `ensureDeployKeys` does not exist yet, so no `RepositoryDeployKey` resources are created; test expects 1 but gets 0.

- [ ] **Step 3: Commit**

```bash
git add api/deploy_keys_test.go
git commit -m "Add failing tests for deploy key provisioning"
```

---

### Task 3: Implement ensureDeployKeys and wire into Ensure

**Files:**
- Modify: `api/repository.go:78-190` (add call in `Ensure`)
- Modify: `api/repository.go` (add `ensureDeployKeys` method)

- [ ] **Step 1: Add the ensureDeployKeys method**

Add this method after `ensureCollaborators` (after line 220) in `api/repository.go`:

```go
func (r *Repository) ensureDeployKeys(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	for title, args := range r.DeployKeys {
		if args == nil {
			args = &github.RepositoryDeployKeyArgs{}
		}
		argsCopy := *args
		argsCopy.Repository = repo.Name
		argsCopy.Title = pulumi.String(title)

		resourceName := r.resourceName("github_repository_deploy_key", title)
		if _, err := github.NewRepositoryDeployKey(ctx, resourceName, &argsCopy, opts...); err != nil {
			return fmt.Errorf("failed to create deploy key %s for %s: %w", title, r.Name, err)
		}
	}
	return nil
}
```

- [ ] **Step 2: Wire ensureDeployKeys into Ensure**

Add the call in the `Ensure` method, after the `ensureCollaborators` call (after line 156):

```go
if err := r.ensureDeployKeys(ctx, repo, opts); err != nil {
	return err
}
```

- [ ] **Step 3: Run the tests to verify they pass**

Run: `go test ./api/ -run TestDeployKey -v && go test ./api/ -run TestMultipleDeployKeys -v && go test ./api/ -run TestNoDeployKeys -v`
Expected: all PASS

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`
Expected: all PASS, no regressions

- [ ] **Step 5: Verify the project builds**

Run: `go build ./...`
Expected: success

- [ ] **Step 6: Commit**

```bash
git add api/repository.go api/deploy_keys_test.go
git commit -m "Implement deploy key provisioning for repositories"
```

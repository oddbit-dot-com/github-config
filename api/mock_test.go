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

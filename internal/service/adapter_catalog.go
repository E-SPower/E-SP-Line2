package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConfigField describes one field of an adapter's instance configuration.
type ConfigField struct {
	Label       string `json:"label" yaml:"label"`
	Type        string `json:"type" yaml:"type"` // text, password, number
	Required    bool   `json:"required" yaml:"required"`
	Placeholder string `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Help        string `json:"help,omitempty" yaml:"help,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
}

// CatalogAdapter is the parsed representation of an adapter.yaml file.
type CatalogAdapter struct {
	ID          string                 `json:"id" yaml:"id"`
	PlatformCode string                `json:"platform_code" yaml:"platform_code"`
	Name        string                 `json:"name" yaml:"name"`
	Version     string                 `json:"version" yaml:"version"`
	RuntimeType string                 `json:"runtime_type" yaml:"runtime_type"`
	Description string                 `json:"description" yaml:"description"`
	Hidden      bool                   `json:"hidden" yaml:"hidden"`
	Icon        string                 `json:"icon,omitempty" yaml:"icon,omitempty"`
	ConfigSchema map[string]ConfigField `json:"config_schema" yaml:"config_schema"`
	Capabilities []string              `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`

	// Directory path (not serialized by default unless needed).
	Dir string `json:"-" yaml:"-"`
}

// AdapterCatalog scans the adapters/ directory for adapter.yaml files and
// exposes the list of available adapters, their instance config schemas and
// hidden flags. 接入器是"现实存在"的：每个子目录 + adapter.yaml 定义一个接入器。
type AdapterCatalog struct {
	mu       sync.RWMutex
	rootDir  string
	adapters map[string]*CatalogAdapter
	order    []string
}

// NewAdapterCatalog scans adaptersDir for adapter.yaml files.
func NewAdapterCatalog(adaptersDir string) (*AdapterCatalog, error) {
	c := &AdapterCatalog{
		rootDir:  adaptersDir,
		adapters: make(map[string]*CatalogAdapter),
	}
	if err := c.Reload(); err != nil {
		return nil, err
	}
	return c, nil
}

// Reload rescans the adapters directory.
func (c *AdapterCatalog) Reload() error {
	entries, err := os.ReadDir(c.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No adapters dir yet — treat as empty catalog.
			c.mu.Lock()
			c.adapters = make(map[string]*CatalogAdapter)
			c.order = nil
			c.mu.Unlock()
			return nil
		}
		return fmt.Errorf("failed to read adapters dir: %w", err)
	}

	newMap := make(map[string]*CatalogAdapter)
	var order []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		yamlPath := filepath.Join(c.rootDir, entry.Name(), "adapter.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			// Directory without adapter.yaml is not an adapter.
			continue
		}
		var a CatalogAdapter
		if err := yaml.Unmarshal(data, &a); err != nil {
			return fmt.Errorf("failed to parse %s: %w", yamlPath, err)
		}
		a.ID = strings.TrimSpace(a.ID)
		if a.ID == "" {
			return fmt.Errorf("adapter.yaml %s missing required field: id", yamlPath)
		}
		a.Dir = entry.Name()
		if a.ConfigSchema == nil {
			a.ConfigSchema = map[string]ConfigField{}
		}
		if _, exists := newMap[a.ID]; exists {
			return fmt.Errorf("duplicate adapter id: %s", a.ID)
		}
		newMap[a.ID] = &a
		order = append(order, a.ID)
	}
	sort.Strings(order)

	c.mu.Lock()
	c.adapters = newMap
	c.order = order
	c.mu.Unlock()
	return nil
}

// All returns all adapters, optionally excluding hidden ones.
func (c *AdapterCatalog) All(includeHidden bool) []*CatalogAdapter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*CatalogAdapter, 0, len(c.order))
	for _, id := range c.order {
		a := c.adapters[id]
		if a == nil {
			continue
		}
		if !includeHidden && a.Hidden {
			continue
		}
		result = append(result, a)
	}
	return result
}

// Get returns a single adapter by id.
func (c *AdapterCatalog) Get(id string) (*CatalogAdapter, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.adapters[id]
	return a, ok
}

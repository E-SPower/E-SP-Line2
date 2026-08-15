package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// OptionItem represents a single selectable option in a form dropdown.
type OptionItem struct {
	Value   string `json:"value" yaml:"value"`
	Label   string `json:"label" yaml:"label"`
	LabelEN string `json:"label_en,omitempty" yaml:"label_en,omitempty"`
}

// OptionGroup represents a named group of form options loaded from YAML.
type OptionGroup struct {
	Key     string       `json:"key"`
	Options []OptionItem `json:"options"`
}

// FormOptions is the root structure of the form-options.yaml registry file.
type FormOptions struct {
	PlatformCodes    []OptionItem `yaml:"platform_codes"`
	RuntimeTypes     []OptionItem `yaml:"runtime_types"`
	AdapterStatuses  []OptionItem `yaml:"adapter_statuses"`
	InstanceStatuses []OptionItem `yaml:"instance_statuses"`
	MessageTypes     []OptionItem `yaml:"message_types"`
	RouteTargetTypes []OptionItem `yaml:"route_target_types"`
	BooleanOptions   []OptionItem `yaml:"boolean_options"`
	PlatformStatuses []OptionItem `yaml:"platform_statuses"`
}

// OptionService loads form dropdown options from the YAML registry file.
type OptionService struct {
	mu      sync.RWMutex
	path    string
	options FormOptions
}

// NewOptionService creates an OptionService and loads the options file.
func NewOptionService(path string) (*OptionService, error) {
	s := &OptionService{path: path}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// reload reads and parses the YAML registry file into memory.
func (s *OptionService) reload() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("failed to read form options file %s: %w", s.path, err)
	}

	var options FormOptions
	if err := yaml.Unmarshal(data, &options); err != nil {
		return fmt.Errorf("failed to parse form options file %s: %w", s.path, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.options = options
	return nil
}

// AllGroups returns every registered option group.
func (s *OptionService) AllGroups() []OptionGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return []OptionGroup{
		{Key: "platform_codes", Options: s.options.PlatformCodes},
		{Key: "runtime_types", Options: s.options.RuntimeTypes},
		{Key: "adapter_statuses", Options: s.options.AdapterStatuses},
		{Key: "instance_statuses", Options: s.options.InstanceStatuses},
		{Key: "message_types", Options: s.options.MessageTypes},
		{Key: "route_target_types", Options: s.options.RouteTargetTypes},
		{Key: "boolean_options", Options: s.options.BooleanOptions},
		{Key: "platform_statuses", Options: s.options.PlatformStatuses},
	}
}

// Get returns the option group identified by key.
func (s *OptionService) Get(key string) (OptionGroup, bool) {
	for _, g := range s.AllGroups() {
		if g.Key == key {
			return g, true
		}
	}
	return OptionGroup{}, false
}

// ResolveFile returns the resolved absolute path to the options file.
func (s *OptionService) ResolveFile() string {
	abs, err := filepath.Abs(s.path)
	if err != nil {
		return s.path
	}
	return abs
}

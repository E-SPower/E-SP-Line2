package adapter

import (
	"encoding/json"
	"sync"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// AdapterInfo represents adapter information
type AdapterInfo struct {
	ID              string                     `json:"id"`
	PlatformID      string                     `json:"platform_id"`
	Name            string                     `json:"name"`
	Version         string                     `json:"version"`
	RuntimeType     string                     `json:"runtime_type"` // python, node, go
	ProtocolVersion string                     `json:"protocol_version"`
	Capabilities    []string                   `json:"capabilities"`
	ConfigSchema    map[string]interface{}     `json:"config_schema"`
	I18n            map[string]v3.I18nResource `json:"i18n"`
	Operations      v3.OperationPolicy         `json:"operations"`
	Security        v3.SecurityPolicy          `json:"security"`
	Status          string                     `json:"status"` // active, deprecated, disabled
}

// Registry manages adapter packages
type Registry struct {
	adapters map[string]*AdapterInfo
	mu       sync.RWMutex
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]*AdapterInfo),
	}
}

// Register registers an adapter
func (r *Registry) Register(adapter *AdapterInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if adapter.ID == "" {
		return ErrInvalidAdapterID
	}

	if _, exists := r.adapters[adapter.ID]; exists {
		return ErrAdapterAlreadyExists
	}

	r.adapters[adapter.ID] = adapter
	logger.Info("Adapter registered",
		logger.String("id", adapter.ID),
		logger.String("platform", adapter.PlatformID),
		logger.String("version", adapter.Version))

	return nil
}

// Unregister unregisters an adapter
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[id]; !exists {
		return ErrAdapterNotFound
	}

	delete(r.adapters, id)
	logger.Info("Adapter unregistered", logger.String("id", id))

	return nil
}

// Get gets an adapter by ID
func (r *Registry) Get(id string) (*AdapterInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[id]
	if !exists {
		return nil, ErrAdapterNotFound
	}

	return adapter, nil
}

// GetByPlatform gets adapters by platform ID
func (r *Registry) GetByPlatform(platformID string) []*AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AdapterInfo, 0)
	for _, adapter := range r.adapters {
		if adapter.PlatformID == platformID {
			result = append(result, adapter)
		}
	}

	return result
}

// GetAll gets all adapters
func (r *Registry) GetAll() []*AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AdapterInfo, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		result = append(result, adapter)
	}

	return result
}

// Update updates an adapter
func (r *Registry) Update(adapter *AdapterInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[adapter.ID]; !exists {
		return ErrAdapterNotFound
	}

	r.adapters[adapter.ID] = adapter
	logger.Info("Adapter updated", logger.String("id", adapter.ID))

	return nil
}

// HasCapability checks if an adapter has a specific capability
func (r *Registry) HasCapability(id, capability string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[id]
	if !exists {
		return false
	}

	for _, cap := range adapter.Capabilities {
		if cap == capability {
			return true
		}
	}

	return false
}

// GetManifest gets adapter manifest as JSON
func (r *Registry) GetManifest(id string) (string, error) {
	adapter, err := r.Get(id)
	if err != nil {
		return "", err
	}

	manifest := v3.AdapterManifest{
		ID:              adapter.ID,
		PlatformID:      adapter.PlatformID,
		Name:            adapter.Name,
		Version:         adapter.Version,
		RuntimeType:     adapter.RuntimeType,
		ProtocolVersion: adapter.ProtocolVersion,
		Capabilities:    adapter.Capabilities,
		ConfigSchema:    adapter.ConfigSchema,
		I18n:            adapter.I18n,
		Operations:      adapter.Operations,
		Security:        adapter.Security,
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Registry errors
var (
	ErrInvalidAdapterID     = &AdapterError{Code: "INVALID_ADAPTER_ID", Message: "invalid adapter ID"}
	ErrAdapterNotFound      = &AdapterError{Code: "ADAPTER_NOT_FOUND", Message: "adapter not found"}
	ErrAdapterAlreadyExists = &AdapterError{Code: "ADAPTER_ALREADY_EXISTS", Message: "adapter already exists"}
)

// AdapterError represents an adapter error
type AdapterError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AdapterError) Error() string {
	return e.Message
}

package adapter

import (
	"context"
	"errors"
	"sync"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// InstanceStatus represents adapter instance status
type InstanceStatus string

const (
	InstanceStatusStopped      InstanceStatus = "stopped"
	InstanceStatusStarting     InstanceStatus = "starting"
	InstanceStatusRunning      InstanceStatus = "running"
	InstanceStatusStopping     InstanceStatus = "stopping"
	InstanceStatusError        InstanceStatus = "error"
	InstanceStatusDisconnected InstanceStatus = "disconnected"
)

// Instance represents an adapter instance
type Instance struct {
	ID            string                 `json:"id"`
	AdapterID     string                 `json:"adapter_id"`
	PlatformID    string                 `json:"platform_id"`
	Name          string                 `json:"name"`
	Status        InstanceStatus         `json:"status"`
	Config        map[string]interface{} `json:"config"`
	Credentials   map[string]interface{} `json:"credentials"`
	ConnectedAt   *time.Time             `json:"connected_at,omitempty"`
	LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
	MessageCount  int64                  `json:"message_count"`
	ErrorCount    int64                  `json:"error_count"`
	LastError     string                 `json:"last_error,omitempty"`
}

// Manager manages adapter instances
type Manager struct {
	registry  *Registry
	bridge    *BridgeManager
	instances map[string]*Instance
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a new adapter manager
func NewManager(registry *Registry, bridge *BridgeManager) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		registry:  registry,
		bridge:    bridge,
		instances: make(map[string]*Instance),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// CreateInstance creates a new adapter instance
func (m *Manager) CreateInstance(adapterID, name string, config, credentials map[string]interface{}) (*Instance, error) {
	adapter, err := m.registry.Get(adapterID)
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		ID:          v3.GenerateEventID(),
		AdapterID:   adapterID,
		PlatformID:  adapter.PlatformID,
		Name:        name,
		Status:      InstanceStatusStopped,
		Config:      config,
		Credentials: credentials,
	}

	m.mu.Lock()
	m.instances[instance.ID] = instance
	m.mu.Unlock()

	logger.Info("Adapter instance created",
		logger.String("instance_id", instance.ID),
		logger.String("adapter_id", adapterID),
		logger.String("name", name))

	return instance, nil
}

// StartInstance starts an adapter instance
func (m *Manager) StartInstance(instanceID string) error {
	m.mu.Lock()
	instance, exists := m.instances[instanceID]
	if !exists {
		m.mu.Unlock()
		return ErrInstanceNotFound
	}

	if instance.Status == InstanceStatusRunning {
		m.mu.Unlock()
		return ErrInstanceAlreadyRunning
	}

	instance.Status = InstanceStatusStarting
	m.mu.Unlock()

	// Get bridge client for platform
	bridgeClient, ok := m.bridge.GetClient(instance.PlatformID)
	if !ok {
		m.mu.Lock()
		instance.Status = InstanceStatusError
		instance.LastError = "no bridge client for platform"
		m.mu.Unlock()
		return errors.New("no bridge client for platform")
	}

	// Register with Python adapter
	req := &RegisterRequest{
		Platform:    instance.PlatformID,
		AccountID:   instanceID,
		Credentials: instance.Credentials,
		Config:      instance.Config,
	}

	resp, err := bridgeClient.Register(req)
	if err != nil {
		m.mu.Lock()
		instance.Status = InstanceStatusError
		instance.LastError = err.Error()
		m.mu.Unlock()
		return err
	}

	now := time.Now()
	m.mu.Lock()
	instance.Status = InstanceStatusRunning
	instance.ConnectedAt = &now
	instance.LastHeartbeat = &now
	m.mu.Unlock()

	logger.Info("Adapter instance started",
		logger.String("instance_id", instanceID),
		logger.String("adapter_id", resp.AdapterID))

	return nil
}

// StopInstance stops an adapter instance
func (m *Manager) StopInstance(instanceID string) error {
	m.mu.Lock()
	instance, exists := m.instances[instanceID]
	if !exists {
		m.mu.Unlock()
		return ErrInstanceNotFound
	}

	if instance.Status != InstanceStatusRunning {
		m.mu.Unlock()
		return ErrInstanceNotRunning
	}

	instance.Status = InstanceStatusStopping
	m.mu.Unlock()

	// Get bridge client for platform
	bridgeClient, ok := m.bridge.GetClient(instance.PlatformID)
	if ok {
		// Unregister from Python adapter
		if err := bridgeClient.Unregister(instanceID); err != nil {
			logger.Warn("Failed to unregister adapter",
				logger.String("instance_id", instanceID),
				logger.String("error", err.Error()))
		}
	}

	m.mu.Lock()
	instance.Status = InstanceStatusStopped
	instance.ConnectedAt = nil
	m.mu.Unlock()

	logger.Info("Adapter instance stopped", logger.String("instance_id", instanceID))

	return nil
}

// DeleteInstance deletes an adapter instance
func (m *Manager) DeleteInstance(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return ErrInstanceNotFound
	}

	if instance.Status == InstanceStatusRunning {
		return ErrInstanceRunning
	}

	delete(m.instances, instanceID)
	logger.Info("Adapter instance deleted", logger.String("instance_id", instanceID))

	return nil
}

// GetInstance gets an adapter instance
func (m *Manager) GetInstance(instanceID string) (*Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return nil, ErrInstanceNotFound
	}

	return instance, nil
}

// GetAllInstances gets all adapter instances
func (m *Manager) GetAllInstances() []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Instance, 0, len(m.instances))
	for _, instance := range m.instances {
		result = append(result, instance)
	}

	return result
}

// GetInstancesByPlatform gets instances by platform ID
func (m *Manager) GetInstancesByPlatform(platformID string) []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Instance, 0)
	for _, instance := range m.instances {
		if instance.PlatformID == platformID {
			result = append(result, instance)
		}
	}

	return result
}

// UpdateHeartbeat updates instance heartbeat
func (m *Manager) UpdateHeartbeat(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return ErrInstanceNotFound
	}

	now := time.Now()
	instance.LastHeartbeat = &now

	return nil
}

// IncrementMessageCount increments message count
func (m *Manager) IncrementMessageCount(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return ErrInstanceNotFound
	}

	instance.MessageCount++

	return nil
}

// SendMessage sends a message through an instance
func (m *Manager) SendMessage(instanceID, targetID string, messageChain *v3.MessageChain) error {
	m.mu.RLock()
	instance, exists := m.instances[instanceID]
	if !exists {
		m.mu.RUnlock()
		return ErrInstanceNotFound
	}

	if instance.Status != InstanceStatusRunning {
		m.mu.RUnlock()
		return ErrInstanceNotRunning
	}
	m.mu.RUnlock()

	// Get bridge client for platform
	bridgeClient, ok := m.bridge.GetClient(instance.PlatformID)
	if !ok {
		return errors.New("no bridge client for platform")
	}

	req := &SendMessageRequest{
		AdapterID:    instanceID,
		TargetID:     targetID,
		MessageChain: messageChain,
	}

	return bridgeClient.SendMessage(req)
}

// Close closes the manager
func (m *Manager) Close() error {
	m.cancel()

	// Stop all running instances
	m.mu.RLock()
	instances := make([]*Instance, 0)
	for _, instance := range m.instances {
		if instance.Status == InstanceStatusRunning {
			instances = append(instances, instance)
		}
	}
	m.mu.RUnlock()

	for _, instance := range instances {
		if err := m.StopInstance(instance.ID); err != nil {
			logger.Warn("Failed to stop instance during shutdown",
				logger.String("instance_id", instance.ID),
				logger.String("error", err.Error()))
		}
	}

	return nil
}

// Manager errors
var (
	ErrInstanceNotFound       = errors.New("instance not found")
	ErrInstanceAlreadyRunning = errors.New("instance already running")
	ErrInstanceNotRunning     = errors.New("instance not running")
	ErrInstanceRunning        = errors.New("instance is running")
)

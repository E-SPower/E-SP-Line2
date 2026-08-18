package service

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// accessKeyCharset is the character set used for generated access keys:
// digits plus upper/lowercase letters.
const accessKeyCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenerateAccessKey generates a random 16-character access key composed of
// digits and upper/lowercase letters. It is used when a new adapter is created
// without an explicit key, so no insecure default key is ever used.
func GenerateAccessKey() string {
	b := make([]byte, 16)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(accessKeyCharset))))
		if err != nil {
			// Extremely unlikely; fall back to a deterministic char so the
			// key still has the right length.
			b[i] = accessKeyCharset[i%len(accessKeyCharset)]
			continue
		}
		b[i] = accessKeyCharset[n.Int64()]
	}
	return string(b)
}

// AdapterGatewayService manages adapter (接入器) entities and their
// WebSocket connections. An adapter is a configurable WebSocket endpoint
// that external systems use to exchange e-commerce messages with E-SP-Line2.
//
// It supports two modes:
//   - "server": E-SP-Line2 listens; external systems connect with the key.
//   - "client": E-SP-Line2 actively connects to an external WS URL.
type AdapterGatewayService struct {
	adapterRepo    *repository.AdapterGatewayRepository
	connectionRepo *repository.AdapterConnectionRepository
	messageService *MessageService
	onChange       func()
}

// NewAdapterGatewayService creates a new adapter gateway service.
func NewAdapterGatewayService(
	adapterRepo *repository.AdapterGatewayRepository,
	connectionRepo *repository.AdapterConnectionRepository,
	messageService *MessageService,
) *AdapterGatewayService {
	return &AdapterGatewayService{
		adapterRepo:    adapterRepo,
		connectionRepo: connectionRepo,
		messageService: messageService,
	}
}

// SetOnChangeCallback sets a callback that is invoked whenever an adapter is
// created, updated or deleted. The server uses this to reload client-mode
// outbound connections and re-register dynamic listen paths.
func (s *AdapterGatewayService) SetOnChangeCallback(fn func()) {
	s.onChange = fn
}

// CreateGatewayAdapterRequest represents a create adapter (接入器) request.
type CreateGatewayAdapterRequest struct {
	Name       string `json:"name" binding:"required"`
	Mode       string `json:"mode"`        // server / client
	ListenPath string `json:"listen_path"` // server mode
	WSURL      string `json:"ws_url"`      // client mode
	Key        string `json:"key"`         // user-defined, editable access key
	Platform   string `json:"platform"`    // taobao / xianyu / "" (all)
	Scope      string `json:"scope"`       // read / write / read+write
	Enabled    *bool  `json:"enabled"`
}

// UpdateGatewayAdapterRequest represents an update adapter (接入器) request.
type UpdateGatewayAdapterRequest struct {
	Name       *string `json:"name"`
	Mode       *string `json:"mode"`
	ListenPath *string `json:"listen_path"`
	WSURL      *string `json:"ws_url"`
	Key        *string `json:"key"`
	Platform   *string `json:"platform"`
	Scope      *string `json:"scope"`
	Enabled    *bool   `json:"enabled"`
	Status     *string `json:"status"`
}

// Create creates a new adapter.
func (s *AdapterGatewayService) Create(req *CreateGatewayAdapterRequest, createdBy string) (*models.Adapter, error) {
	mode := req.Mode
	if mode == "" {
		mode = "server"
	}
	if mode != "server" && mode != "client" {
		return nil, errors.New("invalid mode: must be server or client")
	}

	scope := req.Scope
	if scope == "" {
		scope = "read+write"
	}
	if scope != "read" && scope != "write" && scope != "read+write" {
		return nil, errors.New("invalid scope: must be read, write or read+write")
	}

	// Auto-generate a random 16-character access key when none is provided,
	// so no insecure default key is ever used.
	key := req.Key
	if key == "" {
		key = GenerateAccessKey()
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	adapter := &models.Adapter{
		ID:         models.GenerateID(),
		Name:       req.Name,
		Mode:       mode,
		ListenPath: req.ListenPath,
		WSURL:      req.WSURL,
		Key:        key,
		Platform:   req.Platform,
		Scope:      scope,
		Status:     "active",
		Enabled:    enabled,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CreatedBy:  createdBy,
	}

	if err := s.adapterRepo.Create(adapter); err != nil {
		return nil, err
	}

	logger.Info("Adapter created",
		logger.String("adapter_id", adapter.ID),
		logger.String("name", adapter.Name),
		logger.String("mode", adapter.Mode),
		logger.String("created_by", createdBy))

	if s.onChange != nil {
		s.onChange()
	}

	return adapter, nil
}

// List lists all adapters.
func (s *AdapterGatewayService) List(limit, offset int) ([]models.Adapter, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.adapterRepo.List(limit, offset)
}

// GetByID gets an adapter by ID.
func (s *AdapterGatewayService) GetByID(id string) (*models.Adapter, error) {
	return s.adapterRepo.FindByID(id)
}

// Update updates an adapter.
func (s *AdapterGatewayService) Update(id string, req *UpdateGatewayAdapterRequest) (*models.Adapter, error) {
	adapter, err := s.adapterRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		adapter.Name = *req.Name
	}
	if req.Mode != nil {
		if *req.Mode != "server" && *req.Mode != "client" {
			return nil, errors.New("invalid mode: must be server or client")
		}
		adapter.Mode = *req.Mode
	}
	if req.ListenPath != nil {
		adapter.ListenPath = *req.ListenPath
	}
	if req.WSURL != nil {
		adapter.WSURL = *req.WSURL
	}
	if req.Key != nil {
		if *req.Key == "" {
			return nil, errors.New("key cannot be empty")
		}
		adapter.Key = *req.Key
	}
	if req.Platform != nil {
		adapter.Platform = *req.Platform
	}
	if req.Scope != nil {
		if *req.Scope != "read" && *req.Scope != "write" && *req.Scope != "read+write" {
			return nil, errors.New("invalid scope: must be read, write or read+write")
		}
		adapter.Scope = *req.Scope
	}
	if req.Enabled != nil {
		adapter.Enabled = *req.Enabled
	}
	if req.Status != nil {
		adapter.Status = *req.Status
	}
	adapter.UpdatedAt = time.Now()

	if err := s.adapterRepo.Update(adapter); err != nil {
		return nil, err
	}

	if s.onChange != nil {
		s.onChange()
	}

	return adapter, nil
}

// Delete deletes an adapter.
func (s *AdapterGatewayService) Delete(id string) error {
	err := s.adapterRepo.Delete(id)
	if err != nil {
		return err
	}

	if s.onChange != nil {
		s.onChange()
	}

	return nil
}

// ValidateKey validates an access key and returns the matching adapter.
// It checks that the adapter is enabled and active.
func (s *AdapterGatewayService) ValidateKey(key string) (*models.Adapter, error) {
	if key == "" {
		return nil, errors.New("key is required")
	}
	adapter, err := s.adapterRepo.FindByKey(key)
	if err != nil {
		return nil, errors.New("invalid key")
	}
	if !adapter.Enabled {
		return nil, errors.New("adapter is disabled")
	}
	if adapter.Status != "active" {
		return nil, errors.New("adapter is not active")
	}
	now := time.Now()
	_ = s.adapterRepo.UpdateLastConnected(adapter.ID, now)
	return adapter, nil
}

// ValidateAdapterIDAndKey validates an adapter ID and its access key.
// The adapter must be enabled, active, and the key must match.
func (s *AdapterGatewayService) ValidateAdapterIDAndKey(adapterID, key string) (*models.Adapter, error) {
	if adapterID == "" {
		return nil, errors.New("adapter id is required")
	}
	adapter, err := s.adapterRepo.FindByID(adapterID)
	if err != nil {
		return nil, errors.New("adapter not found")
	}
	if !adapter.Enabled {
		return nil, errors.New("adapter is disabled")
	}
	if adapter.Status != "active" {
		return nil, errors.New("adapter is not active")
	}
	if adapter.Key != key {
		return nil, errors.New("invalid key")
	}
	now := time.Now()
	_ = s.adapterRepo.UpdateLastConnected(adapter.ID, now)
	return adapter, nil
}

// CreateConnection records a new adapter connection.
func (s *AdapterGatewayService) CreateConnection(adapter *models.Adapter, remoteAddr string) (*models.AdapterConnection, error) {
	now := time.Now()
	conn := &models.AdapterConnection{
		ID:            models.GenerateID(),
		AdapterID:     adapter.ID,
		AdapterName:   adapter.Name,
		Mode:          adapter.Mode,
		Platform:      adapter.Platform,
		RemoteAddr:    remoteAddr,
		Status:        "connected",
		ConnectedAt:   now,
		LastHeartbeat: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.connectionRepo.Create(conn); err != nil {
		return nil, err
	}
	return conn, nil
}

// MarkConnectionDisconnected marks a connection as disconnected.
func (s *AdapterGatewayService) MarkConnectionDisconnected(id string) error {
	now := time.Now()
	return s.connectionRepo.MarkDisconnected(id, now)
}

// TouchConnection updates the heartbeat of a connection. It uses a single
// UPDATE statement (no read-then-write) so it stays cheap on the hot path.
func (s *AdapterGatewayService) TouchConnection(id string) error {
	return s.connectionRepo.TouchHeartbeat(id)
}

// IncrementConnectionMessageCount increments the message count of a connection.
func (s *AdapterGatewayService) IncrementConnectionMessageCount(id string) error {
	return s.connectionRepo.IncrementMessageCount(id)
}

// ListConnections lists all connection records.
func (s *AdapterGatewayService) ListConnections(limit, offset int) ([]models.AdapterConnection, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.connectionRepo.List(limit, offset)
}

// ListConnectionsByAdapter lists connection records for an adapter.
func (s *AdapterGatewayService) ListConnectionsByAdapter(adapterID string, limit, offset int) ([]models.AdapterConnection, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.connectionRepo.ListByAdapterID(adapterID, limit, offset)
}

// PersistInboundMessage persists an inbound message coming from a bridge
// (internal) through the message service, preserving the full raw payload.
func (s *AdapterGatewayService) PersistInboundMessage(instanceID string, payload map[string]interface{}) string {
	if s.messageService == nil {
		return ""
	}
	req := &CreateMessageRequest{
		PlatformID:     getStringField(payload, "platform_id"),
		InstanceID:     instanceID,
		ConversationID: getStringField(payload, "conversation_id"),
		SenderID:       getStringField(payload, "sender_id"),
		SenderName:     getStringField(payload, "sender_name"),
		MessageType:    getStringField(payload, "message_type"),
		MessageContent: getStringField(payload, "message_content"),
		RawMessage:     payload,
		IdempotencyKey: getStringField(payload, "idempotency_key"),
	}
	event, err := s.messageService.Create(req)
	if err != nil {
		logger.Warn("Failed to persist inbound message",
			logger.String("error", err.Error()))
		return ""
	}
	return event.ID
}

// getStringField extracts a string field from a payload map.
func getStringField(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

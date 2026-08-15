package adapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// BridgeClient handles communication with Python adapters
type BridgeClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	signer     *v3.RequestSigner
}

// NewBridgeClient creates a new bridge client
func NewBridgeClient(baseURL, apiKey, apiSecret string) *BridgeClient {
	return &BridgeClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		signer: v3.NewRequestSigner(apiKey, apiSecret, v3.AlgorithmHMACSHA256),
	}
}

// AdapterStatus represents adapter status
type AdapterStatus struct {
	ID           string `json:"id"`
	Platform     string `json:"platform"`
	Status       string `json:"status"` // running, stopped, error
	ConnectedAt  int64  `json:"connected_at,omitempty"`
	MessageCount int64  `json:"message_count,omitempty"`
}

// RegisterRequest represents adapter registration request
type RegisterRequest struct {
	Platform    string                 `json:"platform"`
	AccountID   string                 `json:"account_id"`
	Credentials map[string]interface{} `json:"credentials"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// RegisterResponse represents adapter registration response
type RegisterResponse struct {
	AdapterID string `json:"adapter_id"`
	Status    string `json:"status"`
}

// SendMessageRequest represents send message request
type SendMessageRequest struct {
	AdapterID    string      `json:"adapter_id"`
	TargetID     string      `json:"target_id"`
	MessageChain interface{} `json:"message_chain"`
}

// Register registers a new adapter instance
func (c *BridgeClient) Register(req *RegisterRequest) (*RegisterResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	headers, err := c.signer.SignRequest("POST", "/api/v1/adapters/register", body, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/adapters/register", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed: %s", string(bodyBytes))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Unregister unregisters an adapter instance
func (c *BridgeClient) Unregister(adapterID string) error {
	path := fmt.Sprintf("/api/v1/adapters/%s", adapterID)

	headers, err := c.signer.SignRequest("DELETE", path, nil, time.Now().Unix())
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unregistration failed: %s", string(bodyBytes))
	}

	return nil
}

// GetStatus gets adapter status
func (c *BridgeClient) GetStatus(adapterID string) (*AdapterStatus, error) {
	path := fmt.Sprintf("/api/v1/adapters/%s/status", adapterID)

	headers, err := c.signer.SignRequest("GET", path, nil, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get status failed: %s", string(bodyBytes))
	}

	var status AdapterStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SendMessage sends a message through the adapter
func (c *BridgeClient) SendMessage(req *SendMessageRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	path := "/api/v1/messages/send"
	headers, err := c.signer.SignRequest("POST", path, body, time.Now().Unix())
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send message failed: %s", string(bodyBytes))
	}

	return nil
}

// HealthCheck checks adapter bridge health
func (c *BridgeClient) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("bridge health check failed")
	}

	return nil
}

// BridgeManager manages multiple bridge clients
type BridgeManager struct {
	clients map[string]*BridgeClient
	mu      sync.RWMutex
}

// NewBridgeManager creates a new bridge manager
func NewBridgeManager() *BridgeManager {
	return &BridgeManager{
		clients: make(map[string]*BridgeClient),
	}
}

// AddClient adds a bridge client
func (m *BridgeManager) AddClient(platform string, client *BridgeClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[platform] = client
	logger.Info("Bridge client added", logger.String("platform", platform))
}

// GetClient gets a bridge client by platform
func (m *BridgeManager) GetClient(platform string) (*BridgeClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[platform]
	return client, ok
}

// RemoveClient removes a bridge client
func (m *BridgeManager) RemoveClient(platform string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, platform)
	logger.Info("Bridge client removed", logger.String("platform", platform))
}

// GetAllClients returns all bridge clients
func (m *BridgeManager) GetAllClients() map[string]*BridgeClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*BridgeClient)
	for k, v := range m.clients {
		result[k] = v
	}
	return result
}

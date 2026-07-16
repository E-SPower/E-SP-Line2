package v3

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SigningAlgorithm represents the signing algorithm
type SigningAlgorithm string

const (
	AlgorithmHMACSHA256 SigningAlgorithm = "HMAC-SHA256"
	AlgorithmHMACMD5    SigningAlgorithm = "HMAC-MD5"
	AlgorithmMD5        SigningAlgorithm = "MD5"
)

// Signer handles message signing and verification
type Signer struct {
	algorithm SigningAlgorithm
	secret    string
}

// NewSigner creates a new signer
func NewSigner(algorithm SigningAlgorithm, secret string) *Signer {
	return &Signer{
		algorithm: algorithm,
		secret:    secret,
	}
}

// Sign signs the data and returns the signature
func (s *Signer) Sign(data []byte) (string, error) {
	switch s.algorithm {
	case AlgorithmHMACSHA256:
		return s.signHMACSHA256(data), nil
	case AlgorithmHMACMD5:
		return s.signHMACMD5(data), nil
	case AlgorithmMD5:
		return s.signMD5(data), nil
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", s.algorithm)
	}
}

// Verify verifies the signature
func (s *Signer) Verify(data []byte, signature string) bool {
	expectedSignature, err := s.Sign(data)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// signHMACSHA256 signs data using HMAC-SHA256
func (s *Signer) signHMACSHA256(data []byte) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// signHMACMD5 signs data using HMAC-MD5
func (s *Signer) signHMACMD5(data []byte) string {
	mac := hmac.New(md5.New, []byte(s.secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// signMD5 signs data using MD5
func (s *Signer) signMD5(data []byte) string {
	hash := md5.Sum(append(data, []byte(s.secret)...))
	return hex.EncodeToString(hash[:])
}

// SignEnvelope signs a message envelope
func (s *Signer) SignEnvelope(envelope *MessageEnvelope) error {
	// Create a copy without signature
	envelopeCopy := *envelope
	envelopeCopy.Signature = ""

	data, err := json.Marshal(envelopeCopy)
	if err != nil {
		return err
	}

	signature, err := s.Sign(data)
	if err != nil {
		return err
	}

	envelope.Signature = signature
	return nil
}

// VerifyEnvelope verifies a message envelope signature
func (s *Signer) VerifyEnvelope(envelope *MessageEnvelope) bool {
	if envelope.Signature == "" {
		return false
	}

	// Create a copy without signature
	envelopeCopy := *envelope
	envelopeCopy.Signature = ""

	data, err := json.Marshal(envelopeCopy)
	if err != nil {
		return false
	}

	return s.Verify(data, envelope.Signature)
}

// GenerateAPIKey generates an API key
func GenerateAPIKey() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return "espl_" + hex.EncodeToString(bytes)
}

// GenerateAPISecret generates an API secret
func GenerateAPISecret() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return "secret_" + hex.EncodeToString(bytes)
}

// RequestSigner signs HTTP requests
type RequestSigner struct {
	apiKey    string
	apiSecret string
	algorithm SigningAlgorithm
}

// NewRequestSigner creates a new request signer
func NewRequestSigner(apiKey, apiSecret string, algorithm SigningAlgorithm) *RequestSigner {
	return &RequestSigner{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		algorithm: algorithm,
	}
}

// SignRequest signs an HTTP request
func (rs *RequestSigner) SignRequest(method, path string, body []byte, timestamp int64) (map[string]string, error) {
	headers := make(map[string]string)

	// Build string to sign
	stringToSign := fmt.Sprintf("%s\n%s\n%d\n%s",
		method,
		path,
		timestamp,
		hashBytes(body),
	)

	signer := NewSigner(rs.algorithm, rs.apiSecret)
	signature, err := signer.Sign([]byte(stringToSign))
	if err != nil {
		return nil, err
	}

	headers["X-API-Key"] = rs.apiKey
	headers["X-Timestamp"] = fmt.Sprintf("%d", timestamp)
	headers["X-Signature"] = signature
	headers["X-Algorithm"] = string(rs.algorithm)

	return headers, nil
}

// VerifyRequest verifies an HTTP request signature
func (rs *RequestSigner) VerifyRequest(method, path string, body []byte, headers map[string]string) error {
	apiKey := headers["X-API-Key"]
	if apiKey != rs.apiKey {
		return errors.New("invalid API key")
	}

	timestampStr := headers["X-Timestamp"]
	var timestamp int64
	fmt.Sscanf(timestampStr, "%d", &timestamp)

	// Check timestamp validity (within 5 minutes)
	now := time.Now().Unix()
	if abs(now-timestamp) > 300 {
		return errors.New("request expired")
	}

	signature := headers["X-Signature"]
	if signature == "" {
		return errors.New("missing signature")
	}

	// Build string to sign
	stringToSign := fmt.Sprintf("%s\n%s\n%d\n%s",
		method,
		path,
		timestamp,
		hashBytes(body),
	)

	signer := NewSigner(rs.algorithm, rs.apiSecret)
	if !signer.Verify([]byte(stringToSign), signature) {
		return errors.New("invalid signature")
	}

	return nil
}

// hashBytes returns MD5 hash of bytes
func hashBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// abs returns absolute value
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// ParseAuthorizationHeader parses Authorization header
func ParseAuthorizationHeader(auth string) (string, string, error) {
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid authorization header")
	}

	scheme := strings.ToLower(parts[0])
	if scheme != "bearer" {
		return "", "", fmt.Errorf("unsupported scheme: %s", scheme)
	}

	return "Bearer", parts[1], nil
}

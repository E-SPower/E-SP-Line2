package message

import (
	"context"
	"sync"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// RouteRule represents a routing rule
type RouteRule struct {
	ID         string
	PlatformID string
	Priority   int
	Conditions map[string]interface{}
	TargetType string // "app", "webhook", "queue"
	TargetID   string
	Enabled    bool
}

// Router handles message routing
type Router struct {
	rules  []*RouteRule
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRouter creates a new message router
func NewRouter() *Router {
	ctx, cancel := context.WithCancel(context.Background())
	return &Router{
		rules:  make([]*RouteRule, 0),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddRule adds a routing rule
func (r *Router) AddRule(rule *RouteRule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rules = append(r.rules, rule)
	logger.Info("Route rule added",
		logger.String("rule_id", rule.ID),
		logger.String("platform", rule.PlatformID),
		logger.Int("priority", rule.Priority))
}

// RemoveRule removes a routing rule
func (r *Router) RemoveRule(ruleID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, rule := range r.rules {
		if rule.ID == ruleID {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			logger.Info("Route rule removed", logger.String("rule_id", ruleID))
			break
		}
	}
}

// GetRules gets all routing rules
func (r *Router) GetRules() []*RouteRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*RouteRule, len(r.rules))
	copy(result, r.rules)
	return result
}

// Route routes a message to target based on rules
func (r *Router) Route(envelope *v3.MessageEnvelope) ([]*RouteTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	targets := make([]*RouteTarget, 0)

	// Sort rules by priority (higher priority first)
	sortedRules := make([]*RouteRule, len(r.rules))
	copy(sortedRules, r.rules)

	// Simple bubble sort for priority
	for i := 0; i < len(sortedRules); i++ {
		for j := i + 1; j < len(sortedRules); j++ {
			if sortedRules[i].Priority < sortedRules[j].Priority {
				sortedRules[i], sortedRules[j] = sortedRules[j], sortedRules[i]
			}
		}
	}

	// Match rules
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		if rule.PlatformID != "" && rule.PlatformID != envelope.Platform {
			continue
		}

		// Check conditions
		if r.matchConditions(rule.Conditions, envelope) {
			targets = append(targets, &RouteTarget{
				Type:     rule.TargetType,
				ID:       rule.TargetID,
				RuleID:   rule.ID,
				Priority: rule.Priority,
			})
		}
	}

	return targets, nil
}

// matchConditions checks if envelope matches conditions
func (r *Router) matchConditions(conditions map[string]interface{}, envelope *v3.MessageEnvelope) bool {
	if len(conditions) == 0 {
		return true
	}

	// Check event type
	if eventType, ok := conditions["event_type"]; ok {
		if string(envelope.EventType) != eventType.(string) {
			return false
		}
	}

	// Check adapter ID
	if adapterID, ok := conditions["adapter_id"]; ok {
		if envelope.AdapterID != adapterID.(string) {
			return false
		}
	}

	return true
}

// RouteTarget represents a routing target
type RouteTarget struct {
	Type     string
	ID       string
	RuleID   string
	Priority int
}

// Close closes the router
func (r *Router) Close() error {
	r.cancel()
	return nil
}

// RouteResult represents routing result
type RouteResult struct {
	Envelope  *v3.MessageEnvelope
	Targets   []*RouteTarget
	Timestamp time.Time
}

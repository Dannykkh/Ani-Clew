package router

import (
	"fmt"
	"log"
	"sync"

	"github.com/aniclew/aniclew/internal/providers"
	"github.com/aniclew/aniclew/internal/types"
)

type Router struct {
	mu            sync.RWMutex
	config        RouterConfig
	providerCache map[string]types.Provider
	providerCfgs  map[string]*types.ProviderConfig
	costs         map[string]*CostEntry // key: "provider/model"
}

func New(cfg *RouterConfig, provCfgs map[string]*types.ProviderConfig) *Router {
	if cfg == nil {
		def := DefaultConfig()
		cfg = &def
	}
	if provCfgs == nil {
		provCfgs = map[string]*types.ProviderConfig{}
	}
	return &Router{
		config:        *cfg,
		providerCache: map[string]types.Provider{},
		providerCfgs:  provCfgs,
		costs:         map[string]*CostEntry{},
	}
}

// Route decides which provider+model to use.
func (r *Router) Route(req *types.MessagesRequest) RouteDecision {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.config.Enabled {
		return RouteDecision{Role: RoleDefault, Provider: "default", Model: req.Model, Reason: "Router disabled"}
	}

	role, reason := Classify(req, &r.config)
	rule := r.findRule(role)
	if rule == nil {
		rule = r.findRule(RoleDefault)
	}
	if rule == nil {
		return RouteDecision{Role: role, Provider: "default", Model: req.Model, Reason: reason + " → no rule"}
	}

	log.Printf("Route: [%s] → %s/%s (%s)", role, rule.Provider, rule.Model, reason)
	return RouteDecision{
		Role:     role,
		Provider: rule.Provider,
		Model:    rule.Model,
		Reason:   reason,
	}
}

// GetProvider returns a provider instance for the given decision.
func (r *Router) GetProvider(decision RouteDecision) (types.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.providerCache[decision.Provider]; ok {
		return p, nil
	}

	cfg := r.providerCfgs[decision.Provider]
	if cfg == nil {
		cfg = &types.ProviderConfig{}
	}

	p, err := providers.Create(decision.Provider, cfg)
	if err != nil {
		return nil, err
	}
	r.providerCache[decision.Provider] = p
	return p, nil
}

// GetFallback returns the fallback target for a role, if any.
func (r *Router) GetFallback(role RoleID) *Target {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.config.AutoEscalate {
		return nil
	}
	rule := r.findRule(role)
	if rule == nil {
		return nil
	}
	return rule.Fallback
}

// SetRule updates a routing rule at runtime.
func (r *Router) SetRule(role RoleID, provider, model string, fallback *Target) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, rule := range r.config.Rules {
		if rule.Role == role {
			r.config.Rules[i] = RouteRule{Role: role, Provider: provider, Model: model, Fallback: fallback}
			log.Printf("Rule updated: [%s] → %s/%s", role, provider, model)
			return
		}
	}
	r.config.Rules = append(r.config.Rules, RouteRule{Role: role, Provider: provider, Model: model, Fallback: fallback})
	log.Printf("Rule added: [%s] → %s/%s", role, provider, model)
}

// TrackUsage records cost for a request.
func (r *Router) TrackUsage(provider, model string, outputTokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s/%s", provider, model)
	entry, ok := r.costs[key]
	if !ok {
		entry = &CostEntry{Provider: provider, Model: model}
		r.costs[key] = entry
	}
	entry.Requests++
	entry.Tokens += outputTokens
	entry.Cost += estimateCost(provider, model, outputTokens)
}

// GetCostSummary returns all cost entries sorted by cost descending.
func (r *Router) GetCostSummary() []CostEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]CostEntry, 0, len(r.costs))
	for _, e := range r.costs {
		result = append(result, *e)
	}
	return result
}

// GetTotalCost returns total cost across all models.
func (r *Router) GetTotalCost() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var total float64
	for _, e := range r.costs {
		total += e.Cost
	}
	return total
}

// GetConfig returns current config.
func (r *Router) GetConfig() RouterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

func (r *Router) findRule(role RoleID) *RouteRule {
	for i := range r.config.Rules {
		if r.config.Rules[i].Role == role {
			return &r.config.Rules[i]
		}
	}
	return nil
}

// Price per 1M output tokens
var prices = map[string]float64{
	"ollama/":                        0,
	"openai/gpt-4o":                  15,
	"openai/gpt-4o-mini":             0.6,
	"openai/o3":                      40,
	"anthropic/claude-opus-4-20250514":   75,
	"anthropic/claude-sonnet-4-20250514": 15,
	"anthropic/claude-haiku-4-20250506":  5,
	"gemini/gemini-2.5-pro-preview-05-06":  10,
	"gemini/gemini-2.5-flash-preview-05-20": 1.5,
	"groq/":                          0.5,
	"zai/grok-3":                     10,
}

func estimateCost(provider, model string, outputTokens int) float64 {
	key := fmt.Sprintf("%s/%s", provider, model)
	if p, ok := prices[key]; ok {
		return float64(outputTokens) / 1_000_000 * p
	}
	// Try prefix
	prefix := provider + "/"
	if p, ok := prices[prefix]; ok {
		return float64(outputTokens) / 1_000_000 * p
	}
	return float64(outputTokens) / 1_000_000 * 5 // default
}

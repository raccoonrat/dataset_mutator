package policy

import (
	"encoding/json"
	"os"
	"regexp"
	"sync"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
)

const ActionDegradeToTemplate = "DEGRADE_TO_TEMPLATE"

// Store is the gateway-side read model for async-published rules.
type Store interface {
	MatchPrompt(obfuscatedPrompt string) *contracts.PolicyRule
	Upsert(rule contracts.PolicyRule) error
}

// MemoryStore is the MVP implementation (Milestone 3 can back this with Redis).
type MemoryStore struct {
	mu    sync.RWMutex
	rules []compiledRule
}

type compiledRule struct {
	raw contracts.PolicyRule
	re  *regexp.Regexp
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// LoadFromFile reads a JSON array of PolicyRule (optional bootstrap for demos/tests).
func (m *MemoryStore) LoadFromFile(path string) error {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var rules []contracts.PolicyRule
	if err := json.Unmarshal(b, &rules); err != nil {
		return err
	}
	for _, r := range rules {
		if err := m.Upsert(r); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryStore) Upsert(rule contracts.PolicyRule) error {
	re, err := regexp.Compile(rule.TriggerPattern)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, cr := range m.rules {
		if cr.raw.RuleID == rule.RuleID {
			m.rules[i] = compiledRule{raw: rule, re: re}
			return nil
		}
	}
	m.rules = append(m.rules, compiledRule{raw: rule, re: re})
	return nil
}

func (m *MemoryStore) MatchPrompt(obfuscatedPrompt string) *contracts.PolicyRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cr := range m.rules {
		if cr.re.MatchString(obfuscatedPrompt) {
			r := cr.raw
			return &r
		}
	}
	return nil
}

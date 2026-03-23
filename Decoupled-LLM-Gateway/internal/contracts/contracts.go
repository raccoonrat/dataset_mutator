// Package contracts defines the JSON boundaries between the sync gateway and the async loop.
package contracts

// GatewayLogEvent is emitted by the gateway after each handled request (Gateway → Async loop).
type GatewayLogEvent struct {
	TraceID              string `json:"trace_id"`
	Timestamp            int64  `json:"timestamp"`
	InjectedDecoyID      string `json:"injected_decoy_id"`
	RawUserPrompt        string `json:"raw_user_prompt"`
	ObfuscatedPrompt     string `json:"obfuscated_prompt"`
	LLMResponse          string `json:"llm_response"`
	DegradationTriggered bool   `json:"degradation_triggered"`
	// Optional paper-eval fields (set when clients send experiment headers).
	ExperimentRunID     string `json:"experiment_run_id,omitempty"`
	DefenseBaseline     string `json:"defense_baseline,omitempty"`
	GatewayExperimentMode string `json:"gateway_experiment_mode,omitempty"`
	SyncProcessingMS    int64  `json:"sync_processing_ms,omitempty"`
}

// PolicyRule is written by the async loop into the gateway's policy cache (Async → Gateway).
type PolicyRule struct {
	RuleID           string `json:"rule_id"`
	Action           string `json:"action"` // e.g. DEGRADE_TO_TEMPLATE
	TriggerPattern   string `json:"trigger_pattern"`
	TemplateResponse string `json:"template_response"`
}

package logsink

import "github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"

// Multi fans out each event to every sink (order preserved).
type Multi []Sink

func (m Multi) Emit(e contracts.GatewayLogEvent) {
	for _, s := range m {
		s.Emit(e)
	}
}

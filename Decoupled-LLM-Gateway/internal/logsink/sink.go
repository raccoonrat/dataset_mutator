package logsink

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
)

// Sink receives gateway log events (stdout, queue, etc.).
type Sink interface {
	Emit(e contracts.GatewayLogEvent)
}

// JSONLines writes one JSON object per line to w.
type JSONLines struct {
	mu  sync.Mutex
	w   io.Writer
	enc *json.Encoder
}

func NewJSONLines(w io.Writer) *JSONLines {
	return &JSONLines{w: w, enc: json.NewEncoder(w)}
}

func (j *JSONLines) Emit(e contracts.GatewayLogEvent) {
	j.mu.Lock()
	defer j.mu.Unlock()
	_ = j.enc.Encode(e)
}

// Stdout is the default sink for local dev and piping into the Python worker.
func Stdout() *JSONLines {
	return NewJSONLines(os.Stdout)
}

// Discard drops events (async logging disabled).
type Discard struct{}

func (Discard) Emit(contracts.GatewayLogEvent) {}

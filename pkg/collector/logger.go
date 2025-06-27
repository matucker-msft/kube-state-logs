package collector

import (
	"encoding/json"
	"os"
	"time"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// LoggerImpl handles structured JSON logging
type LoggerImpl struct {
	encoder *json.Encoder
}

// NewLogger creates a new Logger instance
func NewLogger() interfaces.Logger {
	return &LoggerImpl{
		encoder: json.NewEncoder(os.Stdout),
	}
}

// Log writes a log entry as JSON to stdout
func (l *LoggerImpl) Log(entry types.LogEntry) error {
	// Ensure timestamp is set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return l.encoder.Encode(entry)
}

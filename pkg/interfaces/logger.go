package interfaces

import (
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// Logger interface defines the logging contract
type Logger interface {
	Log(entry types.LogEntry) error
}

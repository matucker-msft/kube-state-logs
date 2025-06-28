package interfaces

import (
	"context"
	"time"

	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"k8s.io/client-go/informers"
)

// Logger interface defines the logging contract
type Logger interface {
	Log(entry types.LogEntry) error
}

// ResourceHandler defines the interface for resource-specific collectors
type ResourceHandler interface {
	SetupInformer(factory informers.SharedInformerFactory, logger Logger, resyncPeriod time.Duration) error
	Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error)
}

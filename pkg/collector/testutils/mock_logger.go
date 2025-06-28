package testutils

import (
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

type MockLogger struct {
	logs []types.LogEntry
}

func (m *MockLogger) Log(entry types.LogEntry) error {
	m.logs = append(m.logs, entry)
	return nil
}

func (m *MockLogger) GetLogs() []types.LogEntry {
	return m.logs
}

func (m *MockLogger) Clear() {
	m.logs = nil
}

package application

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"ibsolver/internal/domain"
)

// PreviewExternal keeps the exact parsed file on the service until save/cancel.
// Numbers and checksum never make a round trip through editable bridge input.
func (s *Service) PreviewExternal(configurationID int64, source string, data []byte) (domain.ExternalResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingExternal = nil
	chart, err := ReadSeriesCSV(data)
	if err != nil {
		return domain.ExternalResult{}, err
	}
	state, err := s.store.Load()
	if err != nil {
		return domain.ExternalResult{}, err
	}
	if state.Active.ID != configurationID {
		return domain.ExternalResult{}, fmt.Errorf("Активна конфігурація змінилася")
	}
	r := domain.ExternalResult{ConfigurationID: configurationID, Name: source, Source: source,
		ImportedAt: time.Now().UTC().Format(time.RFC3339Nano), FormatVersion: domain.ExternalFormatVersion,
		Checksum: fmt.Sprintf("%x", sha256.Sum256(data)), Chart: chart}
	// Stable file-specific identities keep CSV preferences separate from model runs.
	for i := range r.Chart.Series {
		r.Chart.Series[i].ID = fmt.Sprintf("external:%s:%d", r.Checksum, i)
	}
	s.pendingExternal = &r
	return r, nil
}

func (s *Service) CancelExternalImport() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingExternal = nil
}

func (s *Service) SaveExternalImport(configurationID int64, name string) (domain.ExternalResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingExternal == nil || s.pendingExternal.ConfigurationID != configurationID {
		return domain.ExternalResult{}, fmt.Errorf("Немає підготовленого імпорту для цієї конфігурації")
	}
	r := *s.pendingExternal
	r.Name = strings.TrimSpace(name)
	// The import timestamp records the confirmed save, not the preview time.
	r.ImportedAt = time.Now().UTC().Format(time.RFC3339Nano)
	id, err := s.store.SaveExternal(r)
	if err != nil {
		return domain.ExternalResult{}, err
	}
	r.ID = id
	s.pendingExternal = nil
	return r, nil
}

func (s *Service) OpenExternal(configurationID, id int64) (domain.ExternalResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.OpenExternal(configurationID, id)
}

func (s *Service) DeleteExternal(configurationID, id int64) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.DeleteExternal(configurationID, id); err != nil {
		return domain.State{}, err
	}
	return s.load()
}

func (s *Service) ExportExternalCSV(configurationID, id int64) ([]byte, error) {
	r, err := s.OpenExternal(configurationID, id)
	if err != nil {
		return nil, err
	}
	return exportChartCSV(r.Chart)
}

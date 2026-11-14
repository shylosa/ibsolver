package application

import (
	"context"
	"fmt"
	"sync"

	"ibsolver/internal/chartdata"
	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

type Service struct {
	mu     sync.Mutex
	store  *sqlite.Store
	cancel context.CancelFunc
}

func New(store *sqlite.Store) *Service { return &Service{store: store} }

func (s *Service) load() (domain.State, error) {
	state, err := s.store.Load()
	if err != nil {
		return state, err
	}
	if len(state.Configurations) == 0 {
		_, err = s.store.SaveConfiguration(domain.Configuration{Name: "Нова конфігурація", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}, state.Settings)
		if err != nil {
			return state, err
		}
		warning := state.Warning
		state, err = s.store.Load()
		state.Warning = warning + state.Warning
	}
	return state, err
}

// Calculate takes an immutable snapshot, then releases the storage lock while
// integrating. A failed, canceled or superseded calculation never replaces a run.
func (s *Service) Calculate(ctx context.Context, configurationID int64) (domain.State, error) {
	s.mu.Lock()
	state, err := s.load()
	if err != nil {
		s.mu.Unlock()
		return state, err
	}
	if state.Active.ID != configurationID || state.Active.Mode != domain.ModeOscillator || state.Active.Parameters.Oscillator == nil {
		s.mu.Unlock()
		return state, fmt.Errorf("Активна конфігурація змінилася")
	}
	if s.cancel != nil {
		s.mu.Unlock()
		return state, fmt.Errorf("Розрахунок уже виконується")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	input := *state.Active.Parameters.Oscillator
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); s.cancel = nil; s.mu.Unlock() }()
	run, err := chartdata.Oscillator(ctx, input)
	if err != nil {
		return state, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.load()
	if err != nil {
		return state, err
	}
	if current.Active.ID != configurationID || !run.Matches(current.Active) {
		return current, fmt.Errorf("Параметри змінилися під час розрахунку; результат не записано")
	}
	if err = ctx.Err(); err != nil {
		return current, fmt.Errorf("Розрахунок скасовано: %w", err)
	}
	run.ConfigurationID = configurationID
	if err = s.store.SaveRun(run); err != nil {
		return current, err
	}
	return s.load()
}

func (s *Service) CancelCalculation() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Service) Load() (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Service) SaveConfiguration(c domain.Configuration, settings domain.Settings) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.store.SaveConfiguration(c, settings); err != nil {
		return domain.State{}, err
	}
	return s.load()
}

func (s *Service) SaveSettings(settings domain.Settings) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.SaveSettings(settings); err != nil {
		return domain.State{}, err
	}
	return s.load()
}

func (s *Service) CreateDemo(configurationID int64) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return state, err
	}
	if state.Active.ID != configurationID || state.Active.Mode != domain.ModeDemo || state.Active.Parameters.Demo == nil {
		return state, fmt.Errorf("Активна конфігурація змінилася")
	}
	run, err := chartdata.Demo(*state.Active.Parameters.Demo)
	if err != nil {
		return state, err
	}
	run.ConfigurationID = configurationID
	if err = s.store.SaveRun(run); err != nil {
		return state, err
	}
	return s.load()
}

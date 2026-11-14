package application

import (
	"fmt"
	"sync"

	"ibsolver/internal/chartdata"
	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

type Service struct {
	mu    sync.Mutex
	store *sqlite.Store
}

func New(store *sqlite.Store) *Service { return &Service{store: store} }

func (s *Service) load() (domain.State, error) {
	state, err := s.store.Load()
	if err != nil {
		return state, err
	}
	if len(state.Configurations) == 0 {
		_, err = s.store.SaveConfiguration(domain.Configuration{Name: "Нова конфігурація", Inputs: domain.DefaultInputs()}, state.Settings)
		if err != nil {
			return state, err
		}
		warning := state.Warning
		state, err = s.store.Load()
		state.Warning = warning + state.Warning
	}
	return state, err
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
	if state.Active.ID != configurationID {
		return state, fmt.Errorf("Активна конфігурація змінилася")
	}
	run, err := chartdata.Demo(state.Active.Inputs)
	if err != nil {
		return state, err
	}
	run.ConfigurationID = configurationID
	if err = s.store.SaveRun(run); err != nil {
		return state, err
	}
	return s.load()
}

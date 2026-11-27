package application

import (
	"context"
	"fmt"
	"sync"

	"ibsolver/internal/chartdata"
	"ibsolver/internal/domain"
	"ibsolver/internal/prepcalc"
	"ibsolver/internal/storage/sqlite"
)

type Service struct {
	mu              sync.Mutex
	store           *sqlite.Store
	cancel          context.CancelFunc
	pendingExternal *domain.ExternalResult
}

func New(store *sqlite.Store) *Service { return &Service{store: store} }

func (s *Service) load() (domain.State, error) {
	state, err := s.store.Load()
	if err != nil {
		return state, err
	}
	if len(state.Configurations) == 0 {
		_, err = s.store.SaveConfiguration(domain.Configuration{Name: "Нова конфігурація", Mode: domain.ModePreparation, Parameters: domain.NewPreparationParameters()}, state.Settings)
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
	if state.Active.ID != configurationID {
		s.mu.Unlock()
		return state, fmt.Errorf("Активна конфігурація змінилася")
	}
	mode := state.Active.Mode
	if mode != domain.ModeOscillator && mode != domain.ModePreparation {
		s.mu.Unlock()
		return state, fmt.Errorf("Для активного режиму розрахунок не підтримується")
	}
	if mode == domain.ModeOscillator && state.Active.Parameters.Oscillator == nil {
		s.mu.Unlock()
		return state, fmt.Errorf("Відсутні параметри осцилятора")
	}
	if s.cancel != nil {
		s.mu.Unlock()
		return state, fmt.Errorf("Розрахунок уже виконується")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	activeParams := state.Active.Parameters
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); s.cancel = nil; s.mu.Unlock() }()

	var run domain.Run
	if mode == domain.ModeOscillator {
		run, err = chartdata.Oscillator(ctx, *activeParams.Oscillator)
	} else {
		prep := domain.PreparationInputs{}
		if activeParams.Preparation != nil {
			prep = *activeParams.Preparation
		}
		run, err = chartdata.Ballistics(ctx, prep)
	}
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
	run.Summary.ConfigurationName = state.Active.Name
	if err = s.store.SaveRun(run); err != nil {
		return current, err
	}
	return s.load()
}

// CalculateWorkbook runs the Диплом.xls preparation kernel (package prepcalc)
// for the seven workbook fields of the active configuration. The result is a
// summary-only run: it has no time series and never touches the chart data
// of other run kinds.
func (s *Service) CalculateWorkbook(ctx context.Context, configurationID int64) (domain.State, error) {
	s.mu.Lock()
	state, err := s.load()
	if err != nil {
		s.mu.Unlock()
		return state, err
	}
	if state.Active.ID != configurationID {
		s.mu.Unlock()
		return state, fmt.Errorf("Активна конфігурація змінилася")
	}
	if state.Active.Mode != domain.ModePreparation {
		s.mu.Unlock()
		return state, fmt.Errorf("Підготовчий розрахунок доступний лише в режимі підготовчих даних")
	}
	if s.cancel != nil {
		s.mu.Unlock()
		return state, fmt.Errorf("Розрахунок уже виконується")
	}
	inputs, err := domain.WorkbookInputs(state.Active.Parameters.PreparationWorkbook)
	if err != nil {
		s.mu.Unlock()
		return state, err
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	snapshot := state.Active.Parameters
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); s.cancel = nil; s.mu.Unlock() }()

	result, err := prepcalc.Calculate(inputs, prepcalc.DefaultTable1())
	if err != nil {
		return state, err
	}
	run := domain.Run{
		Mode:          domain.ModePreparation,
		Snapshot:      snapshot,
		ResultVersion: domain.WorkbookVersion,
		Summary:       domain.Summary{Kind: "WORKBOOK", Workbook: &result},
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
	run.Summary.ConfigurationName = state.Active.Name
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

// RenameConfiguration changes only the display name of an existing
// configuration; the current parameter draft is irrelevant and untouched.
func (s *Service) RenameConfiguration(id int64, name string) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.RenameConfiguration(id, name); err != nil {
		return domain.State{}, err
	}
	return s.load()
}

// DeleteConfiguration removes the configuration with all of its runs and
// external results; the state reload picks the new active configuration.
func (s *Service) DeleteConfiguration(id int64) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.DeleteConfiguration(id); err != nil {
		return domain.State{}, err
	}
	return s.load()
}

// DuplicateConfiguration copies the configuration under a unique name and
// activates the copy.
func (s *Service) DuplicateConfiguration(id int64) (domain.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.store.DuplicateConfiguration(id); err != nil {
		return domain.State{}, err
	}
	return s.load()
}


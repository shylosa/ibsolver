package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"ibsolver/internal/application"
	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

// App struct
type App struct {
	closeAllowed atomic.Bool
	service      *application.Service
	store        *sqlite.Store
	startupErr   error
	initDone     chan struct{}
	ctx          context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{initDone: make(chan struct{})}
}

// startup opens local storage once; bridge calls report initialization failures.
func (a *App) startup(ctx context.Context) {
	defer close(a.initDone)
	a.ctx = ctx
	path, err := sqlite.DefaultPath()
	if err == nil {
		a.store, err = sqlite.Open(path)
	}
	a.startupErr = err
	if err == nil {
		a.service = application.New(a.store)
	}
}

func (a *App) shutdown(ctx context.Context) {
	<-a.initDone
	if a.store != nil {
		a.service.CancelCalculation()
		a.store.Close()
	}
}

func (a *App) Calculate(configurationID int64) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.Calculate(a.ctx, configurationID)
}

// CalculateWorkbook runs the Диплом.xls preparation kernel for the active
// configuration; the result is a summary-only run without time series.
func (a *App) CalculateWorkbook(configurationID int64) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.CalculateWorkbook(a.ctx, configurationID)
}

func (a *App) CancelCalculation() {
	if a.ready() == nil {
		a.service.CancelCalculation()
	}
}

func (a *App) ready() error {
	<-a.initDone
	if a.startupErr != nil {
		return fmt.Errorf("Не вдалося відкрити базу даних: %w", a.startupErr)
	}
	if a.service == nil {
		return fmt.Errorf("Застосунок ще не готовий")
	}
	return nil
}

func (a *App) LoadState() (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.Load()
}

func (a *App) SaveConfiguration(c domain.Configuration, settings domain.Settings) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.SaveConfiguration(c, settings)
}

func (a *App) SaveSettings(settings domain.Settings) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.SaveSettings(settings)
}

func (a *App) RenameConfiguration(id int64, name string) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.RenameConfiguration(id, name)
}

func (a *App) DeleteConfiguration(id int64) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.DeleteConfiguration(id)
}

func (a *App) DuplicateConfiguration(id int64) (domain.State, error) {
	if err := a.ready(); err != nil {
		return domain.State{}, err
	}
	return a.service.DuplicateConfiguration(id)
}

// beforeClose keeps the window alive until the frontend resolves its draft.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.closeAllowed.Load() {
		return false
	}
	runtime.EventsEmit(ctx, "close-request")
	return true
}

func (a *App) CloseWindow() {
	a.closeAllowed.Store(true)
	runtime.Quit(a.ctx)
}

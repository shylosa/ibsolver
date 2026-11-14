package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"ibsolver/internal/domain"
)

func (a *App) PreviewExternalImport(configurationID int64) (*domain.ExternalResult, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	a.service.CancelExternalImport()
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Імпортувати зовнішній результат", Filters: []runtime.FileFilter{{DisplayName: "Часові ряди CSV", Pattern: "*.csv"}}})
	if err != nil || path == "" {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (8<<20)+1))
	if err != nil {
		return nil, err
	}
	r, err := a.service.PreviewExternal(configurationID, filepath.Base(path), data)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (a *App) CancelExternalImport() {
	if a.ready() == nil {
		a.service.CancelExternalImport()
	}
}

func (a *App) SaveExternalImport(configurationID int64, name string) (domain.ExternalResult, error) {
	if err := a.ready(); err != nil {
		return domain.ExternalResult{}, err
	}
	return a.service.SaveExternalImport(configurationID, name)
}

func (a *App) OpenExternalResult(configurationID, id int64) (domain.ExternalResult, error) {
	if err := a.ready(); err != nil {
		return domain.ExternalResult{}, err
	}
	return a.service.OpenExternal(configurationID, id)
}

func (a *App) DeleteExternalResult(configurationID, id int64) (*domain.State, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	// Windows uses system Yes/No buttons and response names, ignoring custom labels.
	answer, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.QuestionDialog, Title: "Видалити зовнішній результат?", Message: "Збережені числові дані буде видалено. Цю дію неможливо скасувати.", Buttons: []string{"Видалити", "Скасувати"}, DefaultButton: "No", CancelButton: "No"})
	if err != nil || (answer != "Yes" && answer != "Видалити") {
		return nil, err
	}
	state, err := a.service.DeleteExternal(configurationID, id)
	return &state, err
}

func (a *App) ExportExternalCSV(configurationID, id int64) (bool, error) {
	if err := a.ready(); err != nil {
		return false, err
	}
	data, err := a.service.ExportExternalCSV(configurationID, id)
	if err != nil {
		return false, err
	}
	return a.saveExchange(data, "csv", "external-result.csv")
}

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"ibsolver/internal/application"
	"ibsolver/internal/domain"
)

func (a *App) ImportConfiguration() (*domain.State, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Імпортувати конфігурацію", Filters: []runtime.FileFilter{{DisplayName: "Конфігурація JSON", Pattern: "*.json"}}})
	if err != nil {
		return nil, fmt.Errorf("Не вдалося відкрити діалог: %w", err)
	}
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Не вдалося відкрити файл: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, application.MaxConfigurationBytes+1))
	if err != nil {
		return nil, fmt.Errorf("Не вдалося прочитати файл: %w", err)
	}
	state, err := a.service.ImportConfiguration(data)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (a *App) ExportConfiguration(id int64) (bool, error) {
	if err := a.ready(); err != nil {
		return false, err
	}
	data, err := a.service.ExportConfiguration(id)
	if err != nil {
		return false, err
	}
	return a.saveExchange(data, "json", "configuration.json")
}

func (a *App) ExportCSV(id int64) (bool, error) {
	if err := a.ready(); err != nil {
		return false, err
	}
	data, err := a.service.ExportCSV(id)
	if err != nil {
		return false, err
	}
	return a.saveExchange(data, "csv", "results.csv")
}

func (a *App) saveExchange(data []byte, extension, filename string) (bool, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Зберегти файл", DefaultFilename: filename, Filters: []runtime.FileFilter{{DisplayName: strings.ToUpper(extension), Pattern: "*." + extension}}})
	if err != nil {
		return false, fmt.Errorf("Не вдалося відкрити діалог: %w", err)
	}
	if path == "" {
		return false, nil
	}
	if filepath.Ext(path) == "" {
		path += "." + extension
	}
	if _, err := os.Stat(path); err == nil {
		answer, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.QuestionDialog, Title: "Замінити файл?", Message: "Файл уже існує. Замінити його?", Buttons: []string{"Так", "Скасувати"}, DefaultButton: "Скасувати", CancelButton: "Скасувати"})
		if err != nil {
			return false, err
		}
		if answer != "Так" {
			return false, nil
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("Не вдалося перевірити файл: %w", err)
	}
	if err := writeExchange(path, data); err != nil {
		return false, fmt.Errorf("Не вдалося зберегти файл: %w", err)
	}
	return true, nil
}

// Stage the complete file before replacing an existing export.
func writeExchange(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".ibsolver-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

package application

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"ibsolver/internal/domain"
)

const MaxConfigurationBytes = 1 << 20

type configurationFile struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	Name       string            `json:"name"`
	Mode       domain.Mode       `json:"mode"`
	Parameters domain.Parameters `json:"parameters"`
}

func validateExchange(c domain.Configuration) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Parameters.Demo != nil {
		if err := c.Parameters.Demo.Validate(); err != nil {
			return err
		}
	}
	if c.Parameters.Oscillator != nil {
		return c.Parameters.Oscillator.Validate()
	}
	return nil
}

func (s *Service) ExportConfiguration(id int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if state.Active.ID != id {
		return nil, fmt.Errorf("Активна конфігурація змінилася")
	}
	c := state.Active
	if err := validateExchange(c); err != nil {
		return nil, err
	}
	return json.MarshalIndent(configurationFile{"ibsolver-configuration", 1, c.Name, c.Mode, c.Parameters}, "", "  ")
}

func (s *Service) ImportConfiguration(data []byte) (domain.State, error) {
	if len(data) > MaxConfigurationBytes {
		return domain.State{}, fmt.Errorf("Файл конфігурації перевищує 1 МіБ")
	}
	if !utf8.Valid(data) {
		return domain.State{}, fmt.Errorf("Файл має бути у UTF-8")
	}
	var f configurationFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f); err != nil {
		return domain.State{}, fmt.Errorf("Некоректний JSON конфігурації: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return domain.State{}, fmt.Errorf("Очікується один JSON-об’єкт")
	}
	if f.Format != "ibsolver-configuration" || f.Version != 1 {
		return domain.State{}, fmt.Errorf("Непідтримуваний формат або версія конфігурації")
	}
	// Require complete typed objects; JSON null must not silently become numeric zero.
	if err := completeConfigurationJSON(data); err != nil {
		return domain.State{}, err
	}
	c := domain.Configuration{Name: f.Name, Mode: f.Mode, Parameters: f.Parameters}
	if err := validateExchange(c); err != nil {
		return domain.State{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load()
	if err != nil {
		return state, err
	}
	if _, err = s.store.SaveConfiguration(c, state.Settings); err != nil {
		return state, err
	}
	return s.store.Load()
}

func completeConfigurationJSON(data []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	for _, key := range []string{"format", "version", "name", "mode", "parameters"} {
		if len(object[key]) == 0 || bytes.Equal(bytes.TrimSpace(object[key]), []byte("null")) {
			return fmt.Errorf("Відсутнє значення: %s", key)
		}
	}
	var parameters map[string]json.RawMessage
	if err := json.Unmarshal(object["parameters"], &parameters); err != nil {
		return err
	}
	for mode, raw := range parameters {
		var template []byte
		if mode == "demo" {
			template, _ = json.Marshal(domain.DefaultInputs())
		} else {
			template, _ = json.Marshal(domain.DefaultNumerical())
		}
		if err := requireJSONFields(raw, template); err != nil {
			return err
		}
	}
	return nil
}

func requireJSONFields(raw, template []byte) error {
	var actual, expected map[string]json.RawMessage
	if err := json.Unmarshal(raw, &actual); err != nil {
		return err
	}
	if err := json.Unmarshal(template, &expected); err != nil {
		return err
	}
	for key, shape := range expected {
		value := bytes.TrimSpace(actual[key])
		if len(value) == 0 || bytes.Equal(value, []byte("null")) {
			return fmt.Errorf("Відсутній параметр: %s", key)
		}
		if shape[0] == '{' {
			if err := requireJSONFields(value, shape); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportCSV reads the saved run, independent of current inputs and display settings.
func (s *Service) ExportCSV(id int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if state.Active.ID != id || state.Run == nil {
		return nil, fmt.Errorf("Немає збереженого запуску для експорту")
	}
	d := state.Run.Chart
	if err := d.Validate(); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	row := make([]string, len(d.Series)+1)
	row[0] = "Час [с]"
	for i, series := range d.Series {
		row[i+1] = series.Name + " [" + series.Unit + "]"
	}
	if err := writer.Write(row); err != nil {
		return nil, err
	}
	for i, t := range d.Time {
		row[0] = strconv.FormatFloat(t, 'g', -1, 64)
		for j, series := range d.Series {
			row[j+1] = strconv.FormatFloat(series.Values[i], 'g', -1, 64)
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return output.Bytes(), writer.Error()
}

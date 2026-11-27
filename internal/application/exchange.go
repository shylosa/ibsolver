package application

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
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
		if err := c.Parameters.Oscillator.Validate(); err != nil {
			return err
		}
	}
	if c.Parameters.Preparation != nil {
		return c.Parameters.Preparation.Validate()
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
	version := 1
	if c.Parameters.Preparation != nil || c.Parameters.PreparationWorkbook != nil {
		version = 2
	}
	return json.MarshalIndent(configurationFile{"ibsolver-configuration", version, c.Name, c.Mode, c.Parameters}, "", "  ")
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
	if f.Format != "ibsolver-configuration" || (f.Version != 1 && f.Version != 2) ||
		(f.Version == 1 && (f.Parameters.Preparation != nil || f.Parameters.PreparationWorkbook != nil)) {
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
		if mode == "preparation" {
			if err := validateNullableNumbers(raw, []string{"d", "q", "vd", "lc", "ns", "f1", "af"}); err != nil {
				return err
			}
			continue
		} else if mode == "preparationWorkbook" {
			if err := validateNullableNumbers(raw, []string{"d", "q", "vd", "lc", "ns", "f1", "af"}); err != nil {
				return err
			}
			continue
		} else if mode == "demo" {
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

func validateNullableNumbers(raw []byte, keys []string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, key := range keys {
		if len(fields[key]) == 0 {
			return fmt.Errorf("Відсутній параметр: %s", key)
		}
		text := strings.TrimSpace(string(fields[key]))
		if text != "null" {
			n, err := strconv.ParseFloat(text, 64)
			if err != nil || decimalUnderflow(text, n) {
				return fmt.Errorf("Число поза діапазоном binary64: %s", key)
			}
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
	return exportRunCSV(*state.Run)
}

func exportRunCSV(run domain.Run) ([]byte, error) {
	if run.Mode != domain.ModePreparation {
		return exportChartCSV(run.Chart)
	}
	return exportBallisticsCSV(run.Chart)
}

func exportBallisticsCSV(d domain.ChartData) ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	// Column order, units and numeric formats follow the legacy BAS export,
	// e.g. D:\go\ibsolver-refs\2A65.CSV. Stored pressure is MPa, exported in Pa.
	ordered := []struct {
		id     string
		header string
		scale  float64
		format string
	}{
		{domain.SeriesID(domain.ModePreparation, 0), "P_Pa", 1e6, "%.1f"},
		{domain.SeriesID(domain.ModePreparation, 1), "T_K", 1, "%.1f"},
		{domain.SeriesID(domain.ModePreparation, 6), "E_mm", 1, "%.9f"},
		{domain.SeriesID(domain.ModePreparation, 7), "Z", 1, "%.9f"},
		{domain.SeriesID(domain.ModePreparation, 4), "PSI", 1, "%.9f"},
		{domain.SeriesID(domain.ModePreparation, 2), "V_m_s", 1, "%.4f"},
		{domain.SeriesID(domain.ModePreparation, 3), "L_m", 1, "%.5f"},
		{domain.SeriesID(domain.ModePreparation, 5), "G_kg_s", 1, "%.4f"},
	}
	seriesByID := make(map[string]domain.Series, len(d.Series)+len(d.Aux))
	for _, series := range d.Series {
		seriesByID[series.ID] = series
	}
	for _, series := range d.Aux {
		seriesByID[series.ID] = series
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	header := make([]string, len(ordered)+1)
	header[0] = "t_ms"
	series := make([]domain.Series, len(ordered))
	for i, item := range ordered {
		value, ok := seriesByID[item.id]
		if !ok {
			return nil, fmt.Errorf("Відсутній балістичний ряд для CSV: %s; виконайте розрахунок", item.id)
		}
		header[i+1] = item.header
		series[i] = value
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	row := make([]string, len(header))
	for i, time := range d.Time {
		timeMS := time * 1000
		if math.IsNaN(timeMS) || math.IsInf(timeMS, 0) {
			return nil, fmt.Errorf("Час поза діапазоном мілісекунд у рядку %d", i+2)
		}
		row[0] = fmt.Sprintf("%.6f", timeMS)
		for j, item := range ordered {
			row[j+1] = fmt.Sprintf(item.format, series[j].Values[i]*item.scale)
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return output.Bytes(), writer.Error()
}

func exportChartCSV(d domain.ChartData) ([]byte, error) {
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

// ReadSeriesCSV opens neutral numeric data independently of configurations/runs.
// The first column is time in seconds, followed by up to six named series.
func ReadSeriesCSV(data []byte) (domain.ChartData, error) {
	var chart domain.ChartData
	if len(data) > 8<<20 || !utf8.Valid(data) {
		return chart, fmt.Errorf("CSV має бути UTF-8 розміром до 8 МіБ")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	header, err := reader.Read()
	if err != nil || len(header) < 2 || len(header) > 7 || (header[0] != "Час [с]" && header[0] != "time [s]" && header[0] != "t [ms]") {
		return chart, fmt.Errorf("CSV: перший стовпець «time [s]», «Час [с]» або «t [ms]», далі 1–6 рядів «Назва [одиниця]»; роздільник — кома")
	}
	timeScale := 1.0
	if header[0] == "t [ms]" {
		timeScale = 0.001
	}
	for i, label := range header[1:] {
		if label == "PSI" {
			chart.Series = append(chart.Series, domain.Series{ID: fmt.Sprintf("csv-%d", i), Name: "PSI"})
			continue
		}
		end := strings.LastIndex(label, " [")
		if end < 1 || !strings.HasSuffix(label, "]") {
			return chart, fmt.Errorf("CSV: вкажіть назву та одиницю кожного ряду")
		}
		chart.Series = append(chart.Series, domain.Series{ID: fmt.Sprintf("csv-%d", i), Name: label[:end], Unit: label[end+2 : len(label)-1]})
	}
	for rowIndex := 2; ; rowIndex++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return chart, fmt.Errorf("CSV, рядок %d: %w", rowIndex, err)
		}
		if len(chart.Time) >= 20000 {
			return chart, fmt.Errorf("CSV: не більше 20000 точок")
		}
		for column, text := range row {
			n, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || decimalUnderflow(text, n) {
				return chart, fmt.Errorf("CSV: некоректне число, рядок %d, стовпець %d", rowIndex, column+1)
			}
			if column == 0 {
				rawTime := n
				n *= timeScale
				if math.IsNaN(n) || math.IsInf(n, 0) || (rawTime != 0 && n == 0) {
					return chart, fmt.Errorf("CSV: час поза діапазоном секунд, рядок %d", rowIndex)
				}
				chart.Time = append(chart.Time, n)
			} else {
				chart.Series[column-1].Values = append(chart.Series[column-1].Values, n)
			}
		}
	}
	return chart, chart.Validate()
}

func decimalUnderflow(text string, value float64) bool {
	if value != 0 {
		return false
	}
	mantissa := strings.FieldsFunc(text, func(r rune) bool { return r == 'e' || r == 'E' })
	return len(mantissa) > 0 && strings.ContainsAny(mantissa[0], "123456789")
}

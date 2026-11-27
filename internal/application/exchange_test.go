package application

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

func TestConfigurationExchangeAndFailedImportsPreserveState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exchange.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	state, err := service.Load()
	if err == nil {
		c := state.Active
		c.Mode, c.Parameters = domain.ModeOscillator, domain.DefaultParameters()
		state, err = service.SaveConfiguration(c, state.Settings)
	}
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.Calculate(t.Context(), state.Active.ID)
	if err != nil {
		t.Fatal(err)
	}
	original := state
	data, err := service.ExportConfiguration(state.Active.ID)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 5 || fields["id"] != nil || fields["settings"] != nil {
		t.Fatal("internal data exported")
	}
	for _, invalid := range [][]byte{
		[]byte(`{`), []byte(`null`), []byte(`[]`),
		bytes.Replace(data, []byte(`"position": 1`), []byte(`"position": null`), 1),
		bytes.Replace(data, []byte(`"position": 1,`), []byte(``), 1),
		bytes.Replace(data, []byte(`"version": 1`), []byte(`"version": 99`), 1),
		bytes.Replace(data, []byte(`"oscillator"`), []byte(`"unknown"`), 1),
		bytes.Replace(data, []byte(`"mass": 1`), []byte(`"mass": 1e999`), 1),
		bytes.Replace(data, []byte(`"mass": 1`), []byte(`"mass": NaN`), 1),
		append(append([]byte{}, data...), []byte(` {}`)...),
		[]byte(strings.Repeat(" ", MaxConfigurationBytes+1)), {0xff},
	} {
		if _, err := service.ImportConfiguration(invalid); err == nil {
			t.Fatalf("accepted invalid import %.200s", invalid)
		}
		after, err := service.Load()
		if err != nil || !reflect.DeepEqual(after, original) {
			t.Fatal("failed import changed state", err)
		}
	}
	imported, err := service.ImportConfiguration(data)
	if err != nil || imported.Active.ID == original.Active.ID || len(imported.Configurations) != 2 || imported.Run != nil || !reflect.DeepEqual(imported.Active.Parameters, original.Active.Parameters) {
		t.Fatalf("round trip: %+v %v", imported, err)
	}
	settings := imported.Settings
	settings.ActiveConfigurationID = original.Active.ID
	restored, err := service.SaveSettings(settings)
	if err != nil || !reflect.DeepEqual(restored.Run, original.Run) {
		t.Fatal("original run changed", err)
	}
	store.Close()
	store, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restarted, err := New(store).Load()
	if err != nil || !reflect.DeepEqual(restarted, restored) {
		t.Fatal("restart changed state", err)
	}
}

func TestCSVPreservesSavedValuesHeadersAndOrder(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "csv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store)
	state, err := service.Load()
	if err == nil {
		c := state.Active
		c.Mode, c.Parameters = domain.ModeOscillator, domain.DefaultParameters()
		state, err = service.SaveConfiguration(c, state.Settings)
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExportCSV(state.Active.ID); err == nil {
		t.Fatal("export without run")
	}
	state, err = service.Calculate(t.Context(), state.Active.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Persist a valid result with a header requiring standard CSV escaping.
	run := *state.Run
	run.Chart.Series[0].Name = "Зміщення, \"x\""
	if err := store.SaveRun(run); err != nil {
		t.Fatal(err)
	}
	c := state.Active
	c.Parameters.Oscillator.Model.Position = 2
	state.Settings.Series[run.Chart.Series[0].ID] = domain.SeriesSettings{Visible: false, Scale: 7}
	state, err = service.SaveConfiguration(c, state.Settings)
	if err != nil {
		t.Fatal(err)
	}
	before := state
	output, err := service.ExportCSV(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(output)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][0] != "Час [с]" || len(rows) != len(run.Chart.Time)+1 || len(rows[0]) != 7 {
		t.Fatal("CSV dimensions/header")
	}
	for j, series := range run.Chart.Series {
		if rows[0][j+1] != series.Name+" ["+series.Unit+"]" {
			t.Fatal("header changed")
		}
	}
	for i, time := range run.Chart.Time {
		for j, text := range rows[i+1] {
			value, err := strconv.ParseFloat(text, 64)
			expected := time
			if j > 0 {
				expected = run.Chart.Series[j-1].Values[i]
			}
			if err != nil || value != expected {
				t.Fatalf("precision lost at %d,%d", i, j)
			}
		}
	}
	after, err := service.Load()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("export mutated state", err)
	}
}

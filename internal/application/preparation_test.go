package application

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

func TestPreparationPersistenceExchangeAndHistoricalRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preparation.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	state, err := service.Load()
	if err != nil || state.Active.Mode != domain.ModePreparation || state.Run != nil {
		t.Fatalf("default: %+v %v", state, err)
	}
	state, err = service.Calculate(t.Context(), state.Active.ID)
	if err != nil || state.Run == nil || len(state.Run.Chart.Series) != 6 || state.Run.Summary.Kind != "BALLISTICS" {
		t.Fatalf("failed to calculate preparation ballistics: %+v %v", state.Run, err)
	}
	if _, err := service.CreateDemo(state.Active.ID); err == nil {
		t.Fatal("generated demo for preparation")
	}
	c := state.Active
	c.Mode = domain.ModeDemo
	state, err = service.SaveConfiguration(c, state.Settings)
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.CreateDemo(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	historical := state.Run
	d, q, vd, lc, ns, f1 := .01234567890123456, 0.0, -5.0, 1e100, 5e-324, 1.2e-5
	c.Mode = domain.ModePreparation
	c.Parameters.Preparation = &domain.PreparationInputs{D: &d, Q: &q, Vd: &vd, Lc: &lc, Ns: &ns, F1: &f1}
	state, err = service.SaveConfiguration(c, state.Settings)
	if err != nil || !reflect.DeepEqual(state.Run, historical) {
		t.Fatal("save changed historical run", err)
	}
	data, err := service.ExportConfiguration(c.ID)
	if err != nil || !bytes.Contains(data, []byte(`"version": 2`)) || !bytes.Contains(data, []byte(`"af": null`)) {
		t.Fatalf("export: %s %v", data, err)
	}
	for _, bad := range [][]byte{
		bytes.Replace(data, []byte(`"version": 2`), []byte(`"version": 3`), 1),
		bytes.Replace(data, []byte(`"version": 2`), []byte(`"version": 1`), 1),
		bytes.Replace(data, []byte(`"af": null`), []byte(`"other": null`), 1),
		bytes.Replace(data, []byte(`"af": null`), []byte(`"af": 1e-999`), 1),
		bytes.Replace(data, []byte(`"af": null`), []byte(`"af": "12"`), 1),
	} {
		if _, err := service.ImportConfiguration(bad); err == nil {
			t.Fatalf("accepted bad JSON %s", bad)
		}
		after, err := service.Load()
		if err != nil || !reflect.DeepEqual(state, after) {
			t.Fatal("failed import changed DB", err)
		}
	}
	imported, err := service.ImportConfiguration(data)
	if err != nil || imported.Active.ID == c.ID || imported.Run != nil || !reflect.DeepEqual(imported.Active.Parameters, c.Parameters) {
		t.Fatalf("round trip: %+v %v", imported, err)
	}
	store.Close()
	store, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service = New(store)
	restored, err := service.Load()
	if err != nil || !reflect.DeepEqual(restored, imported) {
		t.Fatal("restart lost values", err)
	}
	// A real SQLite failure cannot mutate the last committed revision.
	store.Close()
	changed := restored.Active
	changed.Name = "unsaved"
	if _, err := service.SaveConfiguration(changed, restored.Settings); err == nil {
		t.Fatal("write to closed SQLite succeeded")
	}
	store, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	actual, err := New(store).Load()
	if err != nil || !reflect.DeepEqual(actual, restored) {
		t.Fatal("failed write changed revision", err)
	}
}

func TestReadNeutralCSV(t *testing.T) {
	chart, err := ReadSeriesCSV([]byte("time [s],\"Signal, A [V]\",B [m]\n0,1.2345678901234567,-2\n.5,0,5e-324\n"))
	if err != nil || chart.Series[0].Name != "Signal, A" || chart.Series[0].Values[0] != 1.2345678901234567 || chart.Time[1] != .5 {
		t.Fatalf("CSV: %+v %v", chart, err)
	}
	for _, bad := range []string{"time,x\n0,1\n1,2", "time [s],x [m]\n0,1\n0,2", "time [s],x [m]\n0,1\n1,NaN", "time [s],x [m]\n0,1\n1,1e-999", "time [s],x [m]\n0,1\n1"} {
		if _, err := ReadSeriesCSV([]byte(bad)); err == nil {
			t.Fatal("accepted invalid CSV", bad)
		}
	}
}

package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
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
	exported, err := service.ExportCSV(state.Active.ID)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(exported)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := []string{"t_ms", "P_Pa", "T_K", "E_mm", "Z", "PSI", "V_m_s", "L_m", "G_kg_s"}
	if !reflect.DeepEqual(rows[0], wantHeader) || len(rows) != len(state.Run.Chart.Time)+1 || len(rows[1]) != len(wantHeader) {
		t.Fatalf("BAS CSV structure: header=%v rows=%d", rows[0], len(rows))
	}
	// Column order, units and numeric formats mirror the legacy BAS export,
	// e.g. D:\go\ibsolver-refs\2A65.CSV.
	aux := map[string][]float64{}
	for _, s := range state.Run.Chart.Aux {
		aux[s.ID] = s.Values
	}
	id := func(i int) string { return domain.SeriesID(domain.ModePreparation, i) }
	columns := []struct {
		values []float64
		scale  float64
		format string
	}{
		{state.Run.Chart.Time, 1000, "%.6f"},
		{state.Run.Chart.Series[0].Values, 1e6, "%.1f"},
		{state.Run.Chart.Series[1].Values, 1, "%.1f"},
		{aux[id(6)], 1, "%.9f"},
		{aux[id(7)], 1, "%.9f"},
		{state.Run.Chart.Series[4].Values, 1, "%.9f"},
		{state.Run.Chart.Series[2].Values, 1, "%.4f"},
		{state.Run.Chart.Series[3].Values, 1, "%.5f"},
		{state.Run.Chart.Series[5].Values, 1, "%.4f"},
	}
	for i := range state.Run.Chart.Time {
		for j, column := range columns {
			expected := fmt.Sprintf(column.format, column.values[i]*column.scale)
			if rows[i+1][j] != expected {
				t.Fatalf("BAS CSV value at %d,%d: got %q want %q", i+1, j, rows[i+1][j], expected)
			}
		}
	}
	if rows[1][1] != "10000000.0" || rows[1][2] != "1600.0" {
		t.Fatalf("BAS CSV initial pressure/temperature: %q %q", rows[1][1], rows[1][2])
	}
	c := state.Active
	historical := state.Run
	d, q, vd, lc, ns, f1 := .01234567890123456, 0.0, -5.0, 1e100, 5e-324, 1.2e-5
	c.Mode = domain.ModePreparation
	c.Parameters.Preparation = &domain.PreparationInputs{
		D: &d, Q: &q, Vd: &vd, Lc: &lc, Ns: &ns, F1: &f1,
		InputFormats: map[string]string{"q": "0.0", "w0": "17.1e-3"},
	}
	workbookD, workbookQ, workbookVd, workbookLc := 155.0, 42.5, 700.0, 5.2
	workbookNs, workbookF1, workbookAf := 0.83, 4.1, -0.75
	c.Parameters.PreparationWorkbook = &domain.PreparationWorkbookInputs{
		D: &workbookD, Q: &workbookQ, Vd: &workbookVd, Lc: &workbookLc,
		Ns: &workbookNs, F1: &workbookF1, Af: &workbookAf,
		InputFormats: map[string]string{"d": "155,0", "af": "-0,75"},
	}
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
	bas, err := ReadSeriesCSV([]byte("t [ms],Pstv [MPa],Tstv [K],PSI,V [m/s],L [m],G [kg/s]\n0,1,2,0,3,4,5\n500,6,7,0.5,8,9,10\n"))
	if err != nil || bas.Time[1] != .5 || bas.Series[2].Name != "PSI" || bas.Series[2].Unit != "" {
		t.Fatalf("BAS CSV: %+v %v", bas, err)
	}
	for _, bad := range []string{"time,x\n0,1\n1,2", "time [s],x [m]\n0,1\n0,2", "time [s],x [m]\n0,1\n1,NaN", "time [s],x [m]\n0,1\n1,1e-999", "time [s],x [m]\n0,1\n1"} {
		if _, err := ReadSeriesCSV([]byte(bad)); err == nil {
			t.Fatal("accepted invalid CSV", bad)
		}
	}
}

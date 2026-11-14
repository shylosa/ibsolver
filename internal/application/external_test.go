package application

import (
	"crypto/sha256"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

func TestExternalImportLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "external.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { store.Close() }()
	s := New(store)
	state, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	id := state.Active.ID
	data := []byte("\xef\xbb\xbftime [s],\"Measured, A []\",B [custom]\r\n-0,1.2345678901234567,5e-324\r\n0.10000000000000002,-0,1.7976931348623157e308\r\n")
	preview, err := s.PreviewExternal(id, "source.csv", data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Checksum != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("checksum is not of original bytes")
	}
	before, _ := s.Load()
	if !reflect.DeepEqual(state, before) {
		t.Fatal("preview wrote data")
	}
	s.CancelExternalImport()
	if _, err := s.SaveExternalImport(id, "Canceled"); err == nil {
		t.Fatal("saved canceled import")
	}
	if _, err := s.PreviewExternal(id+1, "wrong.csv", data); err == nil {
		t.Fatal("wrong configuration")
	}
	if _, err := s.PreviewExternal(id, "source.csv", data); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveExternalImport(id, " "); err == nil {
		t.Fatal("accepted empty name")
	}
	first, err := s.SaveExternalImport(id, "First")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveExternalImport(id, "Duplicate save"); err == nil {
		t.Fatal("saved preview twice")
	}
	if _, err := s.PreviewExternal(id, "source.csv", data); err != nil {
		t.Fatal(err)
	}
	second, err := s.SaveExternalImport(id, "Second")
	if err != nil || second.ID == first.ID {
		t.Fatal("second result", err)
	}
	state, err = s.Load()
	if err != nil || len(state.ExternalResults) != 2 || state.Run != nil {
		t.Fatal("history or fabricated run", err)
	}
	if _, err := s.OpenExternal(id+1, first.ID); err == nil {
		t.Fatal("cross-configuration read")
	}
	if _, err := s.DeleteExternal(id+1, first.ID); err == nil {
		t.Fatal("cross-configuration delete")
	}
	store.Close()
	store, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s = New(store)
	reopened, err := s.OpenExternal(id, first.ID)
	if err != nil || !reflect.DeepEqual(first, reopened) {
		t.Fatal("restart changed result", err)
	}
	assertChartBits(t, preview.Chart, reopened.Chart)
	exported, err := s.ExportExternalCSV(id, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	chart, err := ReadSeriesCSV(exported)
	if err != nil {
		t.Fatal(err)
	}
	assertChartBits(t, preview.Chart, chart)
	if chart.Series[0].Name != preview.Chart.Series[0].Name || chart.Series[0].Unit != "" {
		t.Fatal("changed labels")
	}
	settings := state.Settings
	settings.Series = map[string]domain.SeriesSettings{first.Chart.Series[0].ID: {Visible: false, Scale: 2}}
	if _, err := s.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	again, err := s.OpenExternal(id, first.ID)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("settings changed raw result", err)
	}
	state, err = s.DeleteExternal(id, first.ID)
	if err != nil || len(state.ExternalResults) != 1 || state.ExternalResults[0].ID != second.ID {
		t.Fatal("delete", err)
	}
	if _, err := s.OpenExternal(id, first.ID); err == nil {
		t.Fatal("opened deleted result")
	}
}

func assertChartBits(t *testing.T, want, got domain.ChartData) {
	t.Helper()
	if len(want.Time) != len(got.Time) || len(want.Series) != len(got.Series) {
		t.Fatal("shape changed")
	}
	for i, x := range want.Time {
		if math.Float64bits(x) != math.Float64bits(got.Time[i]) {
			t.Fatal("X precision changed", i)
		}
		for j, series := range want.Series {
			if math.Float64bits(series.Values[i]) != math.Float64bits(got.Series[j].Values[i]) {
				t.Fatal("Y precision changed", i, j)
			}
		}
	}
}

func TestExternalInvalidFilesNeverPersist(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := New(store)
	state, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{"", "time [s],A []\n", "time [s],A []\n0,1\n", "time [s],A []\n0,1\n1\n", "time [s],A []\n0,1\n0,2\n", "time [s],A []\n0,NaN\n1,2\n", "time [s],A []\n0,1e-999\n1,2\n", "time [s],A\n0,1\n1,2\n", "time [s],A []\n0,1\n1,2,3\n"} {
		if _, err := s.PreviewExternal(state.Active.ID, "bad.csv", []byte(data)); err == nil {
			t.Fatalf("accepted %q", data)
		}
		if _, err := s.SaveExternalImport(state.Active.ID, "bad"); err == nil {
			t.Fatal("saved invalid preview")
		}
	}
	after, err := s.Load()
	if err != nil || !reflect.DeepEqual(state, after) {
		t.Fatal("invalid import mutated state", err)
	}
}

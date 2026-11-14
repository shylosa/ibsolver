package sqlite

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ibsolver/internal/domain"
)

func TestExternalAtomicAndImmutable(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "atomic.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.SaveConfiguration(domain.Configuration{Name: "Preparation", Mode: domain.ModePreparation, Parameters: domain.NewPreparationParameters()}, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	r := domain.ExternalResult{ConfigurationID: id, Name: "External", ImportedAt: now(), Source: "file.csv", FormatVersion: domain.ExternalFormatVersion, Checksum: strings.Repeat("a", 64), Chart: domain.ChartData{Time: []float64{0, 1}, Series: []domain.Series{{ID: "external:a", Name: "A", Values: []float64{1, 2}}}}}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_external AFTER INSERT ON external_results BEGIN SELECT RAISE(ABORT, 'injected failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveExternal(r); err == nil {
		t.Fatal("injected failure ignored")
	}
	list, err := s.ListExternal(id)
	if err != nil || len(list) != 0 {
		t.Fatal("partial record persisted", err)
	}
	if _, err := s.db.Exec("DROP TRIGGER fail_external"); err != nil {
		t.Fatal(err)
	}
	r.ID, err = s.SaveExternal(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveExternal(r); err == nil {
		t.Fatal("rewrote existing result")
	}
	if _, err := s.db.Exec("UPDATE external_results SET checksum='changed' WHERE id=?", r.ID); err == nil {
		t.Fatal("immutable record updated")
	}
	got, err := s.OpenExternal(id, r.ID)
	if err != nil || !reflect.DeepEqual(r, got) {
		t.Fatal("record changed", err)
	}
	r.ID, r.ConfigurationID = 0, id+1
	if _, err := s.SaveExternal(r); err == nil {
		t.Fatal("accepted missing configuration")
	}
}

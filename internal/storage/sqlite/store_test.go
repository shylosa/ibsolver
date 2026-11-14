package sqlite

import (
	"path/filepath"
	"reflect"
	"testing"

	"ibsolver/internal/chartdata"
	"ibsolver/internal/domain"
)

func TestV1MigrationPreservesDemoAndAddsNumericalPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := domain.Configuration{Name: "Stage 1", Inputs: domain.DefaultInputs()}
	settings := domain.DefaultSettings()
	settings.Scales[1] = 2
	c.ID, err = s.SaveConfiguration(c, settings)
	if err != nil {
		t.Fatal(err)
	}
	run, err := chartdata.Demo(c.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	run.ConfigurationID = c.ID
	if err = s.SaveRun(run); err != nil {
		t.Fatal(err)
	}
	before, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	// Remove precisely the Stage 2 additions to reproduce the Stage 1 schema.
	if _, err = s.db.Exec(`ALTER TABLE configurations DROP COLUMN numerical_json;
ALTER TABLE runs DROP COLUMN numerical_snapshot_json;
PRAGMA user_version=1;`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	after, err := s.Load()
	if err != nil || after.Warning != "" || after.Stale || after.Active.Inputs != c.Inputs || after.Settings != before.Settings || !reflect.DeepEqual(after.Run, before.Run) || after.Active.Numerical == nil {
		t.Fatalf("v1 migration lost data: %+v %v", after, err)
	}
	n := domain.DefaultNumerical()
	n.Solver.AbsTolerance = 2e-10
	c.Numerical = &n
	if _, err = s.SaveConfiguration(c, after.Settings); err != nil {
		t.Fatal(err)
	}
	numerical, err := chartdata.Oscillator(t.Context(), n)
	if err != nil {
		t.Fatal(err)
	}
	numerical.ConfigurationID = c.ID
	if err = s.SaveRun(numerical); err != nil {
		t.Fatal(err)
	}
	previous, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	after, err = s.Load()
	if err != nil || after.Warning != "" || after.Stale || *after.Active.Numerical != n || !reflect.DeepEqual(after.Run, previous.Run) {
		t.Fatalf("numerical restart lost data: %v", err)
	}
	var demoCount int
	if err = s.db.QueryRow("SELECT count(*) FROM runs WHERE result_version='demo-v1'").Scan(&demoCount); err != nil || demoCount != 1 {
		t.Fatal("Stage 1 run lost", err)
	}
	// A numerical run must also be atomic across metadata and series writes.
	if _, err = s.db.Exec(`CREATE TRIGGER fail_numerical BEFORE INSERT ON run_series BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveRun(numerical); err == nil {
		t.Fatal("expected injected save failure")
	}
	after, err = s.Load()
	if err != nil || !reflect.DeepEqual(after.Run, previous.Run) {
		t.Fatal("failed save replaced numerical run", err)
	}
}

func TestRestartRestoresInputsSeriesAndSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := domain.DefaultSettings()
	c := domain.Configuration{Name: "Тест ' SQL", Inputs: domain.DefaultInputs()}
	c.ID, err = s.SaveConfiguration(c, settings)
	if err != nil {
		t.Fatal(err)
	}
	run, err := chartdata.Demo(c.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	run.ConfigurationID = c.ID
	if err = s.SaveRun(run); err != nil {
		t.Fatal(err)
	}
	c.Inputs.Amplitude = 7
	settings.ActiveConfigurationID = c.ID
	settings.Visible[2] = false
	settings.Scales[4] = 2.5
	settings.ResultsWidth = 310
	if _, err = s.SaveConfiguration(c, settings); err != nil {
		t.Fatal(err)
	}
	other := domain.Configuration{Name: "Інша конфігурація", Inputs: domain.DefaultInputs()}
	if _, err = s.SaveConfiguration(other, settings); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Warning != "" || len(state.Configurations) != 2 || state.Active.ID != c.ID || state.Active.Inputs != c.Inputs || !state.Stale || state.Settings != settings {
		t.Fatalf("incorrect restored state: %+v", state)
	}
	if state.Run == nil || state.Run.Snapshot != run.Snapshot || !reflect.DeepEqual(state.Run.Chart, run.Chart) {
		t.Fatal("stored numeric data or immutable snapshot changed")
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("migration version=%d err=%v", version, err)
	}
	var count int
	if err = s.db.QueryRow("SELECT count(*) FROM run_series").Scan(&count); err != nil || count != 1 {
		t.Fatalf("series blocks=%d err=%v", count, err)
	}
}

func TestCorruptAndIncompatibleResultsKeepInputs(t *testing.T) {
	for _, mutation := range []string{
		"UPDATE run_series SET data=x'001122'",
		"UPDATE runs SET result_version='future-v99'",
		"UPDATE runs SET summary_json='{'",
		"DELETE FROM run_series",
	} {
		t.Run(mutation, func(t *testing.T) {
			s, err := Open(filepath.Join(t.TempDir(), "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			inputs := domain.DefaultInputs()
			id, err := s.SaveConfiguration(domain.Configuration{Name: "Тест", Inputs: inputs}, domain.DefaultSettings())
			if err != nil {
				t.Fatal(err)
			}
			run, _ := chartdata.Demo(inputs)
			run.ConfigurationID = id
			if err = s.SaveRun(run); err != nil {
				t.Fatal(err)
			}
			if _, err = s.db.Exec(mutation); err != nil {
				t.Fatal(err)
			}
			state, err := s.Load()
			if err != nil || state.Run != nil || state.Warning == "" || state.Active.Inputs != inputs {
				t.Fatalf("state=%+v err=%v", state, err)
			}
		})
	}
}

func TestFailedRunSaveIsAtomic(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.SaveConfiguration(domain.Configuration{Name: "Тест", Inputs: domain.DefaultInputs()}, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	run, _ := chartdata.Demo(domain.DefaultInputs())
	run.ConfigurationID = id
	if err = s.SaveRun(run); err != nil {
		t.Fatal(err)
	}
	previous, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`CREATE TRIGGER fail_series BEFORE INSERT ON run_series BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveRun(run); err == nil {
		t.Fatal("expected transaction failure")
	}
	state, err := s.Load()
	if err != nil || state.Run == nil || state.Run.ID != previous.Run.ID {
		t.Fatalf("previous result lost: %v", err)
	}
	var count int
	if err = s.db.QueryRow("SELECT count(*) FROM runs").Scan(&count); err != nil || count != 1 {
		t.Fatalf("partial run committed: count=%d err=%v", count, err)
	}
}

func TestNewerDatabaseIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if reopened, err := Open(path); err == nil {
		reopened.Close()
		t.Fatal("opened unsupported schema")
	}
}

package sqlite

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ibsolver/internal/chartdata"
	"ibsolver/internal/domain"
)

func TestResolveDBPathPrefersDatabaseNextToExecutable(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "IBSolver.exe")
	if err := os.WriteFile(exe, []byte("exe"), 0600); err != nil {
		t.Fatal(err)
	}
	portable := filepath.Join(dir, "ibsolver.db")
	if err := os.WriteFile(portable, []byte("db"), 0600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "appdata")
	if got := resolveDBPath(exe, config); got != portable {
		t.Fatalf("with db next to exe: got %q, want %q", got, portable)
	}
	if err := os.Remove(portable); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(config, "IBSolver", "ibsolver.db")
	if got := resolveDBPath(exe, config); got != want {
		t.Fatalf("without db next to exe: got %q, want %q", got, want)
	}
	// A directory named ibsolver.db must not count as a portable database.
	if err := os.Mkdir(portable, 0700); err != nil {
		t.Fatal(err)
	}
	if got := resolveDBPath(exe, config); got != want {
		t.Fatalf("db path is a directory: got %q, want %q", got, want)
	}
}

func TestV1MigrationPreservesNumericalAndDropsNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := domain.Configuration{Name: "Stage 1", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
	settings := domain.DefaultSettings()
	settings.Scales[1] = 2
	c.ID, err = s.SaveConfiguration(c, settings)
	if err != nil {
		t.Fatal(err)
	}
	run, err := chartdata.Oscillator(t.Context(), *c.Parameters.Oscillator)
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
	if _, err = s.db.Exec(`DROP TABLE external_results;
ALTER TABLE configurations DROP COLUMN numerical_json;
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
	if err != nil || after.Warning != "" || after.Stale || *after.Active.Parameters.Oscillator != *c.Parameters.Oscillator || !reflect.DeepEqual(after.Settings, before.Settings) || !reflect.DeepEqual(after.Run, before.Run) || after.Active.Parameters.Oscillator == nil {
		t.Fatalf("v1 migration lost data: %+v %v", after, err)
	}
	n := domain.DefaultNumerical()
	n.Solver.AbsTolerance = 2e-10
	c.Parameters.Oscillator = &n
	c.Mode = domain.ModeOscillator
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
	if err != nil || after.Warning != "" || after.Stale || *after.Active.Parameters.Oscillator != n || !reflect.DeepEqual(after.Run, previous.Run) {
		t.Fatalf("numerical restart lost data: %v", err)
	}
	var runCount int
	if err = s.db.QueryRow("SELECT count(*) FROM runs WHERE result_version='oscillator-rk45-v1'").Scan(&runCount); err != nil || runCount != 2 {
		t.Fatal("numerical runs lost", err)
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
	c := domain.Configuration{Name: "Тест ' SQL", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
	c.ID, err = s.SaveConfiguration(c, settings)
	if err != nil {
		t.Fatal(err)
	}
	run, err := chartdata.Oscillator(t.Context(), *c.Parameters.Oscillator)
	if err != nil {
		t.Fatal(err)
	}
	run.ConfigurationID = c.ID
	if err = s.SaveRun(run); err != nil {
		t.Fatal(err)
	}
	n := *c.Parameters.Oscillator
	n.Model.Mass = 7
	c.Parameters.Oscillator = &n
	settings.ActiveConfigurationID = c.ID
	settings.Visible[2] = false
	settings.Scales[4] = 2.5
	settings.ResultsWidth = 310
	settings.BindSeries(run.Chart.Series)
	if _, err = s.SaveConfiguration(c, settings); err != nil {
		t.Fatal(err)
	}
	other := domain.Configuration{Name: "Інша конфігурація", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
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
	if state.Warning != "" || len(state.Configurations) != 2 || state.Active.ID != c.ID || *state.Active.Parameters.Oscillator != *c.Parameters.Oscillator || !state.Stale || !reflect.DeepEqual(state.Settings, settings) {
		t.Fatalf("incorrect restored state: %+v", state)
	}
	if state.Run == nil || !reflect.DeepEqual(state.Run.Snapshot, run.Snapshot) || !reflect.DeepEqual(state.Run.Chart, run.Chart) {
		t.Fatal("stored numeric data or immutable snapshot changed")
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
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
			inputs := domain.DefaultNumerical()
			id, err := s.SaveConfiguration(domain.Configuration{Name: "Тест", Mode: domain.ModeOscillator, Parameters: domain.Parameters{Oscillator: &inputs}}, domain.DefaultSettings())
			if err != nil {
				t.Fatal(err)
			}
			run, _ := chartdata.Oscillator(t.Context(), inputs)
			run.ConfigurationID = id
			if err = s.SaveRun(run); err != nil {
				t.Fatal(err)
			}
			if _, err = s.db.Exec(mutation); err != nil {
				t.Fatal(err)
			}
			state, err := s.Load()
			if err != nil || state.Run != nil || state.Warning == "" || *state.Active.Parameters.Oscillator != inputs {
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
	id, err := s.SaveConfiguration(domain.Configuration{Name: "Тест", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	run, _ := chartdata.Oscillator(t.Context(), domain.DefaultNumerical())
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

func TestRenameDuplicateDeleteConfiguration(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "configs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	settings := domain.DefaultSettings()
	first := domain.Configuration{Name: "Перша", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
	if first.ID, err = s.SaveConfiguration(first, settings); err != nil {
		t.Fatal(err)
	}
	second := domain.Configuration{Name: "Друга", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
	if second.ID, err = s.SaveConfiguration(second, settings); err != nil {
		t.Fatal(err)
	}
	run, err := chartdata.Oscillator(t.Context(), *second.Parameters.Oscillator)
	if err != nil {
		t.Fatal(err)
	}
	run.ConfigurationID = second.ID
	if err = s.SaveRun(run); err != nil {
		t.Fatal(err)
	}
	ext := domain.ExternalResult{
		ConfigurationID: second.ID, Name: "Зовнішній", ImportedAt: "2026-11-27T00:00:00Z",
		Source: "test.csv", FormatVersion: domain.ExternalFormatVersion,
		Checksum: strings.Repeat("ab", 32), Chart: run.Chart,
	}
	if _, err = s.SaveExternal(ext); err != nil {
		t.Fatal(err)
	}
	state, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	byID := func(id int64) domain.Configuration {
		for _, c := range state.Configurations {
			if c.ID == id {
				return c
			}
		}
		t.Fatalf("configuration %d missing", id)
		return domain.Configuration{}
	}
	if c := byID(second.ID); c.RunsCount != 1 || c.Name != "Друга" {
		t.Fatalf("unexpected loaded configuration: %+v", c)
	}

	// Rename touches only the name.
	if err = s.RenameConfiguration(second.ID, "Друга перейменована"); err != nil {
		t.Fatal(err)
	}
	if err = s.RenameConfiguration(second.ID, "   "); err == nil {
		t.Fatal("expected empty name error")
	}
	if err = s.RenameConfiguration(9999, "Ні"); err == nil {
		t.Fatal("expected missing configuration error")
	}
	state, err = s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c := byID(second.ID); c.Name != "Друга перейменована" || c.RunsCount != 1 || !reflect.DeepEqual(c.Parameters, second.Parameters) {
		t.Fatalf("rename changed more than the name: %+v", c)
	}

	// Duplicate copies parameters under a unique name and activates the copy.
	copyID, err := s.DuplicateConfiguration(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	copy2ID, err := s.DuplicateConfiguration(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DuplicateConfiguration(9999); err == nil {
		t.Fatal("expected missing configuration error")
	}
	state, err = s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Active.ID != copy2ID || state.Active.Name != "Друга перейменована (копія) 2" {
		t.Fatalf("unexpected active copy: %+v", state.Active)
	}
	if c := byID(copyID); c.Name != "Друга перейменована (копія)" || !reflect.DeepEqual(c.Parameters, second.Parameters) {
		t.Fatalf("duplicate lost parameters: %+v", c)
	}

	// Delete cascades over runs, series and external results.
	if err = s.DeleteConfiguration(second.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteConfiguration(9999); err == nil {
		t.Fatal("expected missing configuration error")
	}
	for _, query := range []string{
		"SELECT count(*) FROM runs WHERE configuration_id=?",
		"SELECT count(*) FROM run_series WHERE run_id IN (SELECT id FROM runs WHERE configuration_id=?)",
		"SELECT count(*) FROM external_results WHERE configuration_id=?",
		"SELECT count(*) FROM configurations WHERE id=?",
	} {
		var count int
		if err = s.db.QueryRow(query, second.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("cascade left rows (%s): count=%d err=%v", query, count, err)
		}
	}
	state, err = s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Active.ID != first.ID || len(state.Configurations) != 3 {
		t.Fatalf("active pointer not moved: %+v", state.Active)
	}

	// The last remaining configuration cannot be deleted.
	if err = s.DeleteConfiguration(copyID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteConfiguration(copy2ID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteConfiguration(first.ID); err == nil {
		t.Fatal("expected last configuration error")
	}
	state, err = s.Load()
	if err != nil || len(state.Configurations) != 1 || state.Active.ID != first.ID {
		t.Fatalf("last configuration was deleted: %+v err=%v", state.Configurations, err)
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

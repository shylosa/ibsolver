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

// IBSolver uses exactly one database location: ibsolver.db next to the
// executable. These tests pin that rule and never touch a real working DB.

func fakeAppDir(t *testing.T) (exe string) {
	t.Helper()
	dir := t.TempDir()
	exe = filepath.Join(dir, "IBSolver.exe")
	if err := os.WriteFile(exe, []byte("exe"), 0600); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestDBPathNextToIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "IBSolver.exe")
	if err := os.WriteFile(exe, []byte("exe"), 0600); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "ibsolver.db")
	// Database present, absent, or shadowed by a directory: same path always.
	if got := DBPathNextTo(exe); got != want {
		t.Fatalf("db next to exe: got %q, want %q", got, want)
	}
	if err := os.WriteFile(want, []byte("db"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := DBPathNextTo(exe); got != want {
		t.Fatalf("existing db: got %q, want %q", got, want)
	}
	if err := os.Remove(want); err != nil {
		t.Fatal(err)
	}
	if got := DBPathNextTo(exe); got != want {
		t.Fatalf("missing db: got %q, want %q", got, want)
	}
	if err := os.Mkdir(want, 0700); err != nil {
		t.Fatal(err)
	}
	if got := DBPathNextTo(exe); got != want {
		t.Fatalf("db path is a directory: got %q, want %q", got, want)
	}
	// The process working directory must not influence the result.
	other := t.TempDir()
	t.Chdir(other)
	if got := DBPathNextTo(exe); got != want {
		t.Fatalf("cwd changed: got %q, want %q", got, want)
	}
}

func TestDefaultPathIsNextToExecutableAndIgnoresProfileDirs(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(exe), "ibsolver.db")
	// Decoy databases in the user profile must be ignored and left untouched.
	appData := t.TempDir()
	stale := filepath.Join(appData, "IBSolver", "ibsolver.db")
	if err := os.MkdirAll(filepath.Dir(stale), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale appdata db"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", appData)
	t.Setenv("XDG_CONFIG_HOME", appData)
	t.Chdir(t.TempDir())
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("DefaultPath: got %q, want %q", got, want)
	}
	if strings.Contains(got, appData) {
		t.Fatalf("DefaultPath leaked into profile dir: %q", got)
	}
	content, err := os.ReadFile(stale)
	if err != nil || string(content) != "stale appdata db" {
		t.Fatalf("stale appdata db was touched: %q err=%v", content, err)
	}
}

func TestOpenCreatesAndInitializesDatabaseNextToExecutable(t *testing.T) {
	exe := fakeAppDir(t)
	path := DBPathNextTo(exe)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("db already exists before first open: %v", err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		t.Fatalf("db not created next to exe: %v", err)
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("schema not initialized: version=%d err=%v", version, err)
	}
	for _, table := range []string{"configurations", "runs", "run_series", "app_settings", "external_results"} {
		var name string
		if err = s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
	// The standard initial state must be usable right away.
	c := domain.Configuration{Name: "Перша", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
	c.ID, err = s.SaveConfiguration(c, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Load()
	if err != nil || state.Warning != "" || len(state.Configurations) != 1 || state.Active.ID != c.ID {
		t.Fatalf("fresh state invalid: %+v err=%v", state, err)
	}
}

func TestRestartReusesSameDatabaseNextToExecutable(t *testing.T) {
	exe := fakeAppDir(t)
	path := DBPathNextTo(exe)
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := domain.Configuration{Name: "Збережена", Mode: domain.ModeOscillator, Parameters: domain.DefaultParameters()}
	settings := domain.DefaultSettings()
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
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	// A stale %APPDATA% database must stay untouched across the restart.
	appData := filepath.Join(t.TempDir(), "IBSolver")
	stale := filepath.Join(appData, "ibsolver.db")
	if err = os.MkdirAll(appData, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(stale, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", filepath.Dir(appData))
	t.Chdir(t.TempDir())

	s, err = Open(DBPathNextTo(exe))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Load()
	if err != nil || state.Warning != "" || len(state.Configurations) != 1 || state.Active.Name != "Збережена" {
		t.Fatalf("restart did not restore data: %+v err=%v", state, err)
	}
	if state.Run == nil || !reflect.DeepEqual(state.Run.Chart, run.Chart) {
		t.Fatal("stored run lost across restart")
	}
	entries, err := os.ReadDir(filepath.Dir(exe))
	if err != nil {
		t.Fatal(err)
	}
	dbs := 0
	for _, e := range entries {
		if e.Name() == "ibsolver.db" && !e.IsDir() {
			dbs++
		}
	}
	if dbs != 1 {
		t.Fatalf("expected exactly one ibsolver.db next to exe, found %d", dbs)
	}
	if content, err := os.ReadFile(stale); err != nil || string(content) != "stale" {
		t.Fatalf("stale appdata db was modified: %q err=%v", content, err)
	}
}

func TestOpenFailsExplicitlyWhenAppDirectoryNotWritable(t *testing.T) {
	exe := fakeAppDir(t)
	path := DBPathNextTo(exe)
	// A directory named ibsolver.db makes the location unwritable on any OS.
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	t.Chdir(cwd)
	appData := filepath.Join(t.TempDir(), "IBSolver")
	if err := os.MkdirAll(appData, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", filepath.Dir(appData))

	s, err := Open(path)
	if err == nil {
		s.Close()
		t.Fatal("expected explicit error for unwritable app directory")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error must name the actual db path %q: %v", path, err)
	}
	if !strings.Contains(err.Error(), "права запису") {
		t.Fatalf("error must explain the permission problem: %v", err)
	}
	// No alternative database may be created anywhere.
	for _, decoy := range []string{filepath.Join(cwd, "ibsolver.db"), filepath.Join(appData, "ibsolver.db")} {
		if _, statErr := os.Stat(decoy); statErr == nil {
			t.Fatalf("fallback database created at %q", decoy)
		}
	}
}

func TestOpenNeverCreatesMissingDirectories(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	path := filepath.Join(missing, "ibsolver.db")
	s, err := Open(path)
	if err == nil {
		s.Close()
		t.Fatal("expected error for a database path inside a missing directory")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error must name the actual db path %q: %v", path, err)
	}
	if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
		t.Fatal("Open must not create directories outside the app folder")
	}
}

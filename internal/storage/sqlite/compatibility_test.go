package sqlite

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ibsolver/internal/domain"
)

// These literals have the original Stage 1/2 shapes: bare inputs without an
// envelope, separate numerical parameters, and chart series without IDs.
const legacyDemoInputs = `{"duration":1,"samples":2,"amplitude":2,"frequency":0.5}`
const legacyOscillatorInputs = `{"model":{"mass":1,"stiffness":1,"damping":0,"position":1,"velocity":0,"duration":1,"samples":2},"solver":{"absTolerance":1e-9,"relTolerance":1e-6,"initialStep":0.01,"minStep":1e-10,"maxStep":0.1,"maxSteps":200000}}`
const legacyDemoChart = `{"time":[0,1],"series":[{"name":"Синус","unit":"ум. од.","values":[0,2.449293598294703e-16]},{"name":"Косинус","unit":"ум. од.","values":[4,-4]},{"name":"Затухання","unit":"ум. од.","values":[0,3.658293168363983e-17]},{"name":"Хвиля","unit":"ум. од.","values":[0,9.797174393178827e-17]},{"name":"Імпульс","unit":"ум. од.","values":[1.1253517471925912e-6,1.1253517471925912e-6]},{"name":"Перехід","unit":"ум. од.","values":[0,11.919144636010974]}]}`
const legacyOscillatorChart = `{"time":[0,1],"series":[{"name":"Зміщення","unit":"м","values":[1,0.5403023040120198]},{"name":"Швидкість","unit":"м/с","values":[0,-0.841470982617946]},{"name":"Прискорення","unit":"м/с²","values":[-1,-0.5403023040120198]},{"name":"Кінетична енергія","unit":"Дж","values":[0,0.354036707345019]},{"name":"Потенціальна енергія","unit":"Дж","values":[0.5,0.14596329010475645]},{"name":"Повна енергія","unit":"Дж","values":[0.5,0.49999999744977547]}]}`
const legacyOscillatorSummary = `{"kind":"NUMERICAL","samples":2,"duration":1,"numerical":{"finalPosition":0.5403023040120198,"finalVelocity":-0.841470982617946,"initialEnergy":0.5,"finalEnergy":0.49999999744977547,"statistics":{"accepted":12,"rejected":0,"evaluations":84}}}`
const legacySettings = `{"activeConfigurationId":41,"resultsWidth":290,"visible":[false,true,true,true,true,true],"scales":[2,1,1,1,1,1]}`

func gzipBlob(t *testing.T, chart string) []byte {
	t.Helper()
	var blob bytes.Buffer
	writer := gzip.NewWriter(&blob)
	if _, err := writer.Write([]byte(chart)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return blob.Bytes()
}

// A Stage 2 database with an oscillator configuration (bare demo inputs plus
// a valid numerical block) keeps everything byte-identical across reopens.
func TestLegacyOscillatorPreservedAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	blob := gzipBlob(t, legacyOscillatorChart)
	if _, err = s.db.Exec(`INSERT INTO configurations(id,name,inputs_json,numerical_json,created_at,updated_at) VALUES(41,'Legacy',?,?,'created','updated')`, legacyDemoInputs, legacyOscillatorInputs); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO runs(id,configuration_id,snapshot_json,numerical_snapshot_json,result_version,summary_json,created_at) VALUES(73,41,?,?,?,?,'run-created')`, legacyDemoInputs, legacyOscillatorInputs, domain.NumericalVersion, legacyOscillatorSummary); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO run_series(run_id,data) VALUES(73,?)`, blob); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO app_settings(id,settings_json) VALUES(1,?)`, legacySettings); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TABLE external_results; PRAGMA user_version=2;`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	for pass := 0; pass < 2; pass++ {
		s, err = Open(path)
		if err != nil {
			t.Fatal(err)
		}
		state, err := s.Load()
		if err != nil || state.Warning != "" || state.Run == nil || state.Stale || state.Active.ID != 41 || state.Run.ID != 73 || state.Active.Mode != domain.ModeOscillator || state.Active.LatestRunID == nil || *state.Active.LatestRunID != 73 {
			t.Fatalf("state=%+v error=%v", state, err)
		}
		if state.Active.CreatedAt != "created" || state.Active.UpdatedAt != "updated" || state.Run.CreatedAt != "run-created" {
			t.Fatal("timestamps changed")
		}
		var expected domain.ChartData
		if err = json.Unmarshal([]byte(legacyOscillatorChart), &expected); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(state.Run.Chart.Time, expected.Time) {
			t.Fatal("time changed")
		}
		for i, series := range state.Run.Chart.Series {
			if series.ID != domain.SeriesID(domain.ModeOscillator, i) || series.Reference != nil || !reflect.DeepEqual(series.Values, expected.Series[i].Values) {
				t.Fatal("legacy series changed")
			}
		}
		if state.Settings.Series[domain.SeriesID(domain.ModeOscillator, 0)].Visible || state.Settings.Series[domain.SeriesID(domain.ModeOscillator, 0)].Scale != 2 {
			t.Fatal("legacy preferences lost")
		}
		var stored []byte
		var inputs, snapshot string
		if err = s.db.QueryRow(`SELECT data FROM run_series WHERE run_id=73`).Scan(&stored); err != nil || !bytes.Equal(stored, blob) {
			t.Fatal("chart blob rewritten", err)
		}
		if err = s.db.QueryRow(`SELECT inputs_json FROM configurations WHERE id=41`).Scan(&inputs); err != nil || inputs != legacyDemoInputs {
			t.Fatal("configuration rewritten", err)
		}
		if err = s.db.QueryRow(`SELECT snapshot_json FROM runs WHERE id=73`).Scan(&snapshot); err != nil || snapshot != legacyDemoInputs {
			t.Fatal("snapshot rewritten", err)
		}
		var version int
		if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
			t.Fatalf("version=%d err=%v", version, err)
		}
		s.Close()
	}
}

// The demo mode is gone: a Stage 1 database with demo configuration and run
// is migrated to schema 4 with those rows physically removed.
func TestLegacyDemoRemovedOnMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-demo.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	blob := gzipBlob(t, legacyDemoChart)
	if _, err = s.db.Exec(`INSERT INTO configurations(id,name,inputs_json,numerical_json,created_at,updated_at) VALUES(41,'Legacy',?,'null','created','updated')`, legacyDemoInputs); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO runs(id,configuration_id,snapshot_json,numerical_snapshot_json,result_version,summary_json,created_at) VALUES(73,41,?,'null','demo-v1',?,'run-created')`, legacyDemoInputs, `{"kind":"DEMO","samples":2,"duration":1}`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO run_series(run_id,data) VALUES(73,?)`, blob); err != nil {
		t.Fatal(err)
	}
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
	defer s.Close()
	for query, want := range map[string]int{
		"SELECT count(*) FROM configurations": 0,
		"SELECT count(*) FROM runs":           0,
		"SELECT count(*) FROM run_series":     0,
	} {
		var count int
		if err = s.db.QueryRow(query).Scan(&count); err != nil || count != want {
			t.Fatalf("%s: count=%d want=%d err=%v", query, count, want, err)
		}
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	state, err := s.Load()
	if err != nil || len(state.Configurations) != 0 || state.Run != nil {
		t.Fatalf("demo rows survived migration: %+v err=%v", state, err)
	}
}

func TestNewerDatabaseRejectionDoesNotChangeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(path); err == nil {
		reopened.Close()
		t.Fatal("accepted future schema")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rejected database was modified", err)
	}
}

package sqlite

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"ibsolver/internal/domain"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

// resolveDBPath prefers a portable database placed next to the executable;
// otherwise it falls back to %APPDATA%\IBSolver\ibsolver.db.
func resolveDBPath(executable, userConfigDir string) string {
	candidate := filepath.Join(filepath.Dir(executable), "ibsolver.db")
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return filepath.Join(userConfigDir, "IBSolver", "ibsolver.db")
}

func DefaultPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return resolveDBPath(exe, root), nil
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if _, err = db.Exec("PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err == nil {
		err = s.migrate()
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 4 {
		return fmt.Errorf("База даних створена новішою версією IBSolver")
	}
	if version == 0 {
		_, err = tx.Exec(`
CREATE TABLE configurations (
 id INTEGER PRIMARY KEY, name TEXT NOT NULL, inputs_json TEXT NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE runs (
 id INTEGER PRIMARY KEY, configuration_id INTEGER NOT NULL REFERENCES configurations(id),
 snapshot_json TEXT NOT NULL, result_version TEXT NOT NULL, summary_json TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX runs_configuration ON runs(configuration_id, id DESC);
CREATE TABLE run_series (run_id INTEGER PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE, data BLOB NOT NULL);
CREATE TABLE app_settings (id INTEGER PRIMARY KEY CHECK(id=1), settings_json TEXT NOT NULL);
PRAGMA user_version=1;`)
		if err != nil {
			return err
		}
	}
	if version < 2 {
		if _, err = tx.Exec(`ALTER TABLE configurations ADD COLUMN numerical_json TEXT NOT NULL DEFAULT 'null';
ALTER TABLE runs ADD COLUMN numerical_snapshot_json TEXT NOT NULL DEFAULT 'null';
PRAGMA user_version=2;`); err != nil {
			return err
		}
	}
	if version < 3 {
		if _, err = tx.Exec(`CREATE TABLE external_results (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 configuration_id INTEGER NOT NULL REFERENCES configurations(id),
 name TEXT NOT NULL, imported_at TEXT NOT NULL, source TEXT NOT NULL,
 format_version TEXT NOT NULL, checksum TEXT NOT NULL, chart_json BLOB NOT NULL
);
CREATE INDEX external_results_configuration ON external_results(configuration_id, id DESC);
CREATE TRIGGER external_results_immutable BEFORE UPDATE ON external_results
BEGIN SELECT RAISE(ABORT, 'External results are immutable'); END;
PRAGMA user_version=3;`); err != nil {
			return err
		}
	}
	if version < 4 {
		// The demo mode is removed: drop its configurations, runs, series and
		// external results. Legacy envelope-less rows without numerical data
		// were demo configurations and go away as well.
		if _, err = tx.Exec(`DELETE FROM run_series WHERE run_id IN (SELECT id FROM runs WHERE result_version='demo-v1' OR configuration_id IN (SELECT id FROM configurations WHERE json_extract(inputs_json,'$.mode')='demo'));
DELETE FROM runs WHERE result_version='demo-v1' OR configuration_id IN (SELECT id FROM configurations WHERE json_extract(inputs_json,'$.mode')='demo');
DELETE FROM external_results WHERE configuration_id IN (SELECT id FROM configurations WHERE json_extract(inputs_json,'$.mode')='demo');
DELETE FROM configurations WHERE json_extract(inputs_json,'$.mode')='demo' OR (json_extract(inputs_json,'$.format') IS NULL AND json_extract(numerical_json,'$') IS NULL);
PRAGMA user_version=4;`); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func writeSettings(tx *sql.Tx, settings domain.Settings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO app_settings(id, settings_json) VALUES(1, ?) ON CONFLICT(id) DO UPDATE SET settings_json=excluded.settings_json`, string(data))
	return err
}

// SaveConfiguration keeps inputs independent of previous run snapshots.
func (s *Store) SaveConfiguration(c domain.Configuration, settings domain.Settings) (int64, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}
	if err := settings.Validate(); err != nil {
		return 0, err
	}
	data, err := json.Marshal(parameterRecord{Format: 1, Mode: c.Mode, Parameters: c.Parameters})
	if err != nil {
		return 0, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stamp := now()
	if c.ID == 0 {
		result, e := tx.Exec(`INSERT INTO configurations(name, inputs_json, numerical_json, created_at, updated_at) VALUES(?,?,?,?,?)`, c.Name, string(data), "null", stamp, stamp)
		if e != nil {
			return 0, e
		}
		c.ID, err = result.LastInsertId()
	} else {
		var result sql.Result
		result, err = tx.Exec(`UPDATE configurations SET name=?, inputs_json=?, numerical_json=?, updated_at=? WHERE id=?`, c.Name, string(data), "null", stamp, c.ID)
		if err == nil {
			count, e := result.RowsAffected()
			if e != nil {
				err = e
			} else if count != 1 {
				err = fmt.Errorf("Конфігурацію не знайдено")
			}
		}
	}
	if err != nil {
		return 0, err
	}
	settings.ActiveConfigurationID = c.ID
	if err = writeSettings(tx, settings); err != nil {
		return 0, err
	}
	return c.ID, tx.Commit()
}

func (s *Store) SaveSettings(settings domain.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRow("SELECT 1 FROM configurations WHERE id=?", settings.ActiveConfigurationID).Scan(&exists); err != nil {
		return err
	}
	if err = writeSettings(tx, settings); err != nil {
		return err
	}
	return tx.Commit()
}

// RenameConfiguration changes only the display name; parameters and runs stay untouched.
func (s *Store) RenameConfiguration(id int64, name string) error {
	if err := domain.ValidateName(name); err != nil {
		return err
	}
	result, err := s.db.Exec(`UPDATE configurations SET name=?, updated_at=? WHERE id=?`, name, now(), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("Конфігурацію не знайдено")
	}
	return nil
}

// DeleteConfiguration removes a configuration together with its runs, series
// and external results, then moves the active pointer to the oldest remaining
// configuration. The last configuration cannot be deleted.
func (s *Store) DeleteConfiguration(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var total int
	if err = tx.QueryRow("SELECT count(*) FROM configurations").Scan(&total); err != nil {
		return err
	}
	if total <= 1 {
		return fmt.Errorf("Не можна видалити останню конфігурацію")
	}
	// Child tables reference configurations without ON DELETE CASCADE.
	if _, err = tx.Exec("DELETE FROM run_series WHERE run_id IN (SELECT id FROM runs WHERE configuration_id=?)", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM runs WHERE configuration_id=?", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM external_results WHERE configuration_id=?", id); err != nil {
		return err
	}
	result, err := tx.Exec("DELETE FROM configurations WHERE id=?", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("Конфігурацію не знайдено")
	}
	var nextID int64
	if err = tx.QueryRow("SELECT min(id) FROM configurations").Scan(&nextID); err != nil {
		return err
	}
	if err = setActiveConfiguration(tx, nextID); err != nil {
		return err
	}
	return tx.Commit()
}

// DuplicateConfiguration copies all parameters under a fresh unique name and
// makes the copy active.
func (s *Store) DuplicateConfiguration(id int64) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var name, inputs, numerical string
	err = tx.QueryRow("SELECT name, inputs_json, numerical_json FROM configurations WHERE id=?", id).Scan(&name, &inputs, &numerical)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("Конфігурацію не знайдено")
	}
	if err != nil {
		return 0, err
	}
	copyName, err := uniqueConfigurationName(tx, name+" (копія)")
	if err != nil {
		return 0, err
	}
	stamp := now()
	result, err := tx.Exec(`INSERT INTO configurations(name, inputs_json, numerical_json, created_at, updated_at) VALUES(?,?,?,?,?)`, copyName, inputs, numerical, stamp, stamp)
	if err != nil {
		return 0, err
	}
	newID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err = setActiveConfiguration(tx, newID); err != nil {
		return 0, err
	}
	return newID, tx.Commit()
}

// setActiveConfiguration rewrites the single settings row inside an open transaction.
func setActiveConfiguration(tx *sql.Tx, id int64) error {
	var settingsJSON string
	if err := tx.QueryRow("SELECT settings_json FROM app_settings WHERE id=1").Scan(&settingsJSON); err != nil {
		return err
	}
	var settings domain.Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return err
	}
	settings.ActiveConfigurationID = id
	if err := settings.Validate(); err != nil {
		return err
	}
	return writeSettings(tx, settings)
}

// uniqueConfigurationName appends " 2", " 3", ... until the name is free and
// keeps the result within the 120-rune limit.
func uniqueConfigurationName(tx *sql.Tx, base string) (string, error) {
	if runes := []rune(base); len(runes) > 110 {
		base = string(runes[:110])
	}
	name := base
	for i := 2; ; i++ {
		var exists int
		err := tx.QueryRow("SELECT 1 FROM configurations WHERE name=? LIMIT 1", name).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return name, nil
		}
		if err != nil {
			return "", err
		}
		name = fmt.Sprintf("%s %d", base, i)
	}
}

func (s *Store) SaveRun(run domain.Run) error {
	if err := run.Validate(); err != nil {
		return err
	}
	snapshot, err := json.Marshal(parameterRecord{Format: 1, Mode: run.Mode, Parameters: run.Snapshot})
	if err != nil {
		return err
	}
	summary, err := json.Marshal(run.Summary)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(run.Chart)
	if err != nil {
		return err
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err = writer.Write(raw); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO runs(configuration_id, snapshot_json, numerical_snapshot_json, result_version, summary_json, created_at) VALUES(?,?,?,?,?,?)`, run.ConfigurationID, string(snapshot), "null", run.ResultVersion, string(summary), now())
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO run_series(run_id, data) VALUES(?,?)", id, compressed.Bytes()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Load() (domain.State, error) {
	state := domain.State{Settings: domain.DefaultSettings(), Configurations: []domain.Configuration{}}
	var settings string
	err := s.db.QueryRow("SELECT settings_json FROM app_settings WHERE id=1").Scan(&settings)
	if err == nil {
		candidate := domain.DefaultSettings()
		if json.Unmarshal([]byte(settings), &candidate) != nil || candidate.Validate() != nil {
			state.Warning = "Налаштування пошкоджені; використано стандартні."
		} else {
			state.Settings = candidate
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return state, err
	}
	rows, err := s.db.Query("SELECT id, name, inputs_json, numerical_json, created_at, updated_at, (SELECT max(id) FROM runs WHERE configuration_id=configurations.id), coalesce((SELECT result_version FROM runs WHERE configuration_id=configurations.id ORDER BY id DESC LIMIT 1), ''), (SELECT count(*) FROM runs WHERE configuration_id=configurations.id) FROM configurations ORDER BY id")
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var c domain.Configuration
		var inputs, numerical, version string
		if err = rows.Scan(&c.ID, &c.Name, &inputs, &numerical, &c.CreatedAt, &c.UpdatedAt, &c.LatestRunID, &version, &c.RunsCount); err != nil {
			rows.Close()
			return state, err
		}
		record, decodeErr := decodeParameters(inputs, numerical, "")
		c.Mode, c.Parameters = record.Mode, record.Parameters
		if decodeErr != nil || c.Validate() != nil {
			state.Warning += " Пошкоджену або несумісну конфігурацію пропущено."
			continue
		}
		if c.Parameters.Oscillator == nil {
			c.Parameters.Oscillator = domain.DefaultParameters().Oscillator
		}
		state.Configurations = append(state.Configurations, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	if len(state.Configurations) == 0 {
		return state, nil
	}
	state.Active = state.Configurations[0]
	for _, c := range state.Configurations {
		if c.ID == state.Settings.ActiveConfigurationID {
			state.Active = c
			break
		}
	}
	state.Settings.ActiveConfigurationID = state.Active.ID
	run, err := s.latestRun(state.Active.ID)
	if err != nil {
		state.Warning += " Збережені результати пошкоджені або несумісні; параметри відновлено."
	} else {
		state.Run = run
	}
	state.ExternalResults, err = s.ListExternal(state.Active.ID)
	if err != nil {
		return state, err
	}
	state.Stale = state.Run != nil && !state.Run.Matches(state.Active)
	if state.Run != nil {
		state.Settings.BindSeries(state.Run.Chart.Series)
	}
	return state, nil
}

func (s *Store) latestRun(configurationID int64) (*domain.Run, error) {
	var r domain.Run
	var snapshot, numerical, summary string
	var blob []byte
	err := s.db.QueryRow(`SELECT r.id, r.configuration_id, r.snapshot_json, r.numerical_snapshot_json, r.result_version, r.summary_json, r.created_at, s.data
FROM runs r LEFT JOIN run_series s ON s.run_id=r.id WHERE r.configuration_id=? ORDER BY r.id DESC LIMIT 1`, configurationID).Scan(&r.ID, &r.ConfigurationID, &snapshot, &numerical, &r.ResultVersion, &summary, &r.CreatedAt, &blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	record, err := decodeParameters(snapshot, numerical, r.ResultVersion)
	if err != nil {
		return nil, err
	}
	r.Mode, r.Snapshot = record.Mode, record.Parameters
	if err = json.Unmarshal([]byte(summary), &r.Summary); err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	const limit = 20 << 20
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, fmt.Errorf("Завеликий результат")
	}
	if err = json.Unmarshal(raw, &r.Chart); err != nil {
		return nil, err
	}
	for i := range r.Chart.Series {
		if r.Chart.Series[i].ID == "" {
			r.Chart.Series[i].ID = domain.SeriesID(r.Mode, i)
		}
	}
	if err = r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"ibsolver/internal/domain"
)

// SaveExternal inserts one complete record atomically; there is no update path.
func (s *Store) SaveExternal(r domain.ExternalResult) (int64, error) {
	if r.ID != 0 {
		return 0, fmt.Errorf("Збережений результат не можна змінити")
	}
	if err := r.Validate(); err != nil {
		return 0, err
	}
	data, err := json.Marshal(r.Chart)
	if err != nil {
		return 0, err
	}
	result, err := s.db.Exec(`INSERT INTO external_results(configuration_id,name,imported_at,source,format_version,checksum,chart_json) VALUES(?,?,?,?,?,?,?)`, r.ConfigurationID, r.Name, r.ImportedAt, r.Source, r.FormatVersion, r.Checksum, data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func scanExternal(row interface{ Scan(...any) error }, r *domain.ExternalResult, extra ...any) error {
	fields := []any{&r.ID, &r.ConfigurationID, &r.Name, &r.ImportedAt, &r.Source, &r.FormatVersion, &r.Checksum}
	return row.Scan(append(fields, extra...)...)
}

const externalColumns = "id,configuration_id,name,imported_at,source,format_version,checksum"

func (s *Store) ListExternal(configurationID int64) ([]domain.ExternalResult, error) {
	results := []domain.ExternalResult{}
	rows, err := s.db.Query("SELECT "+externalColumns+" FROM external_results WHERE configuration_id=? ORDER BY id DESC", configurationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r domain.ExternalResult
		if err := scanExternal(rows, &r); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) OpenExternal(configurationID, id int64) (domain.ExternalResult, error) {
	var r domain.ExternalResult
	var data []byte
	err := scanExternal(s.db.QueryRow("SELECT "+externalColumns+",chart_json FROM external_results WHERE configuration_id=? AND id=?", configurationID, id), &r, &data)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(data, &r.Chart); err != nil {
		return r, err
	}
	return r, r.Validate()
}

func (s *Store) DeleteExternal(configurationID, id int64) error {
	result, err := s.db.Exec("DELETE FROM external_results WHERE configuration_id=? AND id=?", configurationID, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count != 1 {
		return sql.ErrNoRows
	}
	return err
}

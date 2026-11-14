package domain

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const ExternalFormatVersion = "csv-v1"

// ExternalResult has no model or parameter snapshot. Its numbers are immutable.
type ExternalResult struct {
	ID              int64     `json:"id"`
	ConfigurationID int64     `json:"configurationId"`
	Name            string    `json:"name"`
	ImportedAt      string    `json:"importedAt"`
	Source          string    `json:"source"`
	FormatVersion   string    `json:"formatVersion"`
	Checksum        string    `json:"checksum"`
	Chart           ChartData `json:"chart"`
}

func (r ExternalResult) Validate() error {
	hash, err := hex.DecodeString(r.Checksum)
	if err != nil || len(hash) != 32 || r.ConfigurationID <= 0 || r.ID < 0 ||
		strings.TrimSpace(r.Name) == "" || len([]rune(r.Name)) > 120 ||
		strings.TrimSpace(r.Source) == "" || r.FormatVersion != ExternalFormatVersion {
		return fmt.Errorf("Некоректні метадані зовнішнього результату")
	}
	if _, err := time.Parse(time.RFC3339Nano, r.ImportedAt); err != nil {
		return fmt.Errorf("Некоректна дата імпорту")
	}
	return r.Chart.Validate()
}

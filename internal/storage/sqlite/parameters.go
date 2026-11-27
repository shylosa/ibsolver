package sqlite

import (
	"encoding/json"
	"fmt"

	"ibsolver/internal/domain"
)

// The existing JSON columns hold a versioned typed envelope. Legacy columns
// are read without rewriting configuration rows, snapshots or chart blobs.
type parameterRecord struct {
	Format     int               `json:"format"`
	Mode       domain.Mode       `json:"mode"`
	Parameters domain.Parameters `json:"parameters"`
}

func decodeParameters(raw, numerical, resultVersion string) (parameterRecord, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return parameterRecord{}, err
	}
	if _, exists := fields["format"]; exists {
		var record parameterRecord
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			return record, err
		}
		if record.Format != 1 {
			return record, fmt.Errorf("Несумісна версія параметрів")
		}
		return record, record.Parameters.Validate(record.Mode)
	}
	var n *domain.NumericalExample
	if err := json.Unmarshal([]byte(numerical), &n); err != nil {
		return parameterRecord{}, err
	}
	// Legacy rows without an envelope are oscillator configurations when the
	// saved numerical block is valid; former demo rows are rejected (the demo
	// mode no longer exists). Snapshots supply resultVersion because their
	// historical mode is known.
	if resultVersion == domain.NumericalVersion || (resultVersion == "" && n != nil && n.Validate() == nil) {
		p := domain.Parameters{Oscillator: n}
		if err := p.Validate(domain.ModeOscillator); err != nil {
			return parameterRecord{}, err
		}
		return parameterRecord{Mode: domain.ModeOscillator, Parameters: p}, nil
	}
	return parameterRecord{}, fmt.Errorf("Застарілі дані без чисельних параметрів не підтримуються")
}

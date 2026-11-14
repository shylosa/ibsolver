package domain

import (
	"fmt"
	"math"
	"strings"
)

const ResultVersion = "demo-v1"

type Inputs struct {
	Duration  float64 `json:"duration"`
	Samples   int     `json:"samples"`
	Amplitude float64 `json:"amplitude"`
	Frequency float64 `json:"frequency"`
}

func DefaultInputs() Inputs { return Inputs{Duration: 10, Samples: 501, Amplitude: 1, Frequency: 0.5} }

func (v Inputs) Validate() error {
	for _, n := range []float64{v.Duration, v.Amplitude, v.Frequency} {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return fmt.Errorf("Параметри мають бути скінченними числами")
		}
	}
	if v.Duration <= 0 || v.Duration > 10000 || v.Samples < 2 || v.Samples > 20000 || v.Amplitude <= 0 || v.Amplitude > 1e6 || v.Frequency <= 0 || v.Frequency > 1000 {
		return fmt.Errorf("Параметри DEMO поза допустимими межами")
	}
	return nil
}

type Configuration struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Inputs    Inputs            `json:"inputs"`
	Numerical *NumericalExample `json:"numerical,omitempty"`
	CreatedAt string            `json:"createdAt"`
	UpdatedAt string            `json:"updatedAt"`
}

func (c Configuration) Validate() error {
	if strings.TrimSpace(c.Name) == "" || len([]rune(c.Name)) > 120 {
		return fmt.Errorf("Назва має містити від 1 до 120 символів")
	}
	if c.ID < 0 {
		return fmt.Errorf("Невірний ідентифікатор конфігурації")
	}
	if err := c.Inputs.Validate(); err != nil {
		return err
	}
	if c.Numerical != nil {
		return c.Numerical.Validate()
	}
	return nil
}

type Series struct {
	Name   string    `json:"name"`
	Unit   string    `json:"unit"`
	Values []float64 `json:"values"`
}

type ChartData struct {
	Time   []float64 `json:"time"`
	Series []Series  `json:"series"`
}

func (d ChartData) Validate() error {
	if len(d.Time) < 2 || len(d.Time) > 20000 || len(d.Series) < 1 || len(d.Series) > 6 {
		return fmt.Errorf("Невірний розмір часових рядів")
	}
	for i, t := range d.Time {
		if math.IsNaN(t) || math.IsInf(t, 0) || (i > 0 && t <= d.Time[i-1]) {
			return fmt.Errorf("Невірна часова вісь")
		}
	}
	for _, s := range d.Series {
		if s.Name == "" || len(s.Values) != len(d.Time) {
			return fmt.Errorf("Невірна серія")
		}
		for _, n := range s.Values {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return fmt.Errorf("Невірне числове значення")
			}
		}
	}
	return nil
}

type Summary struct {
	Kind      string            `json:"kind"`
	Samples   int               `json:"samples"`
	Duration  float64           `json:"duration"`
	Numerical *NumericalSummary `json:"numerical,omitempty"`
}

type Run struct {
	ID                int64             `json:"id"`
	ConfigurationID   int64             `json:"configurationId"`
	Snapshot          Inputs            `json:"snapshot"`
	NumericalSnapshot *NumericalExample `json:"numericalSnapshot,omitempty"`
	ResultVersion     string            `json:"resultVersion"`
	Summary           Summary           `json:"summary"`
	Chart             ChartData         `json:"chart"`
	CreatedAt         string            `json:"createdAt"`
}

func (r Run) Validate() error {
	if r.ResultVersion == NumericalVersion {
		return r.validateNumerical()
	}
	if r.ResultVersion != ResultVersion || r.Summary.Kind != "DEMO" {
		return fmt.Errorf("Несумісна версія результатів")
	}
	if err := r.Snapshot.Validate(); err != nil {
		return err
	}
	if err := r.Chart.Validate(); err != nil {
		return err
	}
	if r.Summary.Samples != len(r.Chart.Time) || r.Summary.Samples != r.Snapshot.Samples || r.Summary.Duration != r.Snapshot.Duration {
		return fmt.Errorf("Неповні результати")
	}
	return nil
}

func (r Run) validateNumerical() error {
	if r.Summary.Kind != "NUMERICAL" || r.NumericalSnapshot == nil || r.Summary.Numerical == nil {
		return fmt.Errorf("Неповний чисельний результат")
	}
	if err := r.NumericalSnapshot.Validate(); err != nil {
		return err
	}
	if err := r.Chart.Validate(); err != nil {
		return err
	}
	p, s := r.NumericalSnapshot.Model, r.Summary.Numerical
	if r.Summary.Samples != p.Samples || len(r.Chart.Time) != p.Samples || r.Summary.Duration != p.Duration || r.Chart.Time[0] != 0 || r.Chart.Time[len(r.Chart.Time)-1] != p.Duration || len(r.Chart.Series) != 6 {
		return fmt.Errorf("Неповні чисельні ряди")
	}
	for _, v := range []float64{s.FinalPosition, s.FinalVelocity, s.InitialEnergy, s.FinalEnergy} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("Некоректні чисельні підсумки")
		}
	}
	stats := s.Statistics
	if stats.Accepted < p.Samples-1 || stats.Rejected < 0 || stats.Accepted > r.NumericalSnapshot.Solver.MaxSteps || stats.Rejected > r.NumericalSnapshot.Solver.MaxSteps-stats.Accepted || stats.Evaluations != 7*(stats.Accepted+stats.Rejected) {
		return fmt.Errorf("Некоректна статистика інтегратора")
	}
	return nil
}

type Settings struct {
	ActiveConfigurationID int64      `json:"activeConfigurationId"`
	ResultsWidth          int        `json:"resultsWidth"`
	Visible               [6]bool    `json:"visible"`
	Scales                [6]float64 `json:"scales"`
}

func DefaultSettings() Settings {
	return Settings{ResultsWidth: 240, Visible: [6]bool{true, true, true, true, true, true}, Scales: [6]float64{1, 1, 1, 1, 1, 1}}
}

func (s Settings) Validate() error {
	if s.ResultsWidth < 180 || s.ResultsWidth > 480 {
		return fmt.Errorf("Невірна ширина панелі")
	}
	for _, v := range s.Scales {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0.1 || v > 10 {
			return fmt.Errorf("Масштаб має бути від 0,1 до 10")
		}
	}
	return nil
}

type State struct {
	Configurations []Configuration `json:"configurations"`
	Active         Configuration   `json:"active"`
	Settings       Settings        `json:"settings"`
	Run            *Run            `json:"run"`
	Warning        string          `json:"warning"`
	Stale          bool            `json:"stale"`
}

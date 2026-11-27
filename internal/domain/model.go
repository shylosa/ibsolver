package domain

import (
	"fmt"
	"math"
	"strings"

	"ibsolver/internal/prepcalc"
)

const ResultVersion = "demo-v1"
const BallisticsVersion = "ballistics-v1"

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

type Mode string

const (
	ModePreparation Mode = "preparation"
	ModeDemo        Mode = "demo"
	ModeOscillator  Mode = "oscillator"
)

// Parameters retains each mode's values when switching the active mode.
// A nil pointer means absent; zero remains an actual numeric value.
type Parameters struct {
	Preparation         *PreparationInputs         `json:"preparation,omitempty"`
	PreparationWorkbook *PreparationWorkbookInputs `json:"preparationWorkbook,omitempty"`
	Demo                *Inputs                    `json:"demo,omitempty"`
	Oscillator          *NumericalExample          `json:"oscillator,omitempty"`
}

func DefaultParameters() Parameters {
	d, n := DefaultInputs(), DefaultNumerical()
	return Parameters{Demo: &d, Oscillator: &n}
}

func (p Parameters) Validate(mode Mode) error {
	if p.PreparationWorkbook != nil {
		if err := p.PreparationWorkbook.Validate(); err != nil {
			return err
		}
	}
	switch mode {
	case ModePreparation:
		if p.Preparation != nil {
			return p.Preparation.Validate()
		}
	case ModeDemo:
		if p.Demo != nil {
			return p.Demo.Validate()
		}
	case ModeOscillator:
		if p.Oscillator != nil {
			return p.Oscillator.Validate()
		}
	default:
		return fmt.Errorf("Невідомий режим конфігурації")
	}
	return fmt.Errorf("Відсутні параметри активного режиму")
}

type Configuration struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Mode        Mode       `json:"mode"`
	Parameters  Parameters `json:"parameters"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
	LatestRunID *int64     `json:"latestRunId,omitempty"`
}

func (c Configuration) Validate() error {
	if strings.TrimSpace(c.Name) == "" || len([]rune(c.Name)) > 120 {
		return fmt.Errorf("Назва має містити від 1 до 120 символів")
	}
	if c.ID < 0 {
		return fmt.Errorf("Невірний ідентифікатор конфігурації")
	}
	return c.Parameters.Validate(c.Mode)
}

type Series struct {
	ID        string    `json:"id"`
	Reference *float64  `json:"reference,omitempty"`
	Name      string    `json:"name"`
	Unit      string    `json:"unit"`
	Values    []float64 `json:"values"`
}

type ChartData struct {
	Time   []float64 `json:"time"`
	Series []Series  `json:"series"`
	// Aux holds non-displayed numeric columns of a run, such as the ballistics
	// burnt-layer series, exported to CSV but kept out of the charted series.
	Aux []Series `json:"aux,omitempty"`
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
	ids := make(map[string]bool)
	if err := validateSeries(d.Time, d.Series, ids); err != nil {
		return err
	}
	return validateSeries(d.Time, d.Aux, ids)
}

func validateSeries(time []float64, series []Series, ids map[string]bool) error {
	for _, s := range series {
		if s.ID == "" || ids[s.ID] {
			return fmt.Errorf("Невірний ідентифікатор серії")
		}
		ids[s.ID] = true
		if s.Reference != nil && (math.IsNaN(*s.Reference) || math.IsInf(*s.Reference, 0) || *s.Reference <= 0) {
			return fmt.Errorf("Опорне значення серії має бути скінченним і додатним")
		}
		if s.Name == "" || len(s.Values) != len(time) {
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

type BallisticsSummary struct {
	MaxPressure    float64 `json:"maxPressure"`    // MPa
	MuzzleVelocity float64 `json:"muzzleVelocity"` // m/s
	MuzzleTime     float64 `json:"muzzleTime"`     // s
	BarrelTravel   float64 `json:"barrelTravel"`   // m
	BurntFraction  float64 `json:"burntFraction"`  // ETAk
	ChamberVolume  float64 `json:"chamberVolume"`  // m^3
	ChargeMass     float64 `json:"chargeMass"`     // kg
	BarrelMass     float64 `json:"barrelMass"`     // kg
}

type Summary struct {
	ConfigurationName string             `json:"configurationName,omitempty"`
	Kind              string             `json:"kind"`
	Samples           int                `json:"samples"`
	Duration          float64            `json:"duration"`
	Numerical         *NumericalSummary  `json:"numerical,omitempty"`
	Ballistics        *BallisticsSummary `json:"ballistics,omitempty"`
	Workbook          *prepcalc.Result   `json:"workbook,omitempty"`
}

type Run struct {
	ID              int64      `json:"id"`
	ConfigurationID int64      `json:"configurationId"`
	Mode            Mode       `json:"mode"`
	Snapshot        Parameters `json:"snapshot"`
	ResultVersion   string     `json:"resultVersion"`
	Summary         Summary    `json:"summary"`
	Chart           ChartData  `json:"chart"`
	CreatedAt       string     `json:"createdAt"`
}

func (r Run) Validate() error {
	if r.ResultVersion == NumericalVersion && r.Mode == ModeOscillator {
		return r.validateNumerical()
	}
	if r.ResultVersion == BallisticsVersion && r.Mode == ModePreparation {
		return r.validateBallistics()
	}
	if r.ResultVersion == WorkbookVersion && r.Mode == ModePreparation {
		return r.validateWorkbook()
	}
	if r.ResultVersion != ResultVersion || r.Mode != ModeDemo || r.Summary.Kind != "DEMO" {
		return fmt.Errorf("Несумісна версія результатів")
	}
	if err := r.Snapshot.Validate(r.Mode); err != nil {
		return err
	}
	if err := r.Chart.Validate(); err != nil {
		return err
	}
	if r.Summary.Samples != len(r.Chart.Time) || r.Summary.Samples != r.Snapshot.Demo.Samples || r.Summary.Duration != r.Snapshot.Demo.Duration {
		return fmt.Errorf("Неповні результати")
	}
	return nil
}

func (r Run) validateBallistics() error {
	if r.Summary.Kind != "BALLISTICS" || r.Snapshot.Preparation == nil || r.Summary.Ballistics == nil {
		return fmt.Errorf("Неповний балістичний результат")
	}
	if err := r.Snapshot.Validate(r.Mode); err != nil {
		return err
	}
	if err := r.Chart.Validate(); err != nil {
		return err
	}
	s := r.Summary.Ballistics
	if r.Summary.Samples != len(r.Chart.Time) || len(r.Chart.Series) != 6 {
		return fmt.Errorf("Неповні балістичні ряди")
	}
	for _, v := range []float64{s.MaxPressure, s.MuzzleVelocity, s.MuzzleTime, s.BarrelTravel, s.BurntFraction, s.ChamberVolume, s.ChargeMass, s.BarrelMass} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("Некоректні балістичні підсумки")
		}
	}
	return nil
}

func (r Run) validateNumerical() error {
	if r.Summary.Kind != "NUMERICAL" || r.Snapshot.Oscillator == nil || r.Summary.Numerical == nil {
		return fmt.Errorf("Неповний чисельний результат")
	}
	if err := r.Snapshot.Validate(r.Mode); err != nil {
		return err
	}
	if err := r.Chart.Validate(); err != nil {
		return err
	}
	p, s := r.Snapshot.Oscillator.Model, r.Summary.Numerical
	if r.Summary.Samples != p.Samples || len(r.Chart.Time) != p.Samples || r.Summary.Duration != p.Duration || r.Chart.Time[0] != 0 || r.Chart.Time[len(r.Chart.Time)-1] != p.Duration || len(r.Chart.Series) != 6 {
		return fmt.Errorf("Неповні чисельні ряди")
	}
	for _, v := range []float64{s.FinalPosition, s.FinalVelocity, s.InitialEnergy, s.FinalEnergy} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("Некоректні чисельні підсумки")
		}
	}
	stats := s.Statistics
	if stats.Accepted < p.Samples-1 || stats.Rejected < 0 || stats.Accepted > r.Snapshot.Oscillator.Solver.MaxSteps || stats.Rejected > r.Snapshot.Oscillator.Solver.MaxSteps-stats.Accepted || stats.Evaluations != 7*(stats.Accepted+stats.Rejected) {
		return fmt.Errorf("Некоректна статистика інтегратора")
	}
	return nil
}

type SeriesSettings struct {
	Visible bool    `json:"visible"`
	Scale   float64 `json:"scale"`
}

type Settings struct {
	Series                map[string]SeriesSettings `json:"series,omitempty"`
	ActiveConfigurationID int64                     `json:"activeConfigurationId"`
	ResultsWidth          int                       `json:"resultsWidth"`
	Visible               [6]bool                   `json:"visible"`
	Scales                [6]float64                `json:"scales"`
}

func DefaultSettings() Settings {
	return Settings{ResultsWidth: 240, Visible: [6]bool{true, true, true, true, true, true}, Scales: [6]float64{1, 1, 1, 1, 1, 1}}
}

func (s Settings) Validate() error {
	if s.ResultsWidth < 180 || s.ResultsWidth > 480 {
		return fmt.Errorf("Невірна ширина панелі")
	}
	scales := append([]float64(nil), s.Scales[:]...)
	for id, pref := range s.Series {
		if id == "" {
			return fmt.Errorf("Відсутній ідентифікатор налаштувань серії")
		}
		scales = append(scales, pref.Scale)
	}
	for _, v := range scales {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0.1 || v > 10 {
			return fmt.Errorf("Масштаб має бути від 0,1 до 10")
		}
	}
	return nil
}

type State struct {
	ExternalResults []ExternalResult `json:"externalResults"`
	Configurations  []Configuration  `json:"configurations"`
	Active          Configuration    `json:"active"`
	Settings        Settings         `json:"settings"`
	Run             *Run             `json:"run"`
	Warning         string           `json:"warning"`
	Stale           bool             `json:"stale"`
}

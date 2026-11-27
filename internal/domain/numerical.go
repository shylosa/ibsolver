package domain

import (
	"fmt"
	"math"

	"ibsolver/internal/ode"
)

const NumericalVersion = "oscillator-rk45-v1"

// OscillatorParameters uses SI units: kg, N/m, kg/s, m, m/s and seconds.
type OscillatorParameters struct {
	Mass      float64 `json:"mass"`
	Stiffness float64 `json:"stiffness"`
	Damping   float64 `json:"damping"`
	Position  float64 `json:"position"`
	Velocity  float64 `json:"velocity"`
	Duration  float64 `json:"duration"`
	Samples   int     `json:"samples"`
}

type NumericalExample struct {
	Model  OscillatorParameters `json:"model"`
	Solver ode.Options          `json:"solver"`
}

func DefaultNumerical() NumericalExample {
	return NumericalExample{Model: OscillatorParameters{Mass: 1, Stiffness: 4, Damping: 0.3, Position: 1, Duration: 10, Samples: 501}, Solver: ode.DefaultOptions()}
}

func (n NumericalExample) Validate() error {
	p := n.Model
	for _, v := range []float64{p.Mass, p.Stiffness, p.Damping, p.Position, p.Velocity, p.Duration} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("Параметри осцилятора мають бути скінченними")
		}
	}
	if p.Mass <= 0 || p.Stiffness <= 0 || p.Damping < 0 || p.Duration <= 0 || p.Duration > 10000 || p.Samples < 2 || p.Samples > 20000 {
		return fmt.Errorf("Потрібні m > 0, k > 0, c ≥ 0, 0 < час ≤ 10000 с та 2…20000 точок")
	}
	if err := n.Solver.Validate(); err != nil {
		return fmt.Errorf("Перевірте точність та кроки інтегратора: %w", err)
	}
	if n.Solver.MaxSteps > 1000000 {
		return fmt.Errorf("Ліміт інтегратора — до 1000000 спроб кроку")
	}
	return nil
}

type NumericalSummary struct {
	FinalPosition float64        `json:"finalPosition"`
	FinalVelocity float64        `json:"finalVelocity"`
	InitialEnergy float64        `json:"initialEnergy"`
	FinalEnergy   float64        `json:"finalEnergy"`
	Statistics    ode.Statistics `json:"statistics"`
}

func (r Run) Matches(c Configuration) bool {
	if r.Mode != c.Mode {
		return false
	}
	if r.Mode == ModeDemo {
		return r.Snapshot.Demo != nil && c.Parameters.Demo != nil && *r.Snapshot.Demo == *c.Parameters.Demo
	}
	if r.Mode == ModeOscillator {
		return r.Snapshot.Oscillator != nil && c.Parameters.Oscillator != nil && *r.Snapshot.Oscillator == *c.Parameters.Oscillator
	}
	if r.Mode == ModePreparation {
		if r.ResultVersion == WorkbookVersion {
			return r.Snapshot.PreparationWorkbook != nil && c.Parameters.PreparationWorkbook != nil &&
				r.Snapshot.PreparationWorkbook.Equal(*c.Parameters.PreparationWorkbook)
		}
		return r.Snapshot.Preparation != nil && c.Parameters.Preparation != nil && r.Snapshot.Preparation.Equal(*c.Parameters.Preparation)
	}
	return false
}

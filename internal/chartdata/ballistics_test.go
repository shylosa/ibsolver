package chartdata

import (
	"context"
	"math"
	"testing"

	"ibsolver/internal/domain"
)

func TestBallistics_DefaultParameters(t *testing.T) {
	ctx := context.Background()
	input := domain.PreparationInputs{}

	run, err := Ballistics(ctx, input)
	if err != nil {
		t.Fatalf("Ballistics failed: %v", err)
	}

	if err := run.Validate(); err != nil {
		t.Fatalf("Run validation failed: %v", err)
	}

	if run.Summary.Kind != "BALLISTICS" {
		t.Errorf("Expected kind BALLISTICS, got %s", run.Summary.Kind)
	}

	if len(run.Chart.Series) != 6 {
		t.Fatalf("Expected 6 series, got %d", len(run.Chart.Series))
	}

	summary := run.Summary.Ballistics
	if summary == nil {
		t.Fatal("Expected BallisticsSummary, got nil")
	}

	// OZVB-3A expected results
	if summary.MaxPressure < 250 || summary.MaxPressure > 320 {
		t.Errorf("MaxPressure out of expected range: %g MPa", summary.MaxPressure)
	}

	if summary.MuzzleVelocity < 780 || summary.MuzzleVelocity > 840 {
		t.Errorf("MuzzleVelocity out of expected range: %g m/s", summary.MuzzleVelocity)
	}

	if summary.MuzzleTime <= 0 || summary.MuzzleTime > 0.1 {
		t.Errorf("MuzzleTime unexpected: %g s", summary.MuzzleTime)
	}

	// Verify all points are finite and non-empty
	for i, tm := range run.Chart.Time {
		if math.IsNaN(tm) || math.IsInf(tm, 0) {
			t.Fatalf("Non-finite time at %d", i)
		}
		if i > 0 && tm <= run.Chart.Time[i-1] {
			t.Fatalf("Time not strictly increasing at %d", i)
		}
		for sIdx, s := range run.Chart.Series {
			val := s.Values[i]
			if math.IsNaN(val) || math.IsInf(val, 0) {
				t.Fatalf("Non-finite value in series %s at %d", s.Name, i)
			}
			if sIdx == 0 && val < 0 {
				t.Errorf("Negative pressure at %d: %g", i, val)
			}
		}
	}

	// Aux series E (mm) and Z (relative burnt thickness) mirror the integration
	// state of the burnt layer: finite, non-negative and non-decreasing.
	if len(run.Chart.Aux) != 2 {
		t.Fatalf("Expected 2 aux series (E, Z), got %d", len(run.Chart.Aux))
	}
	for _, s := range run.Chart.Aux {
		if len(s.Values) != len(run.Chart.Time) {
			t.Fatalf("Aux series %s length mismatch", s.ID)
		}
		for i, val := range s.Values {
			if math.IsNaN(val) || math.IsInf(val, 0) || val < 0 {
				t.Fatalf("Invalid aux value in %s at %d: %g", s.ID, i, val)
			}
			if i > 0 && val < s.Values[i-1] {
				t.Fatalf("Aux series %s decreased at %d: %g -> %g", s.ID, i, s.Values[i-1], val)
			}
		}
	}
	if run.Chart.Aux[0].Values[0] != 0 || run.Chart.Aux[1].Values[0] != 0 {
		t.Fatalf("Aux series must start at zero, got %g and %g", run.Chart.Aux[0].Values[0], run.Chart.Aux[1].Values[0])
	}
}

func TestBallistics_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	input := domain.PreparationInputs{}
	_, err := Ballistics(ctx, input)
	if err == nil {
		t.Fatal("Expected cancellation error, got nil")
	}
}

// TestBallistics_RPK74LegacyMetrics checks the extra summary metrics against
// the historical DIPLOM.BAS RPK-74 screen (refs/DIPLOM.BAS lines 70-105):
// Pd=116.1 MPa, TPmax=0.55 ms, Delta=839.8 kg/m^3, Lkan=42.35 mm, Lm=49.22 mm,
// Lad=3.50, Kisp=0.866 MJ/kg, Kzar=0.606, dv=10.07 %. Tolerance covers the
// different integrator (fixed-step RK4 there, fixed-step RK4 here with a
// different step) and the rounded screen output.
func TestBallistics_RPK74LegacyMetrics(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	forma := 4
	input := domain.PreparationInputs{
		Pmax: f(440e6), Vd: f(960), D: f(5.45e-3), Q: f(3.4e-3), Ns: f(0.83),
		Hi: f(2.5), Pfor: f(45e6), Af: f(1.1), Lc: f(0.413),
		Kt: f(1.8e-3), Tn: f(15), Forma: &forma, Omega: f(2.192e-3), W0: f(2.61e-6),
		Ro: f(1600), Tg15: f(2900), Ku1: f(0), F1: f(0.87e6), U15: f(1.11e-9),
		Svod: f(0.37 / 2000), Pvos: f(5e6), Tvos: f(1600),
	}
	run, err := Ballistics(context.Background(), input)
	if err != nil {
		t.Fatalf("Ballistics failed: %v", err)
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("Run validation failed: %v", err)
	}
	s := run.Summary.Ballistics
	const tol = 0.015
	check := func(name string, got *float64, want float64) {
		t.Helper()
		if got == nil {
			t.Errorf("%s: nil, want %.6g", name, want)
			return
		}
		if d := math.Abs(*got-want) / math.Abs(want); d > tol {
			t.Errorf("%s = %.6g, want %.6g (rel diff %.3f)", name, *got, want, d)
		}
	}
	check("MuzzlePressure(Pd) MPa", s.MuzzlePressure, 116.1)
	check("MaxPressureTime(TPmax) s", s.MaxPressureTime, 0.55e-3)
	check("ChargeDensity(Delta)", s.ChargeDensity, 839.8)
	check("ChamberLength(Lkan) m", s.ChamberLength, 42.35e-3)
	check("PeakTravel(Lm) m", s.PeakTravel, 49.22e-3)
	check("ExpansionRatio(Lad)", s.ExpansionRatio, 3.50)
	check("Kisp MJ/kg", s.Kisp, 0.866)
	check("Kzar", s.Kzar, 0.606)
	check("VelocityDeviation(dv) %", s.VelocityDeviation, 10.07)
}

func TestBallistics_UsesPreparationOverrides(t *testing.T) {
	omega := 10.0

	base, err := Ballistics(context.Background(), domain.PreparationInputs{})
	if err != nil {
		t.Fatalf("base calculation: %v", err)
	}
	changed, err := Ballistics(context.Background(), domain.PreparationInputs{
		Omega: &omega,
	})
	if err != nil {
		t.Fatalf("override calculation: %v", err)
	}
	if base.Summary.Ballistics == nil || changed.Summary.Ballistics == nil {
		t.Fatal("missing ballistics summary")
	}
	if changed.Summary.Ballistics.ChargeMass != omega {
		t.Fatalf("changed charge mass not reflected: expected=%g changed=%g",
			omega, changed.Summary.Ballistics.ChargeMass)
	}
}

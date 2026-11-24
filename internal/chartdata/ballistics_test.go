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

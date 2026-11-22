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

	if summary.MuzzleVelocity < 850 || summary.MuzzleVelocity > 980 {
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

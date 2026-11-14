package ode

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

func grid(end float64, n int) []float64 {
	x := make([]float64, n)
	for i := range x {
		x[i] = end * float64(i) / float64(n-1)
	}
	x[n-1] = end
	return x
}

func TestAnalyticalSolutions(t *testing.T) {
	// Time: s. Decay state: dimensionless. Oscillator states: x (m), v (m/s).
	const omega, alpha, x0, v0 = 2.0, 0.2, 1.0, -0.4
	wd := math.Sqrt(omega*omega - alpha*alpha)
	cases := []struct {
		name    string
		initial []float64
		rhs     RHS
		exact   func(float64) []float64
	}{
		{"decay", []float64{1}, func(_ float64, y, dy []float64) error { dy[0] = -0.7 * y[0]; return nil }, func(t float64) []float64 { return []float64{math.Exp(-0.7 * t)} }},
		{"harmonic", []float64{x0, v0}, func(_ float64, y, dy []float64) error { dy[0], dy[1] = y[1], -omega*omega*y[0]; return nil }, func(t float64) []float64 {
			return []float64{x0*math.Cos(omega*t) + v0/omega*math.Sin(omega*t), -x0*omega*math.Sin(omega*t) + v0*math.Cos(omega*t)}
		}},
		{"damped", []float64{x0, v0}, func(_ float64, y, dy []float64) error {
			dy[0], dy[1] = y[1], -omega*omega*y[0]-2*alpha*y[1]
			return nil
		}, func(t float64) []float64 {
			b := (v0 + alpha*x0) / wd
			f := x0*math.Cos(wd*t) + b*math.Sin(wd*t)
			return []float64{math.Exp(-alpha*t) * f, math.Exp(-alpha*t) * (-alpha*f - x0*wd*math.Sin(wd*t) + b*wd*math.Cos(wd*t))}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			previousError := math.Inf(1)
			previousSteps := 0
			for _, tol := range []float64{1e-3, 1e-6, 1e-9} {
				o := DefaultOptions()
				o.RelTolerance, o.AbsTolerance = tol, tol*1e-3
				o.InitialStep, o.MaxStep = 0.5, 2
				times := grid(20, 41)
				result, err := Integrate(context.Background(), tc.rhs, tc.initial, times, o)
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Points) != len(times) || result.Points[len(times)-1].Time != 20 {
					t.Fatal("incomplete integration")
				}
				maxError := 0.0
				componentErrors := make([]float64, len(tc.initial))
				for i, p := range result.Points {
					if p.Time != times[i] || (i > 0 && p.Time <= result.Points[i-1].Time) {
						t.Fatal("invalid time axis")
					}
					exact := tc.exact(p.Time)
					for j, v := range p.State {
						if !finite(v) {
							t.Fatal("non-finite state")
						}
						maxError = math.Max(maxError, math.Abs(v-exact[j]))
						componentErrors[j] = math.Max(componentErrors[j], math.Abs(v-exact[j]))
					}
				}
				if maxError >= previousError || result.Statistics.Accepted < previousSteps {
					t.Fatalf("tighter tolerance did not improve accuracy/work: error=%g previous=%g stats=%+v", maxError, previousError, result.Statistics)
				}
				if tol == 1e-9 && maxError > 2e-7 {
					t.Fatalf("analytical error=%g", maxError)
				}
				repeated, err := Integrate(context.Background(), tc.rhs, tc.initial, times, o)
				if err != nil || !reflect.DeepEqual(result, repeated) {
					t.Fatal("not reproducible")
				}
				if !reflect.DeepEqual(tc.initial, result.Points[0].State) {
					t.Fatal("initial state changed")
				}
				t.Logf("rtol=%g atol=%g max absolute component errors=%v accepted=%d rejected=%d", tol, o.AbsTolerance, componentErrors, result.Statistics.Accepted, result.Statistics.Rejected)
				previousError, previousSteps = maxError, result.Statistics.Accepted
			}
		})
	}
}

func requireKind(t *testing.T, err error, kind ErrorKind) {
	t.Helper()
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != kind {
		t.Fatalf("expected %s, got %v", kind, err)
	}
}

func TestValidationAndFailures(t *testing.T) {
	rhs := func(_ float64, y, dy []float64) error {
		for i := range y {
			dy[i] = -y[i]
		}
		return nil
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := Integrate(context.Background(), rhs, []float64{value}, []float64{0, 1}, DefaultOptions())
		requireKind(t, err, InvalidInput)
		_, err = Integrate(context.Background(), rhs, []float64{1}, []float64{0, value}, DefaultOptions())
		requireKind(t, err, InvalidInput)
		for _, field := range []string{"abs", "rel", "initial", "min", "max"} {
			o := DefaultOptions()
			switch field {
			case "abs":
				o.AbsTolerance = value
			case "rel":
				o.RelTolerance = value
			case "initial":
				o.InitialStep = value
			case "min":
				o.MinStep = value
			case "max":
				o.MaxStep = value
			}
			_, err = Integrate(context.Background(), rhs, []float64{1}, []float64{0, 1}, o)
			requireKind(t, err, InvalidInput)
		}
		_, err = Integrate(context.Background(), func(_ float64, y, dy []float64) error { dy[0] = value; return nil }, []float64{1}, []float64{0, 1}, DefaultOptions())
		requireKind(t, err, NonFinite)
	}
	for _, times := range [][]float64{nil, {0}, {1, 0}, {0, 0}, {-math.MaxFloat64, math.MaxFloat64}} {
		_, err := Integrate(context.Background(), rhs, []float64{1}, times, DefaultOptions())
		requireKind(t, err, InvalidInput)
	}
	for _, o := range []Options{{}, {AbsTolerance: -1, RelTolerance: 1, MinStep: 1, InitialStep: 1, MaxStep: 1, MaxSteps: 1}, {AbsTolerance: 1, RelTolerance: 1, MinStep: 2, InitialStep: 1, MaxStep: 3, MaxSteps: 1}, {AbsTolerance: 1, RelTolerance: 1, MinStep: 1, InitialStep: 3, MaxStep: 2, MaxSteps: 1}} {
		_, err := Integrate(context.Background(), rhs, []float64{1}, []float64{0, 1}, o)
		requireKind(t, err, InvalidInput)
	}
	_, err := Integrate(nil, rhs, []float64{1}, []float64{0, 1}, DefaultOptions())
	requireKind(t, err, InvalidInput)
	_, err = Integrate(context.Background(), nil, []float64{1}, []float64{0, 1}, DefaultOptions())
	requireKind(t, err, InvalidInput)
	_, err = Integrate(context.Background(), rhs, nil, []float64{0, 1}, DefaultOptions())
	requireKind(t, err, InvalidInput)
	_, err = Integrate(context.Background(), func(_ float64, y, dy []float64) error { return nil }, []float64{1}, []float64{0, 1}, DefaultOptions())
	requireKind(t, err, NonFinite)
	o := DefaultOptions()
	o.MaxSteps = 1
	_, err = Integrate(context.Background(), rhs, []float64{1}, []float64{0, 1}, o)
	requireKind(t, err, StepLimit)
	o = DefaultOptions()
	o.MinStep, o.InitialStep, o.MaxStep = 1, 1, 1
	o.AbsTolerance, o.RelTolerance = 1e-15, 1e-15
	_, err = Integrate(context.Background(), rhs, []float64{1}, []float64{0, 1}, o)
	requireKind(t, err, StepTooSmall)
	// Even an output-boundary step below MinStep must meet the error tolerance.
	_, err = Integrate(context.Background(), rhs, []float64{1}, []float64{0, 0.5}, o)
	requireKind(t, err, StepTooSmall)
	_, err = Integrate(context.Background(), rhs, []float64{1}, []float64{1e16, 1e16 + 2}, DefaultOptions())
	requireKind(t, err, StepTooSmall)
	cause := errors.New("model failure")
	_, err = Integrate(context.Background(), func(_ float64, y, dy []float64) error { return cause }, []float64{1}, []float64{0, 1}, DefaultOptions())
	requireKind(t, err, ModelError)
	if !errors.Is(err, cause) {
		t.Fatal("lost model error cause")
	}
}

func TestCancellation(t *testing.T) {
	for _, during := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if !during {
			cancel()
		}
		calls := 0
		_, err := Integrate(ctx, func(_ float64, y, dy []float64) error {
			calls++
			dy[0] = -y[0]
			if calls == 3 {
				cancel()
			}
			return nil
		}, []float64{1}, []float64{0, 10}, DefaultOptions())
		requireKind(t, err, Canceled)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("lost cancellation cause")
		}
		if during && calls != 3 {
			t.Fatal("cancellation not checked between stages")
		}
		cancel()
	}
}

func TestArbitraryDimensionAndStepBounds(t *testing.T) {
	initial := make([]float64, 13)
	for i := range initial {
		initial[i] = float64(i + 1)
	}
	o := DefaultOptions()
	o.MaxStep = 0.07
	rhs := func(_ float64, y, dy []float64) error {
		for i := range y {
			dy[i] = -y[i]
		}
		return nil
	}
	r, err := Integrate(context.Background(), rhs, initial, []float64{0, 1}, o)
	if err != nil {
		t.Fatal(err)
	}
	if r.Statistics.Accepted < 15 {
		t.Fatal("maximum step not honored")
	}
	for i, v := range r.Points[1].State {
		if math.Abs(v-initial[i]*math.Exp(-1)) > 1e-7 {
			t.Fatal("incorrect vector solution")
		}
	}
	r.Points[1].State[0] = 999
	if initial[0] != 1 || r.Points[0].State[0] != 1 {
		t.Fatal("states alias")
	}
	// A final boundary remainder below MinStep is legitimate when accurate.
	o.MinStep, o.InitialStep, o.MaxStep = 0.1, 0.1, 0.1
	_, err = Integrate(context.Background(), rhs, initial, []float64{0, 0.001}, o)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNonautonomousEquation(t *testing.T) {
	// y'=4t^3, y(-2)=16 gives y=t^4, including the zero crossing.
	times := []float64{-2, -1, 0, 0.3, 1, 2}
	r, err := Integrate(context.Background(), func(t float64, y, dy []float64) error { dy[0] = 4 * t * t * t; return nil }, []float64{16}, times, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range r.Points {
		if math.Abs(point.State[0]-math.Pow(point.Time, 4)) > 1e-11 {
			t.Fatalf("nonautonomous state=%v", point)
		}
	}
}

func TestFiniteInputsCannotProduceNonfiniteResults(t *testing.T) {
	o := DefaultOptions()
	o.InitialStep, o.MaxStep = 1, 1
	_, err := Integrate(context.Background(), func(_ float64, y, dy []float64) error { dy[0] = math.MaxFloat64; return nil }, []float64{math.MaxFloat64}, []float64{0, 1}, o)
	requireKind(t, err, NonFinite)
}

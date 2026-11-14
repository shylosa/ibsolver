// Package ode integrates nonstiff initial-value problems using Dormand-Prince 5(4).
// It has no application, storage or UI dependencies.
package ode

import (
	"context"
	"fmt"
	"math"
)

// RHS must write every component of dy. It must not mutate y or retain slices.
// Cancellation is checked between calls; a long-running RHS must cooperate itself.
type RHS func(t float64, y, dy []float64) error

type Options struct {
	AbsTolerance float64 `json:"absTolerance"`
	RelTolerance float64 `json:"relTolerance"`
	InitialStep  float64 `json:"initialStep"`
	MinStep      float64 `json:"minStep"`
	MaxStep      float64 `json:"maxStep"`
	MaxSteps     int     `json:"maxSteps"`
}

func DefaultOptions() Options {
	return Options{AbsTolerance: 1e-9, RelTolerance: 1e-7, InitialStep: 0.01, MinStep: 1e-12, MaxStep: 0.1, MaxSteps: 200000}
}

type ErrorKind string

const (
	InvalidInput ErrorKind = "invalid_input"
	NonFinite    ErrorKind = "non_finite"
	StepTooSmall ErrorKind = "step_too_small"
	StepLimit    ErrorKind = "step_limit"
	Canceled     ErrorKind = "canceled"
	ModelError   ErrorKind = "model_error"
)

type Error struct {
	Kind    ErrorKind
	Time    float64
	Message string
	Cause   error
}

func (e *Error) Error() string { return fmt.Sprintf("ode: %s at t=%g: %s", e.Kind, e.Time, e.Message) }
func (e *Error) Unwrap() error { return e.Cause }

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func (o Options) Validate() error {
	for _, v := range []float64{o.AbsTolerance, o.RelTolerance, o.InitialStep, o.MinStep, o.MaxStep} {
		if !finite(v) || v <= 0 {
			return &Error{Kind: InvalidInput, Message: "tolerances and steps must be positive finite numbers"}
		}
	}
	if o.MinStep > o.InitialStep || o.InitialStep > o.MaxStep || o.MaxSteps < 1 {
		return &Error{Kind: InvalidInput, Message: "require minStep <= initialStep <= maxStep and maxSteps >= 1"}
	}
	return nil
}

type Point struct {
	Time  float64
	State []float64
}

type Statistics struct {
	Accepted    int `json:"accepted"`
	Rejected    int `json:"rejected"`
	Evaluations int `json:"evaluations"`
}

type Result struct {
	Points     []Point
	Statistics Statistics
}

// Dormand-Prince tableau and embedded fourth-order difference (5th minus 4th).
// Reference: Dormand & Prince (1980), doi:10.1016/0771-050X(80)90013-3.
var nodes = [...]float64{0, 1.0 / 5, 3.0 / 10, 4.0 / 5, 8.0 / 9, 1, 1}
var tableau = [7][6]float64{
	{},
	{1.0 / 5},
	{3.0 / 40, 9.0 / 40},
	{44.0 / 45, -56.0 / 15, 32.0 / 9},
	{19372.0 / 6561, -25360.0 / 2187, 64448.0 / 6561, -212.0 / 729},
	{9017.0 / 3168, -355.0 / 33, 46732.0 / 5247, 49.0 / 176, -5103.0 / 18656},
	{35.0 / 384, 0, 500.0 / 1113, 125.0 / 192, -2187.0 / 6784, 11.0 / 84},
}
var difference = [...]float64{71.0 / 57600, 0, -71.0 / 16695, 71.0 / 1920, -17253.0 / 339200, 22.0 / 525, -1.0 / 40}

// Integrate returns states at strictly increasing output times, starting at times[0].
// Each output boundary is reached by an actual RK step, without interpolation.
// MinStep bounds adaptive steps; a shorter step may land exactly on an output time,
// but is never accepted if its error estimate fails. MaxSteps counts all attempts.
// AbsTolerance has the units of each state component; RelTolerance is dimensionless.
// The maximum componentwise local error ratio must be <= 1. This is not a bound
// on accumulated global error. On failure Result contains only completed outputs.
func Integrate(ctx context.Context, rhs RHS, initial, times []float64, o Options) (Result, error) {
	var result Result
	if err := o.Validate(); err != nil {
		return result, err
	}
	if ctx == nil || rhs == nil || len(initial) == 0 || len(times) < 2 {
		return result, &Error{Kind: InvalidInput, Message: "context, RHS, initial state and at least two times are required"}
	}
	for _, v := range initial {
		if !finite(v) {
			return result, &Error{Kind: InvalidInput, Message: "initial state must be finite"}
		}
	}
	for i, t := range times {
		if !finite(t) || (i > 0 && (t <= times[i-1] || !finite(t-times[i-1]))) {
			return result, &Error{Kind: InvalidInput, Message: "output times must be finite and strictly increasing"}
		}
	}
	y := append([]float64(nil), initial...)
	work := make([]float64, len(y))
	var k [7][]float64
	for i := range k {
		k[i] = make([]float64, len(y))
	}
	t, h := times[0], o.InitialStep
	result.Points = append(result.Points, Point{t, append([]float64(nil), y...)})
	failure := func(kind ErrorKind, message string, cause error) (Result, error) {
		return result, &Error{Kind: kind, Time: t, Message: message, Cause: cause}
	}
	for _, target := range times[1:] {
		for t < target {
			if err := ctx.Err(); err != nil {
				return failure(Canceled, err.Error(), err)
			}
			if result.Statistics.Accepted+result.Statistics.Rejected >= o.MaxSteps {
				return failure(StepLimit, "maximum step attempts exceeded", nil)
			}
			step := math.Min(h, target-t)
			end := t + step
			if step == target-t {
				end = target
			}
			step = end - t
			if end <= t || !finite(step) {
				return failure(StepTooSmall, "time cannot advance in float64", nil)
			}
			for stage := range k {
				if err := ctx.Err(); err != nil {
					return failure(Canceled, err.Error(), err)
				}
				for i := range y {
					sum := 0.0
					for j := 0; j < stage; j++ {
						sum += tableau[stage][j] * k[j][i]
					}
					work[i] = y[i] + step*sum
					if !finite(work[i]) {
						return failure(NonFinite, "non-finite stage state", nil)
					}
					k[stage][i] = math.NaN()
				}
				stageTime := t + nodes[stage]*step
				if nodes[stage] == 1 {
					stageTime = end
				}
				result.Statistics.Evaluations++
				if err := rhs(stageTime, work, k[stage]); err != nil {
					return failure(ModelError, err.Error(), err)
				}
				for _, v := range k[stage] {
					if !finite(v) {
						return failure(NonFinite, "RHS must write finite derivatives", nil)
					}
				}
			}
			// work now holds the fifth-order solution (stage 7 state).
			ratio := 0.0
			for i := range y {
				estimate := 0.0
				for j := range k {
					estimate += difference[j] * k[j][i]
				}
				scale := o.AbsTolerance + o.RelTolerance*math.Max(math.Abs(y[i]), math.Abs(work[i]))
				if !finite(scale) {
					return failure(NonFinite, "error scale overflow", nil)
				}
				r := math.Abs(step*estimate) / scale
				if !finite(r) {
					return failure(NonFinite, "non-finite error estimate", nil)
				}
				ratio = math.Max(ratio, r)
			}
			factor := 5.0
			if ratio > 0 {
				factor = math.Max(0.2, math.Min(5, 0.9*math.Pow(ratio, -0.2)))
			}
			if ratio <= 1 {
				copy(y, work)
				t = end
				result.Statistics.Accepted++
				h = math.Max(o.MinStep, math.Min(o.MaxStep, step*factor))
			} else {
				result.Statistics.Rejected++
				if step <= o.MinStep {
					return failure(StepTooSmall, "tolerance cannot be met at minimum step", nil)
				}
				h = math.Max(o.MinStep, step*factor)
			}
		}
		result.Points = append(result.Points, Point{t, append([]float64(nil), y...)})
	}
	if err := ctx.Err(); err != nil {
		return failure(Canceled, err.Error(), err)
	}
	return result, nil
}

package chartdata

import (
	"context"
	"fmt"
	"ibsolver/internal/domain"
	"ibsolver/internal/ode"
)

// Oscillator solves x'=v, v'=-(k/m)x-(c/m)v, with state [x (m), v (m/s)].
// Derived chart series are computed from integrated states, not analytic curves.
func Oscillator(ctx context.Context, input domain.NumericalExample) (domain.Run, error) {
	if err := input.Validate(); err != nil {
		return domain.Run{}, err
	}
	p := input.Model
	times := make([]float64, p.Samples)
	for i := range times {
		times[i] = p.Duration * (float64(i) / float64(p.Samples-1))
	}
	times[len(times)-1] = p.Duration
	acceleration := func(x, v float64) float64 { return -(p.Stiffness/p.Mass)*x - (p.Damping/p.Mass)*v }
	rhs := func(_ float64, y, dy []float64) error {
		dy[0], dy[1] = y[1], acceleration(y[0], y[1])
		return nil
	}
	result, err := ode.Integrate(ctx, rhs, []float64{p.Position, p.Velocity}, times, input.Solver)
	if err != nil {
		return domain.Run{}, fmt.Errorf("Не вдалося виконати чисельне інтегрування: %w", err)
	}
	chart := domain.ChartData{Time: times, Series: make([]domain.Series, 6)}
	names := []string{"Зміщення", "Швидкість", "Прискорення", "Кінетична енергія", "Потенціальна енергія", "Повна енергія"}
	units := []string{"м", "м/с", "м/с²", "Дж", "Дж", "Дж"}
	for j := range chart.Series {
		chart.Series[j] = domain.Series{Name: names[j], Unit: units[j], Values: make([]float64, p.Samples)}
	}
	for i, point := range result.Points {
		if err := ctx.Err(); err != nil {
			return domain.Run{}, err
		}
		x, v := point.State[0], point.State[1]
		kinetic, potential := 0.5*p.Mass*v*v, 0.5*p.Stiffness*x*x
		values := []float64{x, v, acceleration(x, v), kinetic, potential, kinetic + potential}
		for j, value := range values {
			chart.Series[j].Values[i] = value
		}
	}
	last := p.Samples - 1
	summary := &domain.NumericalSummary{FinalPosition: chart.Series[0].Values[last], FinalVelocity: chart.Series[1].Values[last], InitialEnergy: chart.Series[5].Values[0], FinalEnergy: chart.Series[5].Values[last], Statistics: result.Statistics}
	run := domain.Run{NumericalSnapshot: &input, ResultVersion: domain.NumericalVersion, Summary: domain.Summary{Kind: "NUMERICAL", Samples: p.Samples, Duration: p.Duration, Numerical: summary}, Chart: chart}
	if err := run.Validate(); err != nil {
		return domain.Run{}, fmt.Errorf("Некоректний результат осцилятора: %w", err)
	}
	return run, nil
}

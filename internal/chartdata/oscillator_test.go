package chartdata

import (
	"context"
	"ibsolver/internal/domain"
	"math"
	"testing"
)

func TestOscillatorPhysics(t *testing.T) {
	for _, damping := range []float64{0, 0.3, 4, 8} { // Undamped, underdamped, critical, overdamped.
		input := domain.DefaultNumerical()
		input.Model.Damping = damping
		run, err := Oscillator(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		p := input.Model
		alpha := p.Damping / (2 * p.Mass)
		omega2 := p.Stiffness / p.Mass
		maxError := 0.0
		for i, time := range run.Chart.Time {
			var x, v float64
			if alpha*alpha < omega2 {
				w := math.Sqrt(omega2 - alpha*alpha)
				b := (p.Velocity + alpha*p.Position) / w
				f := p.Position*math.Cos(w*time) + b*math.Sin(w*time)
				x = math.Exp(-alpha*time) * f
				v = math.Exp(-alpha*time) * (-alpha*f - p.Position*w*math.Sin(w*time) + b*w*math.Cos(w*time))
			} else if alpha*alpha == omega2 {
				b := p.Velocity + alpha*p.Position
				x = (p.Position + b*time) * math.Exp(-alpha*time)
				v = (b - alpha*(p.Position+b*time)) * math.Exp(-alpha*time)
			} else {
				r1, r2 := -alpha+math.Sqrt(alpha*alpha-omega2), -alpha-math.Sqrt(alpha*alpha-omega2)
				a := (p.Velocity - r2*p.Position) / (r1 - r2)
				b := p.Position - a
				x = a*math.Exp(r1*time) + b*math.Exp(r2*time)
				v = r1*a*math.Exp(r1*time) + r2*b*math.Exp(r2*time)
			}
			maxError = math.Max(maxError, math.Max(math.Abs(run.Chart.Series[0].Values[i]-x), math.Abs(run.Chart.Series[1].Values[i]-v)))
			energy := run.Chart.Series[5].Values[i]
			if damping == 0 && math.Abs(energy-2) > 1e-7 {
				t.Fatalf("energy drift=%g J", energy-2)
			}
			if damping > 0 && i > 0 && energy > run.Chart.Series[5].Values[i-1]+1e-10 {
				t.Fatal("damped energy increased")
			}
			if energy != run.Chart.Series[3].Values[i]+run.Chart.Series[4].Values[i] {
				t.Fatal("energy sum mismatch")
			}
			if math.Abs(run.Chart.Series[2].Values[i]+(p.Stiffness/p.Mass)*run.Chart.Series[0].Values[i]+(p.Damping/p.Mass)*run.Chart.Series[1].Values[i]) > 1e-12 {
				t.Fatal("force balance mismatch")
			}
		}
		if maxError > 1e-7 {
			t.Fatalf("state error=%g", maxError)
		}
		t.Logf("c=%g kg/s max absolute state error=%.9g", damping, maxError)
	}
}

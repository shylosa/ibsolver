// Package chartdata provides explicit demonstration data without a physical model.
package chartdata

import (
	"ibsolver/internal/domain"
	"math"
)

func Demo(inputs domain.Inputs) (domain.Run, error) {
	if err := inputs.Validate(); err != nil {
		return domain.Run{}, err
	}
	d := domain.ChartData{Time: make([]float64, inputs.Samples), Series: make([]domain.Series, 6)}
	for j := range d.Series {
		d.Series[j] = domain.Series{Name: []string{"Синус", "Косинус", "Затухання", "Хвиля", "Імпульс", "Перехід"}[j], Unit: "ум. од.", Values: make([]float64, inputs.Samples)}
	}
	for i := range d.Time {
		t := inputs.Duration * float64(i) / float64(inputs.Samples-1)
		x := 2 * math.Pi * inputs.Frequency * t
		d.Time[i] = t
		values := [6]float64{math.Sin(x), math.Cos(x), math.Exp(-3*t/inputs.Duration) * math.Sin(x), math.Sin(x)*0.7 + math.Sin(2*x)*0.3, math.Exp(-math.Pow((t/inputs.Duration-0.5)*8, 2)), 1 - math.Exp(-5*t/inputs.Duration)}
		for j, v := range values {
			d.Series[j].Values[i] = inputs.Amplitude * float64(j+1) * v
		}
	}
	return domain.Run{Snapshot: inputs, ResultVersion: domain.ResultVersion, Summary: domain.Summary{Kind: "DEMO", Samples: inputs.Samples, Duration: inputs.Duration}, Chart: d}, nil
}

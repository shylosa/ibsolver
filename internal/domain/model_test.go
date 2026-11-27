package domain

import (
	"math"
	"testing"
)

func TestConfigurationValidatesOnlyActiveMode(t *testing.T) {
	p := DefaultParameters()
	c := Configuration{Name: "Test", Mode: ModeOscillator, Parameters: p}
	nan := math.NaN()
	c.Parameters.Preparation = &PreparationInputs{Pmax: &nan}
	if err := c.Validate(); err != nil {
		t.Fatal("inactive preparation block must not obstruct oscillator validation", err)
	}
	c.Parameters.Preparation = nil
	c.Parameters.Oscillator.Model.Mass = -1
	if err := c.Validate(); err == nil {
		t.Fatal("accepted invalid active oscillator")
	}
	c.Parameters.Oscillator = nil
	if err := c.Validate(); err == nil {
		t.Fatal("accepted missing oscillator parameters")
	}
	c.Parameters = DefaultParameters()
	c.Parameters.Oscillator.Model.Position, c.Parameters.Oscillator.Model.Velocity = 0, 0
	if err := c.Validate(); err != nil {
		t.Fatalf("zero initial conditions are actual values: %v", err)
	}
	c.Mode = "future"
	if err := c.Validate(); err == nil {
		t.Fatal("accepted unknown mode")
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		c.Mode, c.Parameters = ModeOscillator, DefaultParameters()
		c.Parameters.Oscillator.Model.Duration = value
		if err := c.Validate(); err == nil {
			t.Fatal("accepted non-finite active parameter")
		}
	}
}

func TestSeriesReferencesAndIdentity(t *testing.T) {
	chart := ChartData{Time: []float64{0, 1}, Series: []Series{{ID: "signal", Name: "Signal", Values: []float64{0, -2}}}}
	for _, reference := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		chart.Series[0].Reference = &reference
		if err := chart.Validate(); err == nil {
			t.Fatalf("accepted reference %v", reference)
		}
	}
	reference := 2.0
	chart.Series[0].Reference = &reference
	if err := chart.Validate(); err != nil {
		t.Fatal(err)
	}
	chart.Series = append(chart.Series, chart.Series[0])
	if err := chart.Validate(); err == nil {
		t.Fatal("accepted duplicate ID")
	}
	chart.Series = chart.Series[:1]
	chart.Aux = []Series{{ID: "aux-e", Name: "E", Values: []float64{0, 1}}}
	if err := chart.Validate(); err != nil {
		t.Fatal(err)
	}
	chart.Aux[0].Values = []float64{0}
	if err := chart.Validate(); err == nil {
		t.Fatal("accepted aux series shorter than time")
	}
	chart.Aux[0].Values = []float64{0, 1}
	chart.Aux[0].ID = "signal"
	if err := chart.Validate(); err == nil {
		t.Fatal("accepted aux ID colliding with a series")
	}
}

func TestSeriesPreferencesFollowIdentity(t *testing.T) {
	settings := DefaultSettings()
	settings.Visible[0], settings.Scales[1] = false, 2
	series := []Series{{ID: "a"}, {ID: "b"}}
	settings.BindSeries(series)
	settings.BindSeries([]Series{series[1], series[0]})
	if settings.Series["a"].Visible || settings.Series["b"].Scale != 2 {
		t.Fatal("preferences changed after reorder")
	}
}

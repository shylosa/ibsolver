package domain

import (
	"math"
	"testing"
)

func TestConfigurationValidatesOnlyActiveMode(t *testing.T) {
	p := DefaultParameters()
	c := Configuration{Name: "Test", Mode: ModeOscillator, Parameters: p}
	c.Parameters.Demo.Duration = -1
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Mode = ModeDemo
	if err := c.Validate(); err == nil {
		t.Fatal("accepted invalid active demo")
	}
	c.Parameters.Demo = nil
	if err := c.Validate(); err == nil {
		t.Fatal("accepted missing demo parameters")
	}
	c.Parameters = DefaultParameters()
	c.Parameters.Oscillator.Model.Mass = -1
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Mode = ModeOscillator
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
	for _, mode := range []Mode{ModeDemo, ModeOscillator} {
		for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
			c.Mode, c.Parameters = mode, DefaultParameters()
			c.Parameters.Demo.Duration = value
			c.Parameters.Oscillator.Model.Duration = value
			if err := c.Validate(); err == nil {
				t.Fatal("accepted non-finite active parameter")
			}
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

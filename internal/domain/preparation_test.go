package domain

import (
	"encoding/json"
	"math"
	"testing"
)

func TestPreparationNullableAndActiveValidation(t *testing.T) {
	p := NewPreparationParameters()
	data, err := json.Marshal(p.Preparation)
	if err != nil {
		t.Fatalf("nullable: %s %v", data, err)
	}

	zero, negative, invalid := 0.0, -42.0, math.Inf(1)
	p.Preparation.Q, p.Preparation.Ns = &zero, &negative
	if err := p.Validate(ModePreparation); err != nil {
		t.Fatal(err)
	}
	p.Preparation.D = &invalid
	if p.Validate(ModePreparation) == nil {
		t.Fatal("accepted infinity")
	}
	if err := p.Validate(ModeOscillator); err != nil {
		t.Fatal("validated inactive inputs", err)
	}
	invalid = math.NaN()
	if p.Validate(ModePreparation) == nil {
		t.Fatal("accepted NaN")
	}
	if (Parameters{}).Validate(ModePreparation) == nil {
		t.Fatal("accepted absent active block")
	}
	if (Run{Mode: ModePreparation, Snapshot: p}).Matches(Configuration{Mode: ModePreparation, Parameters: p}) {
		t.Fatal("preparation cannot have a run")
	}
}

func TestPreparationWorkbookInputsValidateFiniteNumbersAndFormats(t *testing.T) {
	zero, negative, invalid := 0.0, -42.0, math.NaN()
	p := PreparationWorkbookInputs{D: &zero, Ns: &negative, InputFormats: map[string]string{"d": "0,0"}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Af = &invalid
	if p.Validate() == nil {
		t.Fatal("accepted NaN")
	}
	p.Af = nil
	p.InputFormats = map[string]string{"other": "1"}
	if p.Validate() == nil {
		t.Fatal("accepted unknown input format")
	}
	if (Parameters{PreparationWorkbook: &p}).Validate(ModeOscillator) == nil {
		t.Fatal("accepted invalid stored workbook inputs in another mode")
	}
}

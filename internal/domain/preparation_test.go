package domain

import (
	"encoding/json"
	"math"
	"testing"
)

func TestPreparationNullableAndActiveValidation(t *testing.T) {
	p := NewPreparationParameters()
	data, err := json.Marshal(p.Preparation)
	if err != nil || string(data) != `{"d":null,"q":null,"vd":null,"lc":null,"ns":null,"f1":null,"af":null}` {
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

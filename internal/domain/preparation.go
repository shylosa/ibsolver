package domain

import (
	"fmt"
	"math"
)

// PreparationInputs stores d and lc in metres, q in kilograms, vd in metres
// per second. Ns, F1 and Af retain source numbers with unknown units.
// Nil is absent; no physical limits or defaults have been established.
type PreparationInputs struct {
	D  *float64 `json:"d"`
	Q  *float64 `json:"q"`
	Vd *float64 `json:"vd"`
	Lc *float64 `json:"lc"`
	Ns *float64 `json:"ns"`
	F1 *float64 `json:"f1"`
	Af *float64 `json:"af"`
}

func (p PreparationInputs) Validate() error {
	for _, n := range []*float64{p.D, p.Q, p.Vd, p.Lc, p.Ns, p.F1, p.Af} {
		if n != nil && (math.IsNaN(*n) || math.IsInf(*n, 0)) {
			return fmt.Errorf("Підготовчі дані мають бути скінченними числами")
		}
	}
	return nil
}

func (p PreparationInputs) Equal(other PreparationInputs) bool {
	eq := func(a, b *float64) bool {
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		return *a == *b
	}
	return eq(p.D, other.D) && eq(p.Q, other.Q) && eq(p.Vd, other.Vd) &&
		eq(p.Lc, other.Lc) && eq(p.Ns, other.Ns) && eq(p.F1, other.F1) && eq(p.Af, other.Af)
}

func NewPreparationParameters() Parameters {
	p := DefaultParameters()
	p.Preparation = &PreparationInputs{}
	return p
}

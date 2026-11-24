package domain

import (
	"fmt"
	"math"
)

// PreparationInputs stores the preparation fields in their established SI units
// where units are known. Unknown BAS units remain source numbers.
// Nil is absent; no physical limits or defaults have been established.
type PreparationInputs struct {
	Pmax         *float64          `json:"pmax"`
	Vd           *float64          `json:"vd"`
	D            *float64          `json:"d"`
	Q            *float64          `json:"q"`
	Ns           *float64          `json:"ns"`
	Hi           *float64          `json:"hi"`
	Pfor         *float64          `json:"pfor"`
	Kt           *float64          `json:"kt"`
	Tn           *float64          `json:"tn"`
	Forma        *int              `json:"forma"`
	Omega        *float64          `json:"omega"`
	W0           *float64          `json:"w0"`
	Ro           *float64          `json:"ro"`
	Tg15         *float64          `json:"tg15"`
	Ku1          *float64          `json:"ku1"`
	F1           *float64          `json:"f1"`
	U15          *float64          `json:"u15"`
	Svod         *float64          `json:"svod"`
	Pvos         *float64          `json:"pvos"`
	Tvos         *float64          `json:"tvos"`
	Lc           *float64          `json:"lc"`
	Af           *float64          `json:"af"`
	InputFormats map[string]string `json:"inputFormats,omitempty"`
}

func (p PreparationInputs) Validate() error {
	for _, n := range []*float64{
		p.Pmax, p.Vd, p.D, p.Q, p.Ns, p.Hi, p.Pfor,
		p.Kt, p.Tn, p.Omega, p.W0, p.Ro, p.Tg15,
		p.Ku1, p.F1, p.U15, p.Svod, p.Pvos, p.Tvos, p.Lc,
		p.Af,
	} {
		if n != nil && (math.IsNaN(*n) || math.IsInf(*n, 0)) {
			return fmt.Errorf("Підготовчі дані мають бути скінченними числами")
		}
	}
	if p.Forma != nil && (*p.Forma < 1 || *p.Forma > 7) {
		return fmt.Errorf("Форма зерна має бути цілим номером від 1 до 7")
	}
	for key, format := range p.InputFormats {
		switch key {
		case "pmax", "vd", "d", "q", "ns", "hi", "af", "pfor", "kt", "tn", "forma",
			"omega", "w0", "ro", "tg15", "ku1", "f1", "u15", "svod", "pvos", "tvos", "lc":
		default:
			return fmt.Errorf("Невідомий формат поля підготовчих даних: %s", key)
		}
		if len(format) > 128 {
			return fmt.Errorf("Формат поля підготовчих даних задовгий: %s", key)
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
	eqInt := func(a, b *int) bool {
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		return *a == *b
	}
	return eq(p.Pmax, other.Pmax) && eq(p.Vd, other.Vd) && eq(p.D, other.D) &&
		eq(p.Q, other.Q) && eq(p.Ns, other.Ns) && eq(p.Hi, other.Hi) &&
		eq(p.Pfor, other.Pfor) && eq(p.Kt, other.Kt) &&
		eq(p.Tn, other.Tn) && eqInt(p.Forma, other.Forma) && eq(p.Omega, other.Omega) &&
		eq(p.W0, other.W0) && eq(p.Ro, other.Ro) && eq(p.Tg15, other.Tg15) &&
		eq(p.Ku1, other.Ku1) && eq(p.F1, other.F1) && eq(p.U15, other.U15) &&
		eq(p.Svod, other.Svod) && eq(p.Pvos, other.Pvos) && eq(p.Tvos, other.Tvos) &&
		eq(p.Lc, other.Lc) && eq(p.Af, other.Af)
}

func NewPreparationParameters() Parameters {
	p := DefaultParameters()
	p.Preparation = &PreparationInputs{}
	return p
}

package domain

import (
	"fmt"

	"ibsolver/internal/prepcalc"
)

// WorkbookVersion marks results produced by the Диплом.xls preparation kernel
// (package prepcalc). Such a run stores a scalar summary and no time series.
const WorkbookVersion = "workbook-v1"

// WorkbookInputCount is the number of manual fields the kernel requires.
const WorkbookInputCount = 7

func (r Run) validateWorkbook() error {
	if r.Summary.Kind != "WORKBOOK" || r.Snapshot.PreparationWorkbook == nil || r.Summary.Workbook == nil {
		return fmt.Errorf("Неповний результат підготовчих даних")
	}
	if err := r.Snapshot.Validate(r.Mode); err != nil {
		return err
	}
	if len(r.Chart.Time) != 0 || len(r.Chart.Series) != 0 || len(r.Chart.Aux) != 0 {
		return fmt.Errorf("Результат підготовчих даних не має містити часових рядів")
	}
	return r.Summary.Workbook.Validate()
}

// WorkbookInputs converts the stored workbook fields into kernel inputs.
// Every field must be present; null is absent, not zero.
func WorkbookInputs(p *PreparationWorkbookInputs) (prepcalc.Inputs, error) {
	if p == nil {
		return prepcalc.Inputs{}, fmt.Errorf("Відсутні підготовчі дані з Excel")
	}
	fields := []struct {
		name string
		v    *float64
	}{
		{"d", p.D}, {"q", p.Q}, {"Vд", p.Vd}, {"lc", p.Lc},
		{"ns", p.Ns}, {"f1", p.F1}, {"af", p.Af},
	}
	for _, f := range fields {
		if f.v == nil {
			return prepcalc.Inputs{}, fmt.Errorf("Заповніть усі сім полів підготовчих даних; відсутнє: %s", f.name)
		}
	}
	inputs := prepcalc.Inputs{D: *p.D, Q: *p.Q, Vd: *p.Vd, Lc: *p.Lc, Ns: *p.Ns, F1: *p.F1, Af: *p.Af}
	if err := inputs.Validate(); err != nil {
		return prepcalc.Inputs{}, err
	}
	return inputs, nil
}

// Equal compares the physical values only; the stored raw text formats do not
// make a saved run stale.
func (p PreparationWorkbookInputs) Equal(other PreparationWorkbookInputs) bool {
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

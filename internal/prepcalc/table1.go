package prepcalc

import "fmt"

// Table1Row is one row of the Slukhotsky generalized-characteristics table
// (sheet «Таблица 1 (Слухоцкого)», rows 7-18 of Диплом.xls). Column letters
// in the comments refer to that sheet.
type Table1Row struct {
	// Ce is the interpolation argument in MJ/m³ (column B).
	Ce float64
	// PmKr is pm.кр, maximum crusher pressure in MPa (column C).
	PmKr float64
	// Delta is Δ, loading density in kg/m³ (column D).
	Delta float64
	// Hi is χ, the chamber widening coefficient (column E).
	Hi float64
	// Kisp is ηω, charge mass usage coefficient in MJ/kg (column F).
	Kisp float64
	// LstvD is Lств/d, barrel length in calibers (column G).
	LstvD float64
}

// Table1 is the reference table used for linear interpolation. Rows must be
// sorted by strictly ascending Ce, as required by the workbook MATCH lookup.
type Table1 []Table1Row

// defaultTable1 holds the values of «Таблица 1 (Слухоцкого)»!B7:G18.
var defaultTable1 = Table1{
	{Ce: 500, PmKr: 125, Delta: 470, Hi: 1.01, Kisp: 1.36, LstvD: 12},
	{Ce: 1000, PmKr: 170, Delta: 510, Hi: 1.02, Kisp: 1.34, LstvD: 18},
	{Ce: 2000, PmKr: 195, Delta: 580, Hi: 1.09, Kisp: 1.30, LstvD: 29},
	{Ce: 3000, PmKr: 220, Delta: 640, Hi: 1.18, Kisp: 1.27, LstvD: 39},
	{Ce: 4000, PmKr: 245, Delta: 680, Hi: 1.28, Kisp: 1.23, LstvD: 47},
	{Ce: 5000, PmKr: 270, Delta: 710, Hi: 1.39, Kisp: 1.20, LstvD: 53},
	{Ce: 6000, PmKr: 295, Delta: 720, Hi: 1.50, Kisp: 1.16, LstvD: 59},
	{Ce: 7000, PmKr: 320, Delta: 740, Hi: 1.61, Kisp: 1.13, LstvD: 63},
	{Ce: 8000, PmKr: 340, Delta: 740, Hi: 1.73, Kisp: 1.11, LstvD: 66},
	{Ce: 9000, PmKr: 365, Delta: 740, Hi: 1.85, Kisp: 1.08, LstvD: 69},
	{Ce: 10000, PmKr: 390, Delta: 740, Hi: 1.98, Kisp: 1.06, LstvD: 71},
	{Ce: 11000, PmKr: 415, Delta: 750, Hi: 2.07, Kisp: 1.03, LstvD: 73},
}

// DefaultTable1 returns a fresh copy of the Slukhotsky table from Диплом.xls.
func DefaultTable1() Table1 {
	out := make(Table1, len(defaultTable1))
	copy(out, defaultTable1)
	return out
}

// Validate rejects tables that the workbook MATCH/INDEX lookup cannot handle:
// fewer than two rows, non-finite values, non-ascending or duplicate
// arguments, and zero denominators of the interpolated columns.
func (t Table1) Validate() error {
	if len(t) < 2 {
		return newError(ErrTableInvalid, "Таблиця Слухоцького має містити щонайменше два рядки, отримано %d", len(t))
	}
	for i, row := range t {
		for _, f := range []struct {
			name string
			v    float64
		}{
			{"Ce", row.Ce}, {"pm.кр", row.PmKr}, {"Delta", row.Delta},
			{"Hi", row.Hi}, {"Kисп", row.Kisp}, {"Lств/d", row.LstvD},
		} {
			if !finite(f.v) {
				return newError(ErrTableInvalid, "Рядок %d таблиці Слухоцького: %s має бути скінченним числом", i+1, f.name)
			}
		}
		if row.Delta <= 0 {
			return newError(ErrTableInvalid, "Рядок %d таблиці Слухоцького: Delta має бути додатним", i+1)
		}
		if row.Hi <= 0 {
			return newError(ErrTableInvalid, "Рядок %d таблиці Слухоцького: Hi має бути додатним", i+1)
		}
		if row.Kisp <= 0 {
			return newError(ErrTableInvalid, "Рядок %d таблиці Слухоцького: Kисп має бути додатним", i+1)
		}
		if i > 0 && row.Ce <= t[i-1].Ce {
			return newError(ErrTableInvalid, "Таблиця Слухоцького має бути відсортована за зростанням Ce: %g ≤ %g у рядках %d і %d", row.Ce, t[i-1].Ce, i+1, i)
		}
	}
	return nil
}

// MatchIndex returns the 1-based row index of the largest table argument not
// exceeding arg, mirroring MATCH(arg, аргументи, 1) of the workbook. It fails
// below the first argument (Excel #N/A) and at or above the last argument,
// where the workbook INDEX of the next row reads past the table (#REF!).
func (t Table1) MatchIndex(arg float64) (int, error) {
	lo, hi := 0, len(t)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if t[mid].Ce <= arg {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	switch {
	case lo == 0:
		return 0, newError(ErrTableRange, "Значення Ce = %g МДж/м³ менше за перший рядок таблиці Слухоцького (%g); у книзі це помилка #N/A", arg, t[0].Ce)
	case lo == len(t):
		return 0, newError(ErrTableRange, "Значення Ce = %g МДж/м³ досягає або перевищує останній рядок таблиці Слухоцького (%g); у книзі це помилка #REF!", arg, t[len(t)-1].Ce)
	}
	return lo, nil
}

// Interpolate linearly interpolates the selected column value between the
// table rows idx and idx+1 (1-based), reproducing the workbook expression
// ($G$24-$L$25)*(K-J)/($M$25-$L$25)+J exactly, including the operand order.
func (t Table1) Interpolate(idx int, arg float64, column func(Table1Row) float64) float64 {
	y1, y2 := t[idx-1].Ce, t[idx].Ce
	x1, x2 := column(t[idx-1]), column(t[idx])
	return (arg-y1)*(x2-x1)/(y2-y1) + x1
}

// String implements fmt.Stringer for diagnostics.
func (r Table1Row) String() string {
	return fmt.Sprintf("Ce=%g PmKr=%g Delta=%g Hi=%g Kisp=%g LstvD=%g",
		r.Ce, r.PmKr, r.Delta, r.Hi, r.Kisp, r.LstvD)
}

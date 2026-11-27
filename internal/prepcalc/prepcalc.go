// Package prepcalc implements the initial ballistic design calculation of the
// legacy workbook Диплом.xls (sheet «Расчеты»), described in Diplom_4.doc,
// section 1 "Проектирование исходного варианта (по обобщенным данным Слухоцкого)".
//
// Every formula was recovered from the BIFF8 FORMULA tokens of the workbook
// (see docs/preparation-xls/formulas.json) and is evaluated here in the exact
// operand order of the original RPN, so the results reproduce the saved cell
// values of the workbook. The reference table of Slukhotsky (sheet
// «Таблица 1 (Слухоцкого)», rows 7-18) is embedded as DefaultTable1.
//
// Fidelity notes:
//   - The workbook input lc (D10) is not referenced by any formula; the barrel
//     length is derived from the table column Lств/d instead. The field is kept
//     in Inputs for parity with the workbook.
//   - For 13000 <= Cq <= 15000 the workbook cell I51 («Уточнённые значения»,
//     LS) falls back to the pm.кр pressure value F25 instead of keeping LS.
//     That anomaly is reproduced unchanged and reported in Result.Warnings.
//   - Below the first table argument Excel MATCH yields #N/A and above or at
//     the last argument the adjacent INDEX reads past the table (#REF!). The
//     package returns ErrTableRange in both cases.
//
// The package has no application, storage, UI or Wails dependencies.
package prepcalc

import (
	"fmt"
	"math"
)

// ErrorKind classifies prepcalc errors.
type ErrorKind string

const (
	// ErrInvalidInput marks a non-finite or physically invalid input value.
	ErrInvalidInput ErrorKind = "invalid_input"
	// ErrTableInvalid marks a reference table that cannot be used
	// (too short, not strictly ascending, non-finite or zero denominators).
	ErrTableInvalid ErrorKind = "table_invalid"
	// ErrTableRange marks an interpolation argument outside the usable
	// table range, where the original workbook fails (#N/A or #REF!).
	ErrTableRange ErrorKind = "table_range"
	// ErrNonFinite marks a non-finite intermediate or final result.
	ErrNonFinite ErrorKind = "non_finite"
)

// Error is a typed calculation error with a user-facing message.
type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("prepcalc: %s: %s", e.Kind, e.Message)
}

func newError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Inputs are the seven manual inputs of the workbook, entered in the yellow
// cells Расчеты!D7:D10 and H7:H9. Units follow the workbook labels.
type Inputs struct {
	// D is the caliber d in millimeters (D7).
	D float64
	// Q is the projectile mass q in kilograms (D8).
	Q float64
	// Vd is the muzzle velocity Vд in meters per second (D9).
	Vd float64
	// Lc is the barrel length lc in meters (D10). No workbook formula reads
	// it; the barrel length is derived from the table column Lств/d.
	Lc float64
	// Ns is the coefficient ns accounting for rifling depth (H7).
	Ns float64
	// F1 is phi1 accounting for friction and projectile rotation (H8).
	F1 float64
	// Af is the coefficient afi chosen from workbook Table 2 (H9).
	Af float64
}

// Validate rejects non-finite numbers and physically meaningless values that
// the calculation cannot proceed with (divisors must be positive).
func (in Inputs) Validate() error {
	fields := []struct {
		name string
		v    float64
	}{
		{"d", in.D}, {"q", in.Q}, {"Vд", in.Vd}, {"lc", in.Lc},
		{"ns", in.Ns}, {"f1", in.F1}, {"af", in.Af},
	}
	for _, f := range fields {
		if !finite(f.v) {
			return newError(ErrInvalidInput, "Параметр %s має бути скінченним числом", f.name)
		}
	}
	if in.D <= 0 {
		return newError(ErrInvalidInput, "Калібр d має бути додатним числом, отримано %g", in.D)
	}
	if in.Q <= 0 {
		return newError(ErrInvalidInput, "Маса снаряда q має бути додатним числом, отримано %g", in.Q)
	}
	if in.Vd <= 0 {
		return newError(ErrInvalidInput, "Дульна швидкість Vд має бути додатним числом, отримано %g", in.Vd)
	}
	if in.Ns <= 0 {
		return newError(ErrInvalidInput, "Коефіцієнт ns має бути додатним числом, отримано %g", in.Ns)
	}
	if in.F1 <= 0 {
		return newError(ErrInvalidInput, "Коефіцієнт f1 має бути додатним числом, отримано %g", in.F1)
	}
	return nil
}

// Result mirrors the computed quantities of the workbook sheet «Расчеты».
// Every field names the source cell in its comment. Units follow the labels
// of the workbook; the mirror fields (MJ, g, l, mm) repeat the primary
// quantities in the secondary units shown on the sheet.
type Result struct {
	// Ed is Eд = q·Vд²/2, muzzle energy in joules (F13).
	Ed float64 `json:"ed"`
	// EdMJ repeats Ed in megajoules (H13).
	EdMJ float64 `json:"edMJ"`
	// Cq is cq = q/(0.001·d)³, projectile mass coefficient in kg/m³ (F15).
	Cq float64 `json:"cq"`
	// Ce is Ce = Eд/(0.001·d)³, power coefficient in J/m³ (F17).
	Ce float64 `json:"ce"`
	// CeMJ repeats Ce in MJ/m³; it is the interpolation argument (H17, G24).
	CeMJ float64 `json:"ceMJ"`
	// S is s = ns·(0.001·d)², barrel cross-section in m² (F20).
	S float64 `json:"s"`

	// TableIndex is the 1-based MATCH row of CeMJ in the table argument
	// column (I25). The interpolated values use rows TableIndex..TableIndex+1.
	TableIndex int `json:"tableIndex"`
	// PmKr is pm.кр, the maximum crusher pressure in MPa (F25).
	PmKr float64 `json:"pmKr"`
	// Delta is Δ, loading density in kg/m³ (F26).
	Delta float64 `json:"delta"`
	// Hi is χ, the chamber widening coefficient (F27).
	Hi float64 `json:"hi"`
	// Kisp is ηω, charge mass usage coefficient in MJ/kg (F28).
	Kisp float64 `json:"kisp"`
	// LstvD is Lств/d, barrel length in calibers (F29).
	LstvD float64 `json:"lstvD"`

	// Omega is ω = Eд/(ηω·10⁶), charge mass in kilograms (F31).
	Omega float64 `json:"omega"`
	// OmegaG repeats Omega in grams (H31).
	OmegaG float64 `json:"omegaG"`
	// W0 is W0 = ω/Δ, chamber volume in m³ (F33).
	W0 float64 `json:"w0"`
	// W0L repeats W0 in liters (H33).
	W0L float64 `json:"w0l"`
	// L0 is l0 = W0/s, reduced chamber length in meters (F35).
	L0 float64 `json:"l0"`
	// L0MM repeats L0 in millimeters (H35).
	L0MM float64 `json:"l0mm"`
	// LcCalc is the barrel length from the table, Lств/d·d, in meters (F38).
	LcCalc float64 `json:"lcCalc"`
	// LcMM repeats LcCalc in millimeters (H38).
	LcMM float64 `json:"lcmm"`
	// LS is the barrel length for the solver, Lств − 1.5d, in meters (I39).
	LS float64 `json:"ls"`
	// Lkam is lкам = l0/χ, chamber length in meters (F40).
	Lkam float64 `json:"lkam"`
	// LkamMM repeats Lkam in millimeters (H40).
	LkamMM float64 `json:"lkammm"`
	// Ld is lд = LS − lкам, projectile travel in meters (F42).
	Ld float64 `json:"ld"`
	// LdMM repeats Ld in millimeters (H42).
	LdMM float64 `json:"ldmm"`
	// LambdaD is Λд = lд/l0, expansion volume ratio (F44).
	LambdaD float64 `json:"lambdaD"`

	// PmKrU is the refined pm.кр after the Cq correction, in MPa (F50).
	PmKrU float64 `json:"pmKrU"`
	// LSu is the refined LS after the Cq correction, in meters (I51).
	LSu float64 `json:"lsu"`
	// LcU is Lс.у = LSu, the refined barrel length in meters (F51).
	LcU float64 `json:"lcU"`
	// LdU is lд.у = LSu − lкам, refined projectile travel in meters (F52).
	LdU float64 `json:"ldU"`
	// LambdaDU is Λд.у = lд.у/l0 (I52).
	LambdaDU float64 `json:"lambdaDU"`

	// Lambda1 is Λ1 = (Λд.у + 1/χ)/(Λд.у + 1) (F55).
	Lambda1 float64 `json:"lambda1"`
	// Fi is φ = af + Λ1·ω/(3q), fictitiousness coefficient (F60).
	Fi float64 `json:"fi"`
	// Pm is pm = 1.12·φ·pm.кр/(φ1·(1 + Λ1·ω/(2q))),
	// maximum mean-ballistic pressure in MPa (F63).
	Pm float64 `json:"pm"`
	// EttaD is ηд = φ·Eд/(pm·10⁶·s·lд.у), indicator diagram
	// filling coefficient (F67).
	EttaD float64 `json:"ettaD"`

	// Transfer mirrors the workbook block «ЭТО ВВОДИТСЯ В ПРОГРАММУ» (F72:F74).
	Transfer TransferResult `json:"transfer"`

	// Warnings reports workbook anomalies reproduced by this calculation.
	// It is empty for the common case Cq > 15000; see the package notes.
	Warnings []string `json:"warnings,omitempty"`
}

// TransferResult holds the three values the workbook marks for transfer into
// the OZVB program (cells F72:F74).
type TransferResult struct {
	// W0 is the chamber volume in m³ (F72).
	W0 float64 `json:"w0"`
	// Pm is the maximum mean-ballistic pressure in MPa (F73).
	Pm float64 `json:"pm"`
	// LS is the refined barrel length Lс.у in meters (F74).
	LS float64 `json:"ls"`
}

const (
	cqUpper  = 15000.0
	cqLower  = 13000.0
	cqAboveF = 1.05 // pm.кр factor for Cq > 15000 (F50)
	cqBelowF = 0.95 // pm.кр factor for Cq < 13000 (F50)
	lsAboveF = 1.1  // LS factor for Cq > 15000 (I51)
	lsBelowF = 0.9  // LS factor for Cq < 13000 (I51)

	warningLSAnomaly = "У книзі при 13000 ≤ Cq ≤ 15000 уточнене значення LS дорівнює pm.кр (комірка I51); відтворено без змін."
)

// Calculate runs the full workbook chain for the given inputs and reference
// table. The table must be the Slukhotsky table sorted by strictly ascending
// Ce; pass DefaultTable1 for the data of Диплом.xls.
func Calculate(in Inputs, table Table1) (Result, error) {
	if err := in.Validate(); err != nil {
		return Result{}, err
	}
	if err := table.Validate(); err != nil {
		return Result{}, err
	}

	res := Result{}
	// Initial quantities (rows 13-20).
	dm := in.D * 0.001
	res.Ed = in.Q * math.Pow(in.Vd, 2) / 2
	res.EdMJ = res.Ed / 1000000.0
	res.Cq = in.Q / math.Pow(dm, 3)
	res.Ce = res.Ed / math.Pow(dm, 3)
	res.CeMJ = res.Ce / 1000000.0
	res.S = in.Ns * math.Pow(dm, 2)

	// Table lookup with linear interpolation (rows 21-29).
	idx, err := table.MatchIndex(res.CeMJ)
	if err != nil {
		return Result{}, err
	}
	res.TableIndex = idx
	res.PmKr = table.Interpolate(idx, res.CeMJ, func(r Table1Row) float64 { return r.PmKr })
	res.Delta = table.Interpolate(idx, res.CeMJ, func(r Table1Row) float64 { return r.Delta })
	res.Hi = table.Interpolate(idx, res.CeMJ, func(r Table1Row) float64 { return r.Hi })
	res.Kisp = table.Interpolate(idx, res.CeMJ, func(r Table1Row) float64 { return r.Kisp })
	res.LstvD = table.Interpolate(idx, res.CeMJ, func(r Table1Row) float64 { return r.LstvD })

	// Intermediate quantities (rows 31-44).
	res.Omega = res.Ed / (res.Kisp * 1000000.0)
	res.OmegaG = res.Omega * 1000.0
	res.W0 = res.Omega / res.Delta
	res.W0L = res.W0 * 1000.0
	res.L0 = res.W0 / res.S
	res.L0MM = res.L0 * 1000.0
	res.LcCalc = res.LstvD * in.D / 1000.0
	res.LcMM = res.LcCalc * 1000.0
	res.LS = res.LcCalc - 1.5*in.D*0.001
	res.Lkam = res.L0 / res.Hi
	res.LkamMM = res.Lkam * 1000.0
	res.Ld = res.LS - res.Lkam
	res.LdMM = res.Ld * 1000.0
	res.LambdaD = res.Ld / res.L0

	// Refined values with the Cq correction (rows 50-52).
	switch {
	case res.Cq > cqUpper:
		res.PmKrU = res.PmKr * cqAboveF
		res.LSu = res.LS * lsAboveF
	case res.Cq < cqLower:
		res.PmKrU = res.PmKr * cqBelowF
		res.LSu = res.LS * lsBelowF
	default:
		res.PmKrU = res.PmKr
		res.LSu = res.PmKr // workbook I51 else-branch, reproduced intentionally
		res.Warnings = append(res.Warnings, warningLSAnomaly)
	}
	res.LcU = res.LSu
	res.LdU = res.LSu - res.Lkam
	res.LambdaDU = res.LdU / res.L0

	// Final quantities (rows 55-67).
	res.Lambda1 = (res.LambdaDU + 1.0/res.Hi) / (res.LambdaDU + 1.0)
	res.Fi = in.Af + 1.0/3.0*res.Lambda1*res.Omega/in.Q
	bracket := 1.0 + 1.0/2.0*res.Lambda1*res.Omega/in.Q
	res.Pm = 1.12 * res.Fi * res.PmKrU / (in.F1 * bracket)
	res.EttaD = res.Fi * res.Ed / (res.Pm * 1000000.0 * res.S * res.LdU)

	// Transfer block (rows 72-74).
	res.Transfer = TransferResult{W0: res.W0, Pm: res.Pm, LS: res.LcU}

	if err := res.Validate(); err != nil {
		return Result{}, err
	}
	return res, nil
}

// Validate rejects any non-finite result value; a stored or transmitted
// result must re-pass it after decoding.
func (r Result) Validate() error {
	fields := []struct {
		name string
		v    float64
	}{
		{"Eд", r.Ed}, {"Cq", r.Cq}, {"Ce", r.Ce}, {"S", r.S},
		{"pm.кр", r.PmKr}, {"Delta", r.Delta}, {"Hi", r.Hi},
		{"Kисп", r.Kisp}, {"Lств/d", r.LstvD},
		{"Omega", r.Omega}, {"W0", r.W0}, {"l0", r.L0},
		{"Lств", r.LcCalc}, {"LS", r.LS}, {"lкам", r.Lkam}, {"lд", r.Ld},
		{"Λд", r.LambdaD}, {"pm.кр.у", r.PmKrU}, {"Lс.у", r.LcU},
		{"lд.у", r.LdU}, {"Λ1", r.Lambda1}, {"φ", r.Fi}, {"pm", r.Pm},
		{"ηд", r.EttaD},
	}
	for _, f := range fields {
		if !finite(f.v) {
			return newError(ErrNonFinite, "Обчислене значення %s не є скінченним числом", f.name)
		}
	}
	return nil
}

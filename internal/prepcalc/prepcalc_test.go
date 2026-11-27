package prepcalc

import (
	"math"
	"strings"
	"testing"
)

// workbookInputs are the yellow input cells of Диплом.xls (Расчеты!D7:D10, H7:H9).
var workbookInputs = Inputs{D: 5.45, Q: 0.0034, Vd: 960, Lc: 0.508, Ns: 0.83, F1: 1.05, Af: 1.1}

// workbookResult holds the saved formula values of Расчеты for the workbook
// inputs. Source: extracted via xlrd from Диплом.xls (SHA-256
// 7445b239...8BF0E2952); see docs/PREPARATION_XLS_AUDIT.md.
var workbookResult = Result{
	Ed:   1566.7199999999998,
	EdMJ: 0.0015667199999999997,
	Cq:   21003.390657660948,
	Ce:   9678362415.050163,
	CeMJ: 9678.362415050164,
	S:    2.4653074999999998e-05,

	TableIndex: 10,
	PmKr:       381.9590603762541,
	Delta:      740.0,
	Hi:         1.9381871139565212,
	Kisp:       1.0664327516989969,
	LstvD:      70.35672483010033,

	Omega:   0.0014691221715611845,
	OmegaG:  1.4691221715611846,
	W0:      1.9853002318394384e-06,
	W0L:     0.0019853002318394384,
	L0:      0.08052951738634789,
	L0MM:    80.52951738634789,
	LcCalc:  0.38344415032404683,
	LcMM:    383.4441503240468,
	LS:      0.37526915032404684,
	Lkam:    0.04154888700191533,
	LkamMM:  41.548887001915325,
	Ld:      0.3337202633221315,
	LdMM:    333.72026332213153,
	LambdaD: 4.1440738024180295,

	PmKrU:    401.05701339506686,
	LSu:      0.4127960653564516,
	LcU:      0.4127960653564516,
	LdU:      0.37124717835453624,
	LambdaDU: 4.610075788402446,

	Lambda1: 0.9137170404053115,
	Fi:      1.2316041139796776,
	Pm:      440.01195661147534,
	EttaD:   0.47914149328454253,

	Transfer: TransferResult{
		W0: 1.9853002318394384e-06,
		Pm: 440.01195661147534,
		LS: 0.4127960653564516,
	},
}

const goldenRelTol = 1e-12

func relDiff(got, want float64) float64 {
	if got == want {
		return 0
	}
	return math.Abs(got-want) / math.Abs(want)
}

func fields(r Result) []struct {
	name string
	got  float64
	want float64
} {
	return []struct {
		name string
		got  float64
		want float64
	}{
		{"Ed", r.Ed, workbookResult.Ed},
		{"EdMJ", r.EdMJ, workbookResult.EdMJ},
		{"Cq", r.Cq, workbookResult.Cq},
		{"Ce", r.Ce, workbookResult.Ce},
		{"CeMJ", r.CeMJ, workbookResult.CeMJ},
		{"S", r.S, workbookResult.S},
		{"PmKr", r.PmKr, workbookResult.PmKr},
		{"Delta", r.Delta, workbookResult.Delta},
		{"Hi", r.Hi, workbookResult.Hi},
		{"Kisp", r.Kisp, workbookResult.Kisp},
		{"LstvD", r.LstvD, workbookResult.LstvD},
		{"Omega", r.Omega, workbookResult.Omega},
		{"OmegaG", r.OmegaG, workbookResult.OmegaG},
		{"W0", r.W0, workbookResult.W0},
		{"W0L", r.W0L, workbookResult.W0L},
		{"L0", r.L0, workbookResult.L0},
		{"L0MM", r.L0MM, workbookResult.L0MM},
		{"LcCalc", r.LcCalc, workbookResult.LcCalc},
		{"LcMM", r.LcMM, workbookResult.LcMM},
		{"LS", r.LS, workbookResult.LS},
		{"Lkam", r.Lkam, workbookResult.Lkam},
		{"LkamMM", r.LkamMM, workbookResult.LkamMM},
		{"Ld", r.Ld, workbookResult.Ld},
		{"LdMM", r.LdMM, workbookResult.LdMM},
		{"LambdaD", r.LambdaD, workbookResult.LambdaD},
		{"PmKrU", r.PmKrU, workbookResult.PmKrU},
		{"LSu", r.LSu, workbookResult.LSu},
		{"LcU", r.LcU, workbookResult.LcU},
		{"LdU", r.LdU, workbookResult.LdU},
		{"LambdaDU", r.LambdaDU, workbookResult.LambdaDU},
		{"Lambda1", r.Lambda1, workbookResult.Lambda1},
		{"Fi", r.Fi, workbookResult.Fi},
		{"Pm", r.Pm, workbookResult.Pm},
		{"EttaD", r.EttaD, workbookResult.EttaD},
		{"Transfer.W0", r.Transfer.W0, workbookResult.Transfer.W0},
		{"Transfer.Pm", r.Transfer.Pm, workbookResult.Transfer.Pm},
		{"Transfer.LS", r.Transfer.LS, workbookResult.Transfer.LS},
	}
}

// TestWorkbookGolden reproduces every saved formula value of the workbook.
func TestWorkbookGolden(t *testing.T) {
	got, err := Calculate(workbookInputs, DefaultTable1())
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if got.TableIndex != workbookResult.TableIndex {
		t.Errorf("TableIndex = %d, want %d", got.TableIndex, workbookResult.TableIndex)
	}
	bitIdentical := 0
	checked := fields(got)
	for _, f := range checked {
		if f.got == f.want {
			bitIdentical++
			continue
		}
		if d := relDiff(f.got, f.want); d > goldenRelTol {
			t.Errorf("%s = %.17g, want %.17g (rel diff %.3e)", f.name, f.got, f.want, d)
		}
	}
	t.Logf("bit-identical: %d/%d", bitIdentical, len(checked))
	if len(got.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none outside the Cq correction band", got.Warnings)
	}
}

// TestCqCorrectionBranches checks the F50/I51 branches below and inside the
// correction band 13000..15000; the in-band LS anomaly must be reported.
func TestCqCorrectionBranches(t *testing.T) {
	t.Run("below13000", func(t *testing.T) {
		got, err := Calculate(Inputs{D: 70, Q: 4.0, Vd: 400, Lc: 0, Ns: 0.83, F1: 1.05, Af: 1.1}, DefaultTable1())
		if err != nil {
			t.Fatalf("Calculate: %v", err)
		}
		if got.Cq >= 13000 {
			t.Fatalf("test setup: Cq = %g, want < 13000", got.Cq)
		}
		if d := relDiff(got.PmKrU, 155.7667638483965); d > goldenRelTol {
			t.Errorf("PmKrU = %.17g, want %.17g", got.PmKrU, 155.7667638483965)
		}
		if d := relDiff(got.LSu, 0.9888061224489795); d > goldenRelTol {
			t.Errorf("LSu = %.17g, want %.17g", got.LSu, 0.9888061224489795)
		}
		for _, w := range got.Warnings {
			if strings.Contains(w, "I51") {
				t.Errorf("unexpected LS anomaly warning: %v", got.Warnings)
			}
		}
	})
	t.Run("inside13000to15000", func(t *testing.T) {
		got, err := Calculate(Inputs{D: 63, Q: 3.4, Vd: 400, Lc: 0, Ns: 0.83, F1: 1.05, Af: 1.1}, DefaultTable1())
		if err != nil {
			t.Fatalf("Calculate: %v", err)
		}
		if got.Cq <= 13000 || got.Cq >= 15000 {
			t.Fatalf("test setup: Cq = %g, want inside (13000, 15000)", got.Cq)
		}
		// Workbook I51 else-branch: LSu equals PmKr, not LS.
		if got.LSu != got.PmKr {
			t.Errorf("LSu = %.17g, want the workbook anomaly value PmKr = %.17g", got.LSu, got.PmKr)
		}
		if d := relDiff(got.PmKrU, 172.19488736117611); d > goldenRelTol {
			t.Errorf("PmKrU = %.17g, want %.17g", got.PmKrU, 172.19488736117611)
		}
		warned := false
		for _, w := range got.Warnings {
			if strings.Contains(w, "I51") {
				warned = true
			}
		}
		if !warned {
			t.Errorf("Warnings = %v, want the I51 anomaly notice", got.Warnings)
		}
	})
}

// TestExactKnot checks interpolation at a table knot with a synthetic table
// whose arithmetic is exact, plus the knot-adjacent behavior of Calculate.
func TestExactKnot(t *testing.T) {
	tbl := Table1{
		{Ce: 1, PmKr: 10, Delta: 100, Hi: 1.5, Kisp: 1.0, LstvD: 20},
		{Ce: 2, PmKr: 20, Delta: 200, Hi: 2.5, Kisp: 2.0, LstvD: 30},
		{Ce: 3, PmKr: 30, Delta: 300, Hi: 3.5, Kisp: 3.0, LstvD: 40},
	}
	idx, err := tbl.MatchIndex(2)
	if err != nil || idx != 2 {
		t.Fatalf("MatchIndex(2) = %d, %v; want 2, nil", idx, err)
	}
	if got := tbl.Interpolate(idx, 2, func(r Table1Row) float64 { return r.PmKr }); got != 20 {
		t.Errorf("Interpolate at the knot = %g, want 20", got)
	}

	// The intended knot CeMJ = 5000 is unreachable exactly in binary64
	// (Ce = 5e6/(0.1)^3 lands just below), so the bracket ends at the knot
	// and the values match the knot row within roundoff.
	got, err := Calculate(Inputs{D: 100, Q: 15.625, Vd: 800, Lc: 0, Ns: 0.83, F1: 1.02, Af: 1.1}, DefaultTable1())
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if got.TableIndex != 5 {
		t.Fatalf("TableIndex = %d, want 5", got.TableIndex)
	}
	knot := DefaultTable1()[5]
	for _, f := range []struct {
		name string
		got  float64
		want float64
	}{
		{"PmKr", got.PmKr, knot.PmKr},
		{"Delta", got.Delta, knot.Delta},
		{"Hi", got.Hi, knot.Hi},
		{"Kisp", got.Kisp, knot.Kisp},
		{"LstvD", got.LstvD, knot.LstvD},
	} {
		if d := relDiff(f.got, f.want); d > goldenRelTol {
			t.Errorf("%s = %.17g, want knot value %.17g (rel diff %.3e)", f.name, f.got, f.want, d)
		}
	}
}

// TestTableRangeErrors mirrors workbook #N/A below the table and #REF! at or
// above its last argument.
func TestTableRangeErrors(t *testing.T) {
	below := Inputs{D: 200, Q: 0.1, Vd: 100, Lc: 0, Ns: 0.83, F1: 1.05, Af: 1.1}
	if _, err := Calculate(below, DefaultTable1()); err == nil {
		t.Fatal("expected an error for Ce below the table")
	} else if e, ok := err.(*Error); !ok || e.Kind != ErrTableRange {
		t.Fatalf("err = %v, want kind %s", err, ErrTableRange)
	}
	// Ce = Cq*Vd^2/2: with Cq = 21003.39, Vd = 1200 gives CeMJ ≈ 15124 > 11000.
	above := Inputs{D: 5.45, Q: 0.0034, Vd: 1200, Lc: 0, Ns: 0.83, F1: 1.05, Af: 1.1}
	if _, err := Calculate(above, DefaultTable1()); err == nil {
		t.Fatal("expected an error for Ce above the table")
	} else if e, ok := err.(*Error); !ok || e.Kind != ErrTableRange {
		t.Fatalf("err = %v, want kind %s", err, ErrTableRange)
	}
	// Ce = Cq*Vd²/2: with Cq = 34400, Vd = 800 gives CeMJ = 11008 > 11000.
	atMax := Inputs{D: 100, Q: 34.4, Vd: 800, Lc: 0, Ns: 0.83, F1: 1.02, Af: 1.1}
	if _, err := Calculate(atMax, DefaultTable1()); err == nil {
		t.Fatal("expected an error for Ce at the last table row")
	} else if e, ok := err.(*Error); !ok || e.Kind != ErrTableRange {
		t.Fatalf("err = %v, want kind %s", err, ErrTableRange)
	}
}

// TestValidate covers input and table validation.
func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		in      Inputs
		wantKnd ErrorKind
	}{
		{"nan", Inputs{D: math.NaN(), Q: 1, Vd: 1, Ns: 1, F1: 1, Af: 1}, ErrInvalidInput},
		{"inf", Inputs{D: 1, Q: math.Inf(1), Vd: 1, Ns: 1, F1: 1, Af: 1}, ErrInvalidInput},
		{"negative d", Inputs{D: -5.45, Q: 1, Vd: 1, Ns: 1, F1: 1, Af: 1}, ErrInvalidInput},
		{"zero q", Inputs{D: 5.45, Q: 0, Vd: 1, Ns: 1, F1: 1, Af: 1}, ErrInvalidInput},
		{"zero ns", Inputs{D: 5.45, Q: 1, Vd: 1, Ns: 0, F1: 1, Af: 1}, ErrInvalidInput},
	}
	for _, tc := range cases {
		if _, err := Calculate(tc.in, DefaultTable1()); err == nil {
			t.Errorf("%s: expected an error", tc.name)
		} else if e, ok := err.(*Error); !ok || e.Kind != tc.wantKnd {
			t.Errorf("%s: err = %v, want kind %s", tc.name, err, tc.wantKnd)
		}
	}

	unsorted := DefaultTable1()
	unsorted[3], unsorted[4] = unsorted[4], unsorted[3]
	if err := unsorted.Validate(); err == nil {
		t.Error("expected an error for an unsorted table")
	}
	if _, err := Calculate(workbookInputs, unsorted); err == nil {
		t.Error("expected Calculate to reject an unsorted table")
	}
	if err := (Table1{defaultTable1[0]}).Validate(); err == nil {
		t.Error("expected an error for a one-row table")
	}
}

// TestDefaultTable1IsACopy guards the shared default table against mutation.
func TestDefaultTable1IsACopy(t *testing.T) {
	tbl := DefaultTable1()
	tbl[0].Ce = 12345
	if defaultTable1[0].Ce != 500 {
		t.Fatalf("default table mutated: %v", defaultTable1[0])
	}
}

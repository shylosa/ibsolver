package chartdata

import (
	"context"
	"fmt"
	"math"

	"ibsolver/internal/domain"
)

// Ballistics solves the historical interior ballistics initial-value problem
// from OZVB-3A and the Slukhotsky model, yielding 6 time-series curves:
// P (pressure), T (temperature), V (velocity), L (travel), PSI (relative burnt fraction),
// and G (gas inflow rate).
func Ballistics(ctx context.Context, input domain.PreparationInputs) (domain.Run, error) {
	if err := input.Validate(); err != nil {
		return domain.Run{}, err
	}

	// 1. Resolve inputs with historical OZVB-3A defaults
	pmax := 270e6
	d := 0.1524
	q := 43.6
	vdz := 810.0
	ls := 6.275
	ns := 0.82
	hi := 1.285
	afi := 1.06
	pfor := 25e6
	kt := 1.8e-3
	tn := 15.0
	omega := 11.64
	w0 := 0.0171
	ro := 1620.0
	tg15 := 2700.0
	ku1 := 0.0
	f15 := 1e6
	u15 := 0.65e-9
	svod := 0.0012
	pvos := 10e6
	tvos := 1600.0

	if input.Pmax != nil && *input.Pmax > 0 {
		pmax = *input.Pmax
	}
	if input.D != nil && *input.D > 0 {
		d = *input.D
		if d > 1.0 { // safety: if accidentally given in mm
			d = d / 1000.0
		}
	}
	if input.Q != nil && *input.Q > 0 {
		q = *input.Q
	}
	if input.Vd != nil && *input.Vd > 0 {
		vdz = *input.Vd
	}
	if input.Lc != nil && *input.Lc > 0 {
		ls = *input.Lc
	}
	if input.Ns != nil && *input.Ns > 0 {
		ns = *input.Ns
	}
	if input.F1 != nil && *input.F1 > 0 {
		f15 = *input.F1
		if f15 < 100.0 { // entered as MJ/kg
			f15 = f15 * 1e6
		}
	}
	if input.Af != nil && *input.Af > 0 {
		afi = *input.Af
	}
	if input.Pfor != nil && *input.Pfor > 0 {
		pfor = *input.Pfor
	}
	if input.Hi != nil && *input.Hi > 0 {
		hi = *input.Hi
	}
	if input.Kt != nil && *input.Kt > 0 {
		kt = *input.Kt
	}
	if input.Tn != nil {
		tn = *input.Tn
	}
	if input.Omega != nil && *input.Omega > 0 {
		omega = *input.Omega
	}
	if input.W0 != nil && *input.W0 > 0 {
		w0 = *input.W0
	}
	if input.Ro != nil && *input.Ro > 0 {
		ro = *input.Ro
	}
	if input.Tg15 != nil && *input.Tg15 > 0 {
		tg15 = *input.Tg15
	}
	if input.Ku1 != nil {
		ku1 = *input.Ku1
	}
	if input.U15 != nil && *input.U15 > 0 {
		u15 = *input.U15
	}
	if input.Svod != nil && *input.Svod > 0 {
		svod = *input.Svod
	}
	if input.Pvos != nil && *input.Pvos > 0 {
		pvos = *input.Pvos
	}
	if input.Tvos != nil && *input.Tvos > 0 {
		tvos = *input.Tvos
	}

	// 2. Derive charge and geometry parameters
	s := ns * d * d
	l0 := w0 / s
	lkam := l0 / hi
	ld := ls - lkam
	if ld <= 0 {
		ld = ls * 0.85
		lkam = ls - ld
	}
	lad := ld / l0
	forma := 6
	if input.Forma != nil && *input.Forma >= 1 && *input.Forma <= 7 {
		forma = *input.Forma
	}

	kappa, lambda, mu := 1.0267, -0.026, 0.0
	sis, psis, zk := 1.0, 0.0, 1.0
	kap2, lam2 := 0.0, 0.0
	switch forma {
	case 1:
		kappa, lambda, mu = 3.0, -1.0, 1.0/3.0
	case 2:
		kappa, lambda, mu = 2.1, -0.5714, 0.04762
	case 3:
		kappa, lambda, mu = 1.2, -0.175, 0.0083
	case 4:
		kappa, lambda, mu = 1.06, -0.0571, 0.000472
	case 5:
		kappa, lambda, mu = 1.394, -0.283, 0.0
	case 6:
		kappa, lambda, mu = 1.0267, -0.026, 0.0
	case 7:
		kappa, lambda, mu = 0.714, 0.2216, -0.0243
		sis, psis, zk = 1.3703, 0.8549, 1.532
		kap2, lam2 = 0.5455, -0.94
	}
	ek := svod
	if forma == 7 {
		ek = svod * zk
	}

	c4 := omega / q / 3.0
	fi := afi + c4*(lad+1.0/hi)/(lad+1.0)

	// Thermodynamic constants
	deltaT := tn - 15.0
	tgor := tg15 + deltaT
	rGas := f15 / tg15
	fPow := rGas * tgor
	kovol := 950.0 / fPow
	teta := 19e4 / fPow
	kAd := teta + 1.0
	c0 := omega / ro
	c1 := kovol - 1.0/ro
	c2 := kAd * s
	c3 := teta * s
	u1 := u15 / (1.0 - kt*deltaT)
	if ku1 > 0 {
		_ = ku1 // reserved for U1 term below
	}
	c5 := 2.0 * lambda
	c6 := 3.0 * mu
	c7 := kappa * omega / svod
	c8 := s / (fi * q)
	c9 := 0.3e-3 * math.Pow(d, 0.8)
	gmax := 2.0 * omega * vdz / ld

	// 3. Numerical integration with event handling
	y := [6]float64{pvos, tvos, 0.0, 0.0, 0.0, 0.0}
	t := 0.0
	dt := 1e-6
	move := false
	kg := false

	type point struct {
		t   float64
		p   float64
		ts  float64
		v   float64
		l   float64
		psi float64
		g   float64
		e   float64
	}

	var rawPoints []point
	var pMax float64
	var lkg float64

	rhs := func(currY [6]float64, moveFlag, kgFlag bool) ([6]float64, float64, float64) {
		p, temp, e, _, v, l := currY[0], currY[1], currY[2], currY[3], currY[4], currY[5]
		if !moveFlag {
			v, l = 0.0, 0.0
		}
		var psi, g, f3 float64
		if kgFlag || e >= ek {
			g = 0.0
			psi = 1.0
			f3 = 0.0
		} else {
			z := e / svod
			var sigma float64
			if forma == 7 {
				z7 := z - 1.0
				lz7 := lam2 * z7
				sigma = sis * (1.0 + 2.0*lz7)
				psi = psis + kap2*z7*(1.0+lz7)
			} else {
				sigma = 1.0 + c5*z + c6*z*z
				psi = kappa * z * (1.0 + lambda*z + mu*z*z)
			}
			if psi > 1.0 {
				psi = 1.0
			}
			u1f := u1 * (1.0 + ku1*(e/svod))
			f3 = u1f * p
			g = c7 * sigma * f3
		}

		w := w0 - c0*(1.0-psi) + s*l
		if w <= 1e-12 {
			w = 1e-12
		}
		a1 := (rGas*temp + kovol*p) / w / rGas
		a2 := p * v
		if a2 < 0 {
			a2 = 0
		}
		f4 := c9 * math.Pow(a2, 0.8) * temp * (l + l0)
		poteriq := teta * f4

		var f5, f6 float64
		if !moveFlag {
			f5, f6 = 0.0, 0.0
		} else {
			f5 = c8 * p
			f6 = v
		}

		f1Deriv := a1 / temp * ((fPow+c1*p)*g - c2*a2 - poteriq)
		f2Deriv := a1 / p * ((fPow-rGas*temp)*g - c3*a2 - poteriq)

		return [6]float64{f1Deriv, f2Deriv, f3, f4, f5, f6}, psi, g
	}

	maxSteps := 500000
	stepCount := 0
	lastL, lastV, lastT := 0.0, 0.0, 0.0

	for stepCount < maxSteps {
		if err := ctx.Err(); err != nil {
			return domain.Run{}, err
		}

		p := y[0]
		if p > pMax {
			pMax = p
		}
		if !move && p > pfor {
			move = true
		}
		if !kg && y[2] >= ek {
			kg = true
			lkg = y[5]
		}

		derivs, psi, g := rhs(y, move, kg)

		// Record sample
		if stepCount%20 == 0 || y[5] >= ld {
			e := y[2]
			if e > ek { // the burn cannot advance past the burnout thickness
				e = ek
			}
			rawPoints = append(rawPoints, point{
				t:   t,
				p:   y[0] / 1e6, // store pressure in MPa for clean series scaling
				ts:  y[1],
				v:   y[4],
				l:   y[5],
				psi: psi,
				g:   g,
				e:   e,
			})
		}

		if y[5] >= ld {
			lastL = y[5]
			lastV = y[4]
			lastT = t
			break
		}

		// RK4 integration step
		k1 := derivs
		var y2, y3, y4 [6]float64
		for i := 0; i < 6; i++ {
			y2[i] = y[i] + 0.5*dt*k1[i]
		}
		k2, _, _ := rhs(y2, move, kg)

		for i := 0; i < 6; i++ {
			y3[i] = y[i] + 0.5*dt*k2[i]
		}
		k3, _, _ := rhs(y3, move, kg)

		for i := 0; i < 6; i++ {
			y4[i] = y[i] + dt*k3[i]
		}
		k4, _, _ := rhs(y4, move, kg)

		for i := 0; i < 6; i++ {
			y[i] += (dt / 6.0) * (k1[i] + 2.0*k2[i] + 2.0*k3[i] + k4[i])
		}

		t += dt
		stepCount++
	}

	if len(rawPoints) < 2 {
		return domain.Run{}, fmt.Errorf("Занадто мало точок інтегрування")
	}

	// 4. Downsample to target sample count (e.g. 501 points)
	targetSamples := 501
	if len(rawPoints) < targetSamples {
		targetSamples = len(rawPoints)
	}

	times := make([]float64, targetSamples)
	seriesP := make([]float64, targetSamples)
	seriesT := make([]float64, targetSamples)
	seriesV := make([]float64, targetSamples)
	seriesL := make([]float64, targetSamples)
	seriesPsi := make([]float64, targetSamples)
	seriesG := make([]float64, targetSamples)

	seriesE := make([]float64, targetSamples)
	seriesZ := make([]float64, targetSamples)

	for i := 0; i < targetSamples; i++ {
		idx := int(float64(i) * float64(len(rawPoints)-1) / float64(targetSamples-1))
		pt := rawPoints[idx]
		times[i] = pt.t
		seriesP[i] = pt.p
		seriesT[i] = pt.ts
		seriesV[i] = pt.v
		seriesL[i] = pt.l
		seriesPsi[i] = pt.psi
		seriesG[i] = pt.g
		seriesE[i] = pt.e * 1000
		seriesZ[i] = pt.e / svod
	}
	times[0] = 0.0
	times[len(times)-1] = rawPoints[len(rawPoints)-1].t

	// 5. Build series with historical reference values
	refP := pmax / 1e6
	refT := tgor
	refV := vdz
	refL := ld
	refPsi := 1.0
	refG := gmax

	chart := domain.ChartData{
		Time: times,
		Series: []domain.Series{
			{ID: domain.SeriesID(domain.ModePreparation, 0), Reference: &refP, Name: "Тиск P", Unit: "МПа", Values: seriesP},
			{ID: domain.SeriesID(domain.ModePreparation, 1), Reference: &refT, Name: "Температура T", Unit: "К", Values: seriesT},
			{ID: domain.SeriesID(domain.ModePreparation, 2), Reference: &refV, Name: "Швидкість V", Unit: "м/с", Values: seriesV},
			{ID: domain.SeriesID(domain.ModePreparation, 3), Reference: &refL, Name: "Переміщення L", Unit: "м", Values: seriesL},
			{ID: domain.SeriesID(domain.ModePreparation, 4), Reference: &refPsi, Name: "Частка згорілого Ψ", Unit: "частка", Values: seriesPsi},
			{ID: domain.SeriesID(domain.ModePreparation, 5), Reference: &refG, Name: "Газоприплив G", Unit: "кг/с", Values: seriesG},
		},
		Aux: []domain.Series{
			{ID: domain.SeriesID(domain.ModePreparation, 6), Name: "Товщина згорілого шару E", Unit: "мм", Values: seriesE},
			{ID: domain.SeriesID(domain.ModePreparation, 7), Name: "Відносна товщина Z", Unit: "частка", Values: seriesZ},
		},
	}

	// 6. Calculate barrel mass estimation (Lame formulas from BAS lines 175-188)
	sigmaE := 900e6
	s3 := 3.0 * sigmaE
	nk, nc, nd := 1.4, 1.2, 1.9
	ksogl := 2.4
	pi4 := math.Pi / 4.0

	denomK := s3 - 4.0*nk*pMax
	if denomK <= 0 {
		denomK = 1e7
	}
	dkn := d * math.Sqrt(hi) * math.Sqrt((s3+2.0*nk*pMax)/denomK)
	vmk := pi4 * (dkn*dkn - d*d*hi) * lkam

	denomC := s3 - 4.0*nc*pMax
	if denomC <= 0 {
		denomC = 1e7
	}
	dcn := d * math.Sqrt((s3+2.0*nc*pMax)/denomC)
	lm := ld * 0.4
	vconc := pi4 * lm * (dkn*dkn + dcn*dcn + dkn*dcn) / 3.0
	vmc := vconc - pi4*d*d*lm

	pd := y[0]
	denomD := s3 - 4.0*nd*pd
	if denomD <= 0 {
		denomD = 1e7
	}
	ddn := d * math.Sqrt((s3+2.0*nd*pd)/denomD)
	vkond := pi4 * (ld - lm) * (dcn*dcn + ddn*ddn + dcn*ddn) / 3.0
	vmd := vkond - pi4*d*d*(ld-lm)

	summaVol := math.Abs(vmk + vmc + vmd)
	qstv := ksogl * 7600.0 * summaVol

	etak := 0.0
	if ld > 0 && lkg > 0 {
		etak = lkg / ld
	}

	summary := &domain.BallisticsSummary{
		MaxPressure:    pMax / 1e6,
		MuzzleVelocity: lastV,
		MuzzleTime:     lastT,
		BarrelTravel:   lastL,
		BurntFraction:  etak,
		ChamberVolume:  w0,
		ChargeMass:     omega,
		BarrelMass:     qstv,
	}

	run := domain.Run{
		Mode:          domain.ModePreparation,
		Snapshot:      domain.Parameters{Preparation: &input},
		ResultVersion: domain.BallisticsVersion,
		Summary: domain.Summary{
			Kind:       "BALLISTICS",
			Samples:    targetSamples,
			Duration:   lastT,
			Ballistics: summary,
		},
		Chart: chart,
	}

	if err := run.Validate(); err != nil {
		return domain.Run{}, fmt.Errorf("Некоректний балістичний результат: %w", err)
	}

	return run, nil
}

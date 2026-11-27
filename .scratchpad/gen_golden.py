# -*- coding: utf-8 -*-
"""Generate Go golden-test literals for the prepcalc package (3 Cq branches)."""
import math

table = [
    (500.0, 125.0, 470.0, 1.01, 1.36, 12.0),
    (1000.0, 170.0, 510.0, 1.02, 1.34, 18.0),
    (2000.0, 195.0, 580.0, 1.09, 1.30, 29.0),
    (3000.0, 220.0, 640.0, 1.18, 1.27, 39.0),
    (4000.0, 245.0, 680.0, 1.28, 1.23, 47.0),
    (5000.0, 270.0, 710.0, 1.39, 1.20, 53.0),
    (6000.0, 295.0, 720.0, 1.50, 1.16, 59.0),
    (7000.0, 320.0, 740.0, 1.61, 1.13, 63.0),
    (8000.0, 340.0, 740.0, 1.73, 1.11, 66.0),
    (9000.0, 365.0, 740.0, 1.85, 1.08, 69.0),
    (10000.0, 390.0, 740.0, 1.98, 1.06, 71.0),
    (11000.0, 415.0, 750.0, 2.07, 1.03, 73.0),
]
arg_col = [r[0] for r in table]

class Err(Exception):
    pass

def calc(d, q, vd, lc, ns, f1, af):
    def match1(x):
        idx = 0
        for i, v in enumerate(arg_col):
            if v <= x:
                idx = i + 1
            else:
                break
        if idx == 0:
            raise Err("below")
        if idx == len(arg_col):
            raise Err("above")
        return idx

    dm = d * 0.001
    ed = q * vd ** 2 / 2
    edMJ = ed / 1000000.0
    cq = q / dm ** 3
    ce = ed / dm ** 3
    ceMJ = ce / 1000000.0
    s = ns * dm ** 2
    g24 = ceMJ
    i25 = match1(g24)
    def interp(col):
        y1, y2 = arg_col[i25 - 1], arg_col[i25]
        x1, x2 = col[i25 - 1], col[i25]
        return (g24 - y1) * (x2 - x1) / (y2 - y1) + x1
    cols = list(zip(*table))
    pmkr = interp(cols[1]); delta = interp(cols[2]); hi = interp(cols[3])
    kisp = interp(cols[4]); lstvd = interp(cols[5])

    omega = ed / (kisp * 1000000.0)
    omegaG = omega * 1000.0
    w0 = omega / delta
    w0l = w0 * 1000.0
    l0 = w0 / s
    l0mm = l0 * 1000.0
    lcc = lstvd * d / 1000.0
    lcmm = lcc * 1000.0
    ls = lcc - 1.5 * d * 0.001
    lkam = l0 / hi
    lkamm = lkam * 1000.0
    ld = ls - lkam
    ldmm = ld * 1000.0
    lamd = ld / l0

    if cq > 15000.0:
        pmkru = pmkr * 1.05
    elif cq < 13000.0:
        pmkru = pmkr * 0.95
    else:
        pmkru = pmkr
    if cq > 15000.0:
        lsu = ls * 1.1
    elif cq < 13000.0:
        lsu = ls * 0.9
    else:
        lsu = pmkr
    lcu = lsu
    ldu = lsu - lkam
    lamdu = ldu / l0
    lam1 = (lamdu + 1.0 / hi) / (lamdu + 1.0)
    fi = af + 1.0 / 3.0 * lam1 * omega / q
    br = 1.0 + 1.0 / 2.0 * lam1 * omega / q
    pm = 1.12 * fi * pmkru / (f1 * br)
    ettad = fi * ed / (pm * 1000000.0 * s * ldu)
    return dict(
        Ed=ed, EdMJ=edMJ, Cq=cq, Ce=ce, CeMJ=ceMJ, S=s, TableIndex=i25,
        PmKr=pmkr, Delta=delta, Hi=hi, Kisp=kisp, LstvD=lstvd,
        Omega=omega, OmegaG=omegaG, W0=w0, W0L=w0l, L0=l0, L0MM=l0mm,
        LcCalc=lcc, LcMM=lcmm, LS=ls, Lkam=lkam, LkamMM=lkamm,
        Ld=ld, LdMM=ldmm, LambdaD=lamd, PmKrU=pmkru, LSu=lsu, LcU=lcu,
        LdU=ldu, LambdaDU=lamdu, Lambda1=lam1, Fi=fi, Pm=pm, EttaD=ettad,
        OutW0=w0, OutPm=pm, OutLS=lcu,
    )

def golit(v):
    r = repr(float(v))
    return r

cases = [
    ("golden", dict(d=5.45, q=0.0034, vd=960.0, lc=0.508, ns=0.83, f1=1.05, af=1.1)),
    ("lowCq",  dict(d=70.0, q=4.0, vd=400.0, lc=0.0, ns=0.83, f1=1.05, af=1.1)),
    ("midCq",  dict(d=63.0, q=3.4, vd=400.0, lc=0.0, ns=0.83, f1=1.05, af=1.1)),
]

for name, inp in cases:
    r = calc(**inp)
    print("// case %s: d=%r q=%r vd=%r  Cq=%r  CeMJ=%r" % (
        name, inp["d"], inp["q"], inp["vd"], r["Cq"], r["CeMJ"]))
    for k, v in r.items():
        print("%s: %s," % (k, golit(v)))
    print()

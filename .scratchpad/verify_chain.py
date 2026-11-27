# -*- coding: utf-8 -*-
"""Verify the reconstructed calculation chain against all saved Диплом.xls values."""
import json

# Saved workbook values (Расчеты) extracted via xlrd
saved = {
    "F13": 1566.7199999999998, "H13": 0.0015667199999999997,
    "F15": 21003.390657660948,
    "F17": 9678362415.050163, "H17": 9678.362415050164,
    "F19": 0.83, "F20": 2.4653074999999998e-05,
    "G24": 9678.362415050164,
    "F25": 381.9590603762541, "I25": 10.0,
    "J25": 365.0, "K25": 390.0, "L25": 9000.0, "M25": 10000.0,
    "F26": 740.0, "J26": 740.0, "K26": 740.0,
    "F27": 1.9381871139565212, "J27": 1.85, "K27": 1.98,
    "F28": 1.0664327516989969, "J28": 1.08, "K28": 1.06,
    "F29": 70.35672483010033, "J29": 69.0, "K29": 71.0,
    "F31": 0.0014691221715611845, "H31": 1.4691221715611846,
    "F33": 1.9853002318394384e-06, "H33": 0.0019853002318394384,
    "I34": -0.4002260438440632,
    "F35": 0.08052951738634789, "H35": 80.52951738634789,
    "F38": 0.38344415032404683, "H38": 383.4441503240468,
    "I39": 0.37526915032404684,
    "F40": 0.04154888700191533, "H40": 41.548887001915325,
    "F42": 0.3337202633221315, "H42": 333.72026332213153,
    "F44": 4.1440738024180295,
    "F50": 401.05701339506686,
    "F51": 0.4127960653564516, "I51": 0.4127960653564516,
    "F52": 0.37124717835453624, "I52": 4.610075788402446,
    "F55": 0.9137170404053115,
    "F57": 1.05, "F58": 1.1,
    "F60": 1.2316041139796776,
    "F63": 440.01195661147534,
    "F67": 0.47914149328454253,
    "F72": 1.9853002318394384e-06, "F73": 440.01195661147534,
    "F74": 0.4127960653564516,
}

# Inputs (yellow cells)
d, q, vd, lc = 5.45, 0.0034, 960.0, 0.508
ns, f1, af = 0.83, 1.05, 1.1

# Таблица 1 (Слухоцкого) rows B7:G18
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
val_cols = [
    [r[1] for r in table],  # C pm.кр
    [r[2] for r in table],  # D Delta
    [r[3] for r in table],  # E Hi
    [r[4] for r in table],  # F Kисп
    [r[5] for r in table],  # G Lств/d
]

# MATCH type 1: largest value <= x (ascending); 1-based index
def match1(x, col):
    idx = 0
    for i, v in enumerate(col):
        if v <= x:
            idx = i + 1
        else:
            break
    return idx

def interp(col_vals, i25):
    y1, y2 = arg_col[i25 - 1], arg_col[i25]      # L25, M25
    x1, x2 = col_vals[i25 - 1], col_vals[i25]    # J, K
    return (G24 - y1) * (x2 - x1) / (y2 - y1) + x1

# Chain with exact RPN grouping
Ed = q * vd ** 2 / 2                      # F13
EdMJ = Ed / 1000000.0                     # H13
Cq = q / (d * 0.001) ** 3                 # F15
Ce = Ed / (d * 0.001) ** 3                # F17
CeMJ = Ce / 1000000.0                     # H17
S = ns * (d * 0.001) ** 2                 # F20
G24 = CeMJ

i25 = match1(CeMJ, arg_col)               # I25
PmKr = interp(val_cols[0], i25)           # F25
Delta = interp(val_cols[1], i25)          # F26
Hi = interp(val_cols[2], i25)             # F27
Kisp = interp(val_cols[3], i25)           # F28
LstvD = interp(val_cols[4], i25)          # F29

Omega = Ed / (Kisp * 1000000.0)           # F31
OmegaG = Omega * 1000.0                   # H31
W0 = Omega / Delta                        # F33
W0L = W0 * 1000.0                         # H33
I34 = (15000.0 - Cq) / 15000.0            # I34
L0 = W0 / S                               # F35
L0MM = L0 * 1000.0                        # H35
LcCalc = LstvD * d / 1000.0               # F38
LcMM = LcCalc * 1000.0                    # H38
LS = LcCalc - 1.5 * d * 0.001             # I39
Lkam = L0 / Hi                            # F40
LkamMM = Lkam * 1000.0                    # H40
Ld = LS - Lkam                            # F42
LdMM = Ld * 1000.0                        # H42
LambdaD = Ld / L0                         # F44

if Cq > 15000.0:
    PmKrU = PmKr * 1.05
elif Cq < 13000.0:
    PmKrU = PmKr * 0.95
else:
    PmKrU = PmKr                          # F50
if Cq > 15000.0:
    Lsu = LS * 1.1
elif Cq < 13000.0:
    Lsu = LS * 0.9
else:
    Lsu = PmKr                            # I51 else-branch = F25 (workbook anomaly)
LcU = Lsu                                 # F51
LdU = Lsu - Lkam                          # F52
LambdaDU = LdU / L0                       # I52

Lambda1 = (LambdaDU + 1.0 / Hi) / (LambdaDU + 1.0)  # F55
Fi = af + 1.0 / 3.0 * Lambda1 * Omega / q           # F60
bracket = 1.0 + 1.0 / 2.0 * Lambda1 * Omega / q     # F63 bracket
Pm = 1.12 * Fi * PmKrU / (f1 * bracket)             # F63
EttaD = Fi * Ed / (Pm * 1000000.0 * S * LdU)        # F67

computed = {
    "F13": Ed, "H13": EdMJ, "F15": Cq, "F17": Ce, "H17": CeMJ,
    "F19": ns, "F20": S, "G24": G24, "F25": PmKr, "I25": float(i25),
    "J25": val_cols[0][i25 - 1], "K25": val_cols[0][i25],
    "L25": arg_col[i25 - 1], "M25": arg_col[i25],
    "F26": Delta, "J26": val_cols[1][i25 - 1], "K26": val_cols[1][i25],
    "F27": Hi, "J27": val_cols[2][i25 - 1], "K27": val_cols[2][i25],
    "F28": Kisp, "J28": val_cols[3][i25 - 1], "K28": val_cols[3][i25],
    "F29": LstvD, "J29": val_cols[4][i25 - 1], "K29": val_cols[4][i25],
    "F31": Omega, "H31": OmegaG, "F33": W0, "H33": W0L, "I34": I34,
    "F35": L0, "H35": L0MM, "F38": LcCalc, "H38": LcMM, "I39": LS,
    "F40": Lkam, "H40": LkamMM, "F42": Ld, "H42": LdMM, "F44": LambdaD,
    "F50": PmKrU, "F51": LcU, "I51": Lsu, "F52": LdU, "I52": LambdaDU,
    "F55": Lambda1, "F57": f1, "F58": af, "F60": Fi, "F63": Pm,
    "F67": EttaD, "F72": W0, "F73": Pm, "F74": LcU,
}

worst = (0.0, None)
fails = []
for cell, expect in saved.items():
    got = computed[cell]
    denom = max(abs(expect), 1e-300)
    rel = abs(got - expect) / denom
    if rel > worst[0]:
        worst = (rel, cell)
    if rel > 1e-12:
        fails.append((cell, expect, got, rel))

print("cells compared:", len(saved))
print("worst rel err: %.3e at %s" % (worst[0], worst[1]))
if fails:
    for cell, expect, got, rel in fails:
        print("FAIL %s: saved=%r computed=%r rel=%.3e" % (cell, expect, got, rel))
else:
    print("ALL MATCH within 1e-12 relative")

# exact bit-level check too
exact = sum(1 for c in saved if computed[c] == saved[c])
print("bit-identical: %d/%d" % (exact, len(saved)))

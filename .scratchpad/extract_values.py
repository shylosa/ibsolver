# -*- coding: utf-8 -*-
"""Extract all cell values from Диплом.xls for Go kernel verification."""
import json
import xlrd

SRC = r"D:\go\ibsolver-refs\Диплом.xls"
OUT = r"D:\go\ibsolver\.scratchpad\xls-values.json"

book = xlrd.open_workbook(SRC, formatting_info=False)
result = {}
for sheet in book.sheets():
    rows = []
    for r in range(sheet.nrows):
        row = []
        for c in range(sheet.ncols):
            cell = sheet.cell(r, c)
            entry = {"t": cell.ctype}
            if cell.ctype == xlrd.XL_CELL_NUMBER:
                entry["v"] = cell.value
            elif cell.ctype == xlrd.XL_CELL_TEXT:
                entry["v"] = cell.value
            elif cell.ctype == xlrd.XL_CELL_BOOLEAN:
                entry["v"] = bool(cell.value)
            rows.append(row if entry else row)
            row.append(entry)
        rows.append(row)
    result[sheet.name] = rows

with open(OUT, "w", encoding="utf-8") as f:
    json.dump(result, f, ensure_ascii=False, indent=1)

# Print Расчеты populated cells and Таблица 1 data in readable form
sh = book.sheet_by_name("Расчеты")
print("=== Расчеты ===")
for r in range(sh.nrows):
    for c in range(sh.ncols):
        cell = sh.cell(r, c)
        if cell.ctype in (xlrd.XL_CELL_NUMBER, xlrd.XL_CELL_TEXT):
            addr = "%s%d" % (chr(65 + c) if c < 26 else "A" + chr(65 + c - 26), r + 1)
            if cell.ctype == xlrd.XL_CELL_NUMBER:
                print("%s\t%r" % (addr, cell.value))
            else:
                txt = cell.value.replace("\n", " | ")
                print("%s\tTEXT %s" % (addr, txt[:100]))

sh1 = book.sheet_by_name("Таблица 1 (Слухоцкого)")
print("=== Таблица 1 ===")
for r in range(sh1.nrows):
    vals = []
    for c in range(sh1.ncols):
        cell = sh1.cell(r, c)
        if cell.ctype == xlrd.XL_CELL_NUMBER:
            vals.append("%s%d=%r" % (chr(65 + c), r + 1, cell.value))
        elif cell.ctype == xlrd.XL_CELL_TEXT:
            vals.append("%s%d='%s'" % (chr(65 + c), r + 1, cell.value.replace("\n", " ")[:60]))
    if vals:
        print("\t".join(vals))

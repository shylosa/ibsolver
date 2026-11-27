#!/usr/bin/env python3
"""Extract BIFF8 formulas and defined names without modifying the source XLS."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import struct
import sys
from collections import Counter
from pathlib import Path

try:
    import olefile
    import xlrd
    import xlrd.formula as xlrd_formula
    from xlrd.book import Name
    from xlrd.formula import FMLA_TYPE_CELL, decompile_formula
    from xlrd.biffh import unpack_unicode_update_pos
except ImportError as exc:
    missing = exc.name or "required package"
    raise SystemExit(
        f"Missing dependency {missing!r}. Install the pinned packages with: "
        f'"{sys.executable}" -m pip install -r '
        f'"{Path(__file__).with_name("requirements.txt")}"'
    ) from exc


FORMULA_RECORD = 0x0006
EOF_RECORD = 0x000A
LEGACY_LABEL_TOKEN = 0x18
LEGACY_ROW_VALUE = 0x06
CELL_REF_TOKEN_BASES = {0x04, 0x05, 0x0C, 0x0D, 0x1A, 0x1B}

CELL_REFERENCE = re.compile(
    r"(?<![A-Za-z0-9_.$])"
    r"(?:(?:'(?P<quoted>(?:[^']|'')+)'|(?P<plain>[^'!]+))!)?"
    r"(?P<reference>\$?[A-Z]{1,3}\$?\d+(?::\$?[A-Z]{1,3}\$?\d+)?)(?![A-Za-z0-9_])",
    re.IGNORECASE,
)
FUNCTION_CALL = re.compile(r"(?<![\w.])([A-Za-z_][A-Za-z0-9_.]*)\s*\(")
CONDITIONAL_FUNCTIONS = {"IF", "IFS", "IFERROR", "IFNA", "CHOOSE", "SWITCH"}


class ExtractionError(Exception):
    """Raised when a BIFF record or formula cannot be safely interpreted."""


def _token_size(tokens: bytes, pos: int, biff_version: int) -> int:
    op = tokens[pos]
    opcode = op & 0x1F
    optype = (op & 0x60) >> 5
    opx = opcode + (32 if optype else 0)
    size = xlrd_formula.szdict[biff_version][opx]

    if opcode == 0x18:
        if pos + 2 > len(tokens):
            raise ExtractionError("truncated PtgElf token")
        if tokens[pos + 1] != LEGACY_ROW_VALUE:
            raise ExtractionError(
                f"unsupported PtgElf subtype 0x{tokens[pos + 1]:02X}"
            )
        return 6
    if opcode == 0x17:
        _, end = unpack_unicode_update_pos(tokens, pos + 1, lenlen=1)
        return end - pos
    if opcode == 0x19:
        if pos + 4 > len(tokens):
            raise ExtractionError("truncated PtgAttr token")
        subop, count = struct.unpack_from("<BH", tokens, pos + 1)
        return count * 2 + 6 if subop == 0x04 else 4
    if size < 1:
        raise ExtractionError(f"unsupported BIFF token 0x{op:02X}")
    return size


def _scan_tokens(tokens: bytes, biff_version: int) -> dict:
    reference_count = 0
    defined_name_indexes = []
    external_name_count = 0
    pos = 0
    while pos < len(tokens):
        size = _token_size(tokens, pos, biff_version)
        if pos + size > len(tokens):
            raise ExtractionError(f"truncated BIFF token at offset {pos}")
        op = tokens[pos]
        opcode = op & 0x1F
        optype = (op & 0x60) >> 5
        if optype and opcode in CELL_REF_TOKEN_BASES:
            reference_count += 1
        elif optype and opcode == 0x03:
            defined_name_indexes.append(
                struct.unpack_from("<H", tokens, pos + 1)[0] - 1
            )
        elif optype and opcode == 0x19:
            external_name_count += 1
        pos += size
    return {
        "cellReferenceCount": reference_count,
        "definedNameIndexes": defined_name_indexes,
        "externalNameCount": external_name_count,
    }


def _formula_records(book: xlrd.book.Book, stream: bytes) -> list[dict]:
    offsets = book._sh_abs_posn
    if len(offsets) != book.nsheets:
        raise ExtractionError("xlrd worksheet offsets do not match worksheet count")

    records = []
    for sheet_index, start in enumerate(offsets):
        end = offsets[sheet_index + 1] if sheet_index + 1 < len(offsets) else len(stream)
        pos = start
        while pos < end:
            if pos + 4 > end:
                raise ExtractionError(f"truncated BIFF record header at stream offset {pos}")
            record_id, record_size = struct.unpack_from("<HH", stream, pos)
            record_start = pos + 4
            record_end = record_start + record_size
            if record_end > len(stream) or record_end > end:
                raise ExtractionError(f"truncated BIFF record at stream offset {pos}")
            data = stream[record_start:record_end]

            if record_id == FORMULA_RECORD:
                if len(data) < 22:
                    raise ExtractionError(f"short FORMULA record at stream offset {pos}")
                row, col = struct.unpack_from("<HH", data)
                token_size = struct.unpack_from("<H", data, 20)[0]
                if token_size > len(data) - 22:
                    raise ExtractionError(f"invalid FORMULA token size at stream offset {pos}")
                tokens = data[22 : 22 + token_size]
                records.append(
                    {
                        "sheetIndex": sheet_index,
                        "row": row,
                        "column": col,
                        "tokens": tokens,
                    }
                )

            pos = record_end
            if record_id == EOF_RECORD:
                break
    return records


def _decode_formula(book: xlrd.book.Book, record: dict) -> tuple[str, list[dict]]:
    sheet = book.sheet_by_index(record["sheetIndex"])
    row = record["row"]
    col = record["column"]
    tokens = record["tokens"]
    rendered = bytearray()
    labels = []
    temporary_names = 0
    pos = 0

    while pos < len(tokens):
        size = _token_size(tokens, pos, book.biff_version)
        op = tokens[pos]
        if op & 0x1F == LEGACY_LABEL_TOKEN:
            label_row, col_flags = struct.unpack_from("<HH", tokens, pos + 2)
            label_col = col_flags & 0x3FFF
            quoted = bool((col_flags >> 14) & 1)
            relative = bool((col_flags >> 15) & 1)
            if label_row >= sheet.nrows or label_col >= sheet.ncols:
                raise ExtractionError(
                    f"PtgElfRwV label location is outside sheet: "
                    f"row={label_row + 1}, column={label_col + 1}"
                )
            label_text = sheet.cell_value(label_row, label_col)
            if not isinstance(label_text, str) or not label_text:
                raise ExtractionError(
                    f"PtgElfRwV points to a non-text label at "
                    f"{_cell_address(label_row, label_col)}"
                )
            display_label = (
                "'" + label_text.replace("'", "''") + "'" if quoted else label_text
            )
            temp_name = Name()
            temp_name.name = display_label
            temp_name.scope = -1
            book.name_obj_list.append(temp_name)
            temporary_names += 1
            name_index = len(book.name_obj_list)
            rendered.extend(b"\x43" + struct.pack("<I", name_index))
            labels.append(
                {
                    "token": "PtgElfRwV",
                    "labelCell": _cell_address(label_row, label_col),
                    "labelText": label_text,
                    "quoted": quoted,
                    "relative": relative,
                    "resolvedRange": None,
                    "note": (
                        "BIFF8 identifies the label cell but does not encode the "
                        "resolved natural-language range in this token."
                    ),
                }
            )
        else:
            rendered.extend(tokens[pos : pos + size])
        pos += size

    original_get_range_addr = xlrd_formula.get_cell_range_addr

    def get_range_addr_with_formula_origin(
        data, offset, biff_version, relative_delta, browx=None, bcolx=None
    ):
        return original_get_range_addr(
            data,
            offset,
            biff_version,
            relative_delta,
            row if browx is None else browx,
            col if bcolx is None else bcolx,
        )

    xlrd_formula.get_cell_range_addr = get_range_addr_with_formula_origin
    try:
        expression = decompile_formula(
            book,
            bytes(rendered),
            len(rendered),
            FMLA_TYPE_CELL,
            browx=row,
            bcolx=col,
        )
    finally:
        xlrd_formula.get_cell_range_addr = original_get_range_addr
        if temporary_names:
            del book.name_obj_list[-temporary_names:]

    if not expression:
        raise ExtractionError("xlrd returned an empty formula expression")
    return "=" + expression, labels


def _cell_address(row: int, col: int) -> str:
    letters = ""
    col += 1
    while col:
        col, remainder = divmod(col - 1, 26)
        letters = chr(65 + remainder) + letters
    return f"{letters}{row + 1}"


def _cell_sort_key(address: str) -> tuple[int, int]:
    letters, row, _, _ = _split_cell_reference(address)
    col = 0
    for char in letters:
        col = col * 26 + ord(char) - 64
    return row, col


def _split_cell_reference(reference: str) -> tuple[str, int, str, int]:
    first, _, last = reference.partition(":")

    def split(address: str) -> tuple[str, int]:
        match = re.fullmatch(r"\$?([A-Z]{1,3})\$?(\d+)", address, re.IGNORECASE)
        if not match:
            raise ExtractionError(f"unsupported cell reference {reference!r}")
        return match.group(1).upper(), int(match.group(2))

    first_col, first_row = split(first)
    if last:
        last_col, last_row = split(last)
    else:
        last_col, last_row = first_col, first_row
    return first_col, first_row, last_col, last_row


def _references(expression: str, current_sheet: str) -> tuple[list[dict], int]:
    text = re.sub(r'"(?:[^"]|"")*"', '""', expression[1:])
    references = []
    occurrences = 0
    for match in CELL_REFERENCE.finditer(text):
        if int(re.search(r"\d+", match.group("reference")).group()) == 0:
            continue
        occurrences += 1
        sheet = match.group("quoted") or match.group("plain") or current_sheet
        sheet = sheet.replace("''", "'").strip()
        reference = match.group("reference")
        item = {"sheet": sheet, "reference": reference}
        if item not in references:
            references.append(item)
    return references, occurrences


def _defined_name_uses(expression: str, names: list[dict]) -> list[str]:
    return sorted(_defined_name_occurrences(expression, names))


def _defined_name_occurrences(expression: str, names: list[dict]) -> Counter:
    text = re.sub(r'"(?:[^"]|"")*"', '""', expression[1:])
    found = Counter()
    for name in (item["name"] for item in names):
        found[name] = len(
            list(re.finditer(rf"(?<![\w.]){re.escape(name)}(?![\w.])", text))
        )
    return +found


def _formula_functions(expression: str) -> list[str]:
    return sorted({match.group(1).upper() for match in FUNCTION_CALL.finditer(expression)})


def _name_records(book: xlrd.book.Book) -> list[dict]:
    names = []
    for item in book.name_obj_list:
        if item.scope == -1:
            scope = {"type": "workbook"}
        else:
            scope = {
                "type": "worksheet",
                "sheet": book.sheet_names()[item.scope],
            }
        result = getattr(item, "result", None)
        refers_to = getattr(result, "text", None) if result is not None else None
        names.append(
            {
                "name": item.name,
                "scope": scope,
                "refersTo": "=" + refers_to if refers_to else None,
                "builtIn": bool(item.builtin),
                "hidden": bool(item.hidden),
                "recovery": (
                    {"status": "direct_biff8", "confidence": "high"}
                    if refers_to
                    else {
                        "status": "unresolved",
                        "confidence": "none",
                        "reason": "xlrd did not produce a definition expression.",
                    }
                ),
                "biff8TokenHex": item.raw_formula[: item.basic_formula_len].hex(),
            }
        )
    return names


def _reference_bounds(reference: dict) -> tuple[str, int, int, int, int]:
    sheet = reference["sheet"]
    first_col, first_row, last_col, last_row = _split_cell_reference(
        reference["reference"]
    )

    def col_index(letters: str) -> int:
        result = 0
        for char in letters:
            result = result * 26 + ord(char) - 64
        return result - 1

    row_min, row_max = sorted((first_row - 1, last_row - 1))
    col_min, col_max = sorted((col_index(first_col), col_index(last_col)))
    return sheet, row_min, row_max, col_min, col_max


def _cycle_check(formulas: list[dict], names: list[dict]) -> list[list[str]]:
    formulas_by_cell = {
        (item["sheet"], item["cell"]): item
        for item in formulas
        if item["expression"] is not None
    }
    names_by_name = {item["name"]: item for item in names}
    graph: dict[str, set[str]] = {}

    def add_cell_edges(node: str, references: list[dict]) -> None:
        for reference in references:
            sheet, row_min, row_max, col_min, col_max = _reference_bounds(reference)
            for formula_sheet, address in formulas_by_cell:
                if formula_sheet != sheet:
                    continue
                match = re.fullmatch(r"([A-Z]+)(\d+)", address)
                if not match:
                    continue
                col = 0
                for char in match.group(1):
                    col = col * 26 + ord(char) - 64
                row = int(match.group(2)) - 1
                if row_min <= row <= row_max and col_min <= col - 1 <= col_max:
                    graph[node].add(f"CELL:{sheet}!{address}")

    for formula in formulas:
        if formula["expression"] is None:
            continue
        node = f"CELL:{formula['sheet']}!{formula['cell']}"
        graph[node] = set()
        add_cell_edges(node, formula["cellReferences"])
        for name in formula["definedNames"]:
            if name in names_by_name:
                graph[node].add(f"NAME:{name}")

    for name in names:
        node = f"NAME:{name['name']}"
        graph[node] = set()
        if name["refersTo"]:
            references, _ = _references(name["refersTo"], "")
            add_cell_edges(node, references)
            for nested_name in _defined_name_uses(name["refersTo"], names):
                if nested_name in names_by_name:
                    graph[node].add(f"NAME:{nested_name}")

    state: dict[str, int] = {}
    stack = []
    cycles = []

    def visit(node: str) -> None:
        state[node] = 1
        stack.append(node)
        for target in sorted(graph.get(node, ())):
            if state.get(target, 0) == 0:
                visit(target)
            elif state[target] == 1:
                start = stack.index(target)
                cycles.append(stack[start:] + [target])
        stack.pop()
        state[node] = 2

    for node in sorted(graph):
        if state.get(node, 0) == 0:
            visit(node)
    return cycles


def extract(source: Path) -> tuple[dict, dict, str]:
    if not source.is_file():
        raise ExtractionError(f"XLS source file does not exist: {source}")

    try:
        book = xlrd.open_workbook(str(source), on_demand=False)
    except Exception as exc:
        raise ExtractionError(f"xlrd could not open {source}: {exc}") from exc
    if book.biff_version != 80:
        raise ExtractionError(
            f"expected an Excel 97-2003 BIFF8 workbook; found BIFF version "
            f"{book.biff_version}"
        )

    try:
        with olefile.OleFileIO(str(source)) as ole:
            workbook_stream = next(
                (parts[-1] for parts in ole.listdir() if parts[-1].casefold() == "workbook"),
                None,
            )
            if workbook_stream is None:
                raise ExtractionError("OLE container has no Workbook stream")
            stream = ole.openstream(workbook_stream).read()
    except ExtractionError:
        raise
    except Exception as exc:
        raise ExtractionError(f"could not read OLE Workbook stream: {exc}") from exc

    names = _name_records(book)
    try:
        records = _formula_records(book, stream)
    except (IndexError, KeyError, struct.error) as exc:
        raise ExtractionError(f"malformed BIFF8 worksheet records: {exc}") from exc
    formulas = []
    for record in records:
        sheet = book.sheet_names()[record["sheetIndex"]]
        cell = _cell_address(record["row"], record["column"])
        formula = {
            "sheet": sheet,
            "cell": cell,
            "expression": None,
            "cellReferences": [],
            "definedNames": [],
            "externalOrInterSheetReferences": [],
            "functions": [],
            "hasConditionalExpression": False,
            "historicalOrAmbiguousReferences": [],
            "recovery": {
                "status": "unresolved",
                "confidence": "none",
                "method": "direct_biff8",
            },
            "biff8TokenHex": record["tokens"].hex(),
        }
        try:
            raw_tokens = _scan_tokens(record["tokens"], book.biff_version)
            raw_name_tokens = [
                book.name_obj_list[index].name
                for index in raw_tokens["definedNameIndexes"]
                if 0 <= index < len(book.name_obj_list)
            ]
            expression, legacy_labels = _decode_formula(book, record)
            references, decoded_ref_count = _references(expression, sheet)
            used_names = _defined_name_uses(expression, names)
            decoded_name_tokens = _defined_name_occurrences(expression, names)
            defined_name_tokens_complete = (
                Counter(raw_name_tokens) == decoded_name_tokens
                and len(raw_name_tokens) == len(raw_tokens["definedNameIndexes"])
                and raw_tokens["externalNameCount"] == 0
            )
            functions = _formula_functions(expression)
            formula.update(
                {
                    "expression": expression,
                    "cellReferences": references,
                    "definedNames": used_names,
                    "externalOrInterSheetReferences": [
                        {
                            "kind": (
                                "external"
                                if "[" in item["sheet"] and "]" in item["sheet"]
                                else "inter_sheet"
                            ),
                            **item,
                        }
                        for item in references
                        if item["sheet"] != sheet
                    ],
                    "functions": functions,
                    "hasConditionalExpression": bool(
                        set(functions) & CONDITIONAL_FUNCTIONS
                    ),
                    "historicalOrAmbiguousReferences": legacy_labels,
                    "recovery": {
                        "status": (
                            "direct_biff8_with_legacy_label"
                            if legacy_labels
                            else "direct_biff8"
                        ),
                        "confidence": "high",
                        "method": "xlrd 2.0.2 BIFF8 token decompiler",
                        "referenceIntegrity": {
                            "rawCellReferenceTokenCount": raw_tokens[
                                "cellReferenceCount"
                            ],
                            "decodedCellReferenceOccurrences": decoded_ref_count,
                            "rawDefinedNameTokenCount": len(raw_name_tokens),
                            "decodedDefinedNameOccurrences": sum(
                                decoded_name_tokens.values()
                            ),
                            "definedNameTokensComplete": defined_name_tokens_complete,
                            "complete": (
                                raw_tokens["cellReferenceCount"] == decoded_ref_count
                                and defined_name_tokens_complete
                            ),
                        },
                    },
                }
            )
            if not formula["recovery"]["referenceIntegrity"]["complete"]:
                formula["recovery"]["confidence"] = "partial"
                formula["recovery"]["referenceIntegrity"]["reason"] = (
                    "Rendered cell or defined-name references differ from BIFF tokens."
                )
        except Exception as exc:
            formula["recovery"]["reason"] = f"{type(exc).__name__}: {exc}"
            try:
                formula["recovery"]["rawReferenceTokenCount"] = _scan_tokens(
                    record["tokens"], book.biff_version
                )["cellReferenceCount"]
            except Exception:
                formula["recovery"]["rawReferenceTokenCount"] = None
        formulas.append(formula)

    sheet_order = {name: index for index, name in enumerate(book.sheet_names())}
    formulas.sort(
        key=lambda item: (
            sheet_order[item["sheet"]],
            *_cell_sort_key(item["cell"]),
        )
    )
    cycles = _cycle_check(formulas, names)
    source_hash = hashlib.sha256(source.read_bytes()).hexdigest()
    integrity = {
        "formulaCount": len(formulas),
        "recoveredFormulaCount": sum(item["expression"] is not None for item in formulas),
        "unrecoveredFormulaCount": sum(item["expression"] is None for item in formulas),
        "referenceIntegrity": all(
            item.get("recovery", {}).get("referenceIntegrity", {}).get("complete", False)
            for item in formulas
            if item["expression"] is not None
        ),
        "definedNameCount": len(names),
        "unresolvedDefinedNameCount": sum(
            item["refersTo"] is None for item in names
        ),
        "conditionalFormulaCount": sum(
            item["hasConditionalExpression"] for item in formulas
        ),
        "cycleCheck": {
            "status": "cycles_found" if cycles else "no_cycles_found",
            "cycles": cycles,
            "scope": "decoded cell and defined-name dependencies",
            "unresolvedHistoricalLabelRangesExcluded": sum(
                len(item["historicalOrAmbiguousReferences"])
                for item in formulas
            ),
        },
    }

    formulas_document = {
        "schemaVersion": 1,
        "source": {
            "fileName": source.name,
            "format": "BIFF8",
            "sha256": source_hash,
            "formulaDecoder": f"xlrd {xlrd.__version__}",
            "oleReader": f"olefile {olefile.__version__}",
            "originalLexicalTextAvailable": False,
            "originalLexicalTextNote": (
                "BIFF8 stores formula tokens, not the original spacing or typed "
                "function-name casing. Expressions are rendered from those tokens."
            ),
        },
        "integrity": integrity,
        "formulas": formulas,
    }
    names_document = {
        "schemaVersion": 1,
        "source": formulas_document["source"],
        "definedNames": names,
    }
    report = _report(formulas_document)
    return formulas_document, names_document, report


def _report(formulas_document: dict) -> str:
    integrity = formulas_document["integrity"]
    labels = [
        formula
        for formula in formulas_document["formulas"]
        if formula["historicalOrAmbiguousReferences"]
    ]
    unresolved_names = integrity["unresolvedDefinedNameCount"]
    unresolved_formulas = integrity["unrecoveredFormulaCount"]
    cycles = integrity["cycleCheck"]["cycles"]
    lines = [
        "# Витяг формул із BIFF8",
        "",
        f"- Формул у BIFF8: **{integrity['formulaCount']}**; відновлено виразів: "
        f"**{integrity['recoveredFormulaCount']}**; не відновлено: "
        f"**{unresolved_formulas}**.",
        f"- Визначених імен: **{integrity['definedNameCount']}**; без "
        f"декодованого визначення: **{unresolved_names}**.",
        f"- Функційні умовні вирази: **{integrity['conditionalFormulaCount']}**.",
        "- Декодування: `xlrd 2.0.2` читає BIFF8 FORMULA-токени; `olefile 0.47` "
        "читає OLE-потік `Workbook`. Конвертація через Excel/LibreOffice не "
        "використовувалася.",
        "- Повторний запуск: `python extract_formulas.py <path-to.xls>`; точні "
        "версії залежностей вказані в `requirements.txt`.",
        "- Відносні 3D-діапазони декодовано з урахуванням адреси формульної "
        "клітинки. BIFF8 має RPN-токени, а не початковий текст із пробілами та "
        "регістром імен функцій; числові літерали виведено у нормалізованому вигляді.",
    ]
    if labels:
        details = ", ".join(
            f"`{item['cell']}` → "
            + ", ".join(
                f"`{ref['token']}` у `{ref['labelCell']}` "
                f"(мітка `{ref['labelText']}`)"
                for ref in item["historicalOrAmbiguousReferences"]
            )
            for item in labels
        )
        lines.append(
            "- Історичні посилання: "
            + details
            + ". Токен визначає клітинку мітки, але не розгортає діапазон; "
            "діапазон не вгадувався і виключений із перевірки циклів."
        )
    else:
        lines.append("- Історичних або неоднозначних посилань `PtgElfRwV` не виявлено.")
    lines.extend(
        [
            f"- Циклічні залежності: **{'виявлено' if cycles else 'не виявлено'}** "
            "у декодованих посиланнях клітинок та визначених імен.",
            f"- Перевірка відповідності посилань BIFF8-токенам: "
            f"**{'пройдена' if integrity['referenceIntegrity'] else 'є розбіжності'}**.",
            "",
            "Деталізація, включно з адресами й сирими hex-токенами, міститься у "
            "`formulas.json` та `defined-names.json`.",
            "",
        ]
    )
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Extract formulas and defined names directly from an Excel BIFF8 XLS."
    )
    parser.add_argument("xls", type=Path, help="path to the source .xls file (read-only)")
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path(__file__).resolve().parent,
        help="directory for formulas.json, defined-names.json, and extraction-report.md",
    )
    args = parser.parse_args(argv)

    try:
        formulas, names, report = extract(args.xls)
        args.output_dir.mkdir(parents=True, exist_ok=True)
        outputs = {
            "formulas.json": json.dumps(
                formulas, ensure_ascii=False, indent=2
            )
            + "\n",
            "defined-names.json": json.dumps(
                names, ensure_ascii=False, indent=2
            )
            + "\n",
            "extraction-report.md": report,
        }
        source_path = args.xls.resolve()
        for name, content in outputs.items():
            output_path = args.output_dir / name
            if output_path.resolve() == source_path or (
                output_path.exists() and output_path.samefile(source_path)
            ):
                raise ExtractionError(
                    f"refusing to overwrite the source XLS: {source_path}"
                )
            output_path.write_text(content, encoding="utf-8", newline="\n")
    except (ExtractionError, OSError, ValueError, xlrd.XLRDError) as exc:
        print(f"Extraction failed: {exc}", file=sys.stderr)
        return 1

    print(
        f"Extracted {formulas['integrity']['recoveredFormulaCount']}/"
        f"{formulas['integrity']['formulaCount']} formulas to {args.output_dir}"
    )
    if (
        formulas["integrity"]["unrecoveredFormulaCount"]
        or formulas["integrity"]["unresolvedDefinedNameCount"]
        or not formulas["integrity"]["referenceIntegrity"]
        or formulas["integrity"]["cycleCheck"]["cycles"]
    ):
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

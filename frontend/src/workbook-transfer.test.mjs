import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {workbookTransferMatches, workbookTransferRows, workbookTransferTargets, parseTransferRows, parseTransferValue} from './workbook-transfer.mjs';
import {parsePreparation} from './preparation.mjs';

// Numeric values mirror the Go golden test (internal/prepcalc) for the
// reference inputs d=5.45, q=0.0034, Vд=960, ns=0.83, f1=1.05, af=1.1.
const transfer = {w0: 1.9853002318394384e-06, pm: 440.01195661147534, ls: 0.4127960653564516};
const inputs = {d: 0.00545, q: 0.0034, vd: 960, lc: 0.414, ns: 0.83, f1: 1.05, af: 1.1};
const snapshot = {...inputs};
const current = {...inputs, inputFormats: {}};

test('transfer targets map the F72:F74 block onto working-screen fields', () => {
    assert.deepEqual(workbookTransferTargets.map(target => target.key), ['w0', 'pmax', 'lc']);
    assert.deepEqual(workbookTransferTargets.map(target => target.technicalId), ['W0', 'PMAX', 'LS']);
    // pm arrives in MPa while the pmax field stores pascals.
    const [w0, pmax, lc] = workbookTransferTargets.map(target => target.stored(transfer));
    assert.equal(w0, transfer.w0);
    assert.equal(pmax, transfer.pm * 1e6);
    assert.equal(lc, transfer.ls);
});

test('transfer is offered only for results matching the current seven inputs', () => {
    assert.equal(workbookTransferMatches(snapshot, current), true);
    assert.equal(workbookTransferMatches(snapshot, {...inputs, d: 0.006}), false);
    assert.equal(workbookTransferMatches(snapshot, {...inputs, q: null}), false);
    assert.equal(workbookTransferMatches(null, current), false);
    assert.equal(workbookTransferMatches(snapshot, null), false);
    // Input formats are cosmetic: retyping the same numbers keeps the match.
    assert.equal(workbookTransferMatches(snapshot, {...inputs, inputFormats: {d: '5,45'}}), true);
});

test('confirmation rows carry current and new values in editor display format', () => {
    const rows = workbookTransferRows(transfer, current);
    assert.equal(rows.length, 3);
    const [w0, pmax, lc] = rows;
    assert.equal(w0.newStored, transfer.w0);
    assert.equal(w0.newText, String(transfer.w0.toExponential()));
    assert.equal(pmax.newStored, transfer.pm * 1e6);
    assert.equal(pmax.newText, String(transfer.pm));
    assert.equal(lc.newText, String(transfer.ls));
    // Display text must round-trip through the field parser.
    for (const row of rows) {
        assert.equal(parsePreparation(row.newText, row.key), row.newStored, row.key);
    }
});

test('rows keep the current stored value and show a dash when the field was empty', () => {
    const previous = {w0: 2e-06, pmax: 3e8, lc: 0.4};
    const rows = workbookTransferRows(transfer, previous);
    assert.deepEqual(rows.map(row => [row.currentStored, row.newStored]), [
        [2e-06, transfer.w0],
        [3e8, transfer.pm * 1e6],
        [0.4, transfer.ls],
    ]);
    const empty = workbookTransferRows(transfer, {w0: null, pmax: null, lc: null});
    assert.deepEqual(empty.map(row => row.currentText), ['—', '—', '—']);
});

test('rows are refused when a transfer value is missing or not finite', () => {
    assert.deepEqual(workbookTransferRows(null, current), []);
    assert.deepEqual(workbookTransferRows(undefined, current), []);
    assert.deepEqual(workbookTransferRows({w0: 1e-06, pm: Number.NaN, ls: 1}, current), []);
    assert.deepEqual(workbookTransferRows({w0: 1e-06}, current), []);
});

test('edited dialog values parse back into stored units like the editor fields', () => {
    const rows = workbookTransferRows(transfer, current);
    // Untouched display text round-trips to the original stored values.
    assert.deepEqual(
        parseTransferRows(rows, rows.map(row => row.newText)),
        rows.map(row => ({key: row.key, stored: row.newStored})),
    );
    // Edited values: comma decimals, plain MPa for pmax (stored as pascals).
    assert.deepEqual(parseTransferRows(rows, ['2e-6', '450', '0,42']), [
        {key: 'w0', stored: 2e-06},
        {key: 'pmax', stored: 450e6},
        {key: 'lc', stored: 0.42},
    ]);
    assert.equal(parseTransferValue(' 1,05 ', 'f1'), 1.05e6);
    assert.equal(parseTransferValue('5,45', 'd'), 0.00545);
});

test('invalid edited values throw with the row index for highlighting', () => {
    const rows = workbookTransferRows(transfer, current);
    assert.throws(() => parseTransferRows(rows, ['abc', '450', '0.42']), error => {
        assert.equal(error.index, 0);
        assert.match(error.message, /Введіть число/);
        return true;
    });
    assert.throws(() => parseTransferRows(rows, ['2e-6', '', '0.42']), error => {
        assert.equal(error.index, 1);
        assert.match(error.message, /Введіть число/);
        return true;
    });
    assert.throws(() => parseTransferRows(rows, ['2e-6', '450']), /Невідповідна кількість/);
    assert.throws(() => parseTransferRows(rows, ['2e-6', '450', '1e999']), error => {
        assert.equal(error.index, 2);
        return true;
    });
});

test('main.js wires the transfer button, dialog and safety rules', () => {
    const main = readFileSync(new URL('./main.js', import.meta.url), 'utf8');
    const results = main.match(/<aside id="preparation-results"[\s\S]*?<\/aside>/)?.[0] ?? '';
    assert.ok(results);
    // The button lives in the first result card, rendered by renderWorkbookResults.
    assert.match(main, /if \(index === 0 && workbook\) \{[\s\S]*?id = 'workbook-transfer'/);
    assert.doesNotMatch(results, /workbook-transfer/);
    // The listener is attached at creation time: the button is recreated on
    // every render and does not exist at module scope.
    assert.match(main, /transferButton\.addEventListener\('click', \(\) => operation\(runWorkbookTransfer\)\)/);
    assert.doesNotMatch(main, /\$\('workbook-transfer'\)\.addEventListener/);
    // The confirmed action changes only the three fields and never calculates.
    const handler = main.slice(
        main.indexOf('async function runWorkbookTransfer()'),
        main.indexOf('function toggleParameters('),
    );
    assert.ok(handler.length > 0);
    // Matching gates on the seven entered inputs; the dialog's "current"
    // column comes from the working-screen preparation fields being replaced.
    assert.match(handler, /workbookTransferMatches\(snapshot, entered\)/);
    assert.match(handler, /const current = state\?\.active\?\.parameters\?\.preparation \?\? null/);
    assert.match(handler, /workbookTransferRows\(run\?\.summary\?\.workbook\?\.transfer, current\)/);
    // The "new value" column is an editable input prefilled with display text.
    assert.match(handler, /input\.id = `transfer-new-\$\{row\.key\}`/);
    assert.match(handler, /input\.value = row\.newText/);
    assert.match(handler, /error\.id = `transfer-new-\$\{row\.key\}-error`/);
    assert.match(handler, /\$\('p-' \+ item\.key\)/);
    assert.match(handler, /displayPreparation\(item\.stored, item\.key\)/);
    assert.match(handler, /setPreparationScreen\(false\)/);
    assert.match(handler, /await save\(\)/);
    assert.match(handler, /Основний розрахунок не запущено/);
    assert.doesNotMatch(handler, /Calculate\(|CalculateWorkbook\(|runCalculate/);
    // Edited values come from the validated dialog inputs, not the raw rows.
    assert.match(handler, /pendingTransferValues \?\? rows\.map\(row => \(\{key: row\.key, stored: row\.newStored\}\)\)/);

    // The dialog form validates on submit and only closes on success.
    assert.match(main, /\$\('transfer-dialog'\)\.querySelector\('form'\)\.addEventListener\('submit', event => \{/);
    assert.match(main, /event\.preventDefault\(\)/);
    assert.match(main, /parseTransferRows\(rows, texts\)/);
    assert.match(main, /\$\('transfer-dialog'\)\.close\('transfer'\)/);
    assert.match(main, /input\.setAttribute\('aria-invalid', 'true'\)/);
    assert.match(main, /<dialog id="transfer-dialog">/);
    assert.match(main, /Значення в колонці «Нове значення» можна редагувати\./);
    // Cancellation leaves everything untouched.
    assert.match(handler, /Передачу скасовано\. Значення не змінено\./);
    // Availability follows the live seven inputs and the shown run snapshot.
    assert.match(main, /function updateTransferAvailability\(\) \{[\s\S]*?workbookTransferMatches\(snapshot, liveWorkbookValues\(\)\)/);
    assert.match(main, /markDirty\(field\);\s*\n\s*\}\s*\n\s*setPreparationScreen\(false\)/s);
});

// Regression guard for the wiring bug: a module-scope listener on a
// dynamically created button crashed the whole module at load time, so
// navigation, autosave and the window close handler never registered.
// Dynamically built controls are only the p-*/workbook-* editor fields
// created by the construction loops; everything else must be static.
test("every single-quoted $('id') lookup in main.js exists in the static template", () => {
    const main = readFileSync(new URL('./main.js', import.meta.url), 'utf8');
    const template = main.match(/innerHTML = `([\s\S]*?)`;/)?.[1] ?? '';
    assert.ok(template.length > 1000);
    const dynamicPrefixes = ['p-', 'workbook-'];
    const ids = [...main.matchAll(/\$\('([A-Za-z][\w-]*)'\)/g)].map(match => match[1]);
    assert.ok(ids.length > 30);
    for (const id of new Set(ids)) {
        if (dynamicPrefixes.some(prefix => id.startsWith(prefix))) continue;
        assert.match(template, new RegExp(`id="${id}"`), id);
    }
});

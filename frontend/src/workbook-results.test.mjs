import {test} from 'node:test';
import assert from 'node:assert/strict';
import {formatWorkbookValue, workbookResultSections} from './workbook-results.mjs';

// Numeric values mirror the Go golden test (internal/prepcalc) for the
// reference inputs d=5.45, q=0.0034, Vд=960, ns=0.83, f1=1.05, af=1.1.
const goldenResult = {
    ed: 1566.7199999999998,
    edMJ: 0.0015667199999999997,
    cq: 21003.390657660948,
    ceMJ: 9678.362415050164,
    s: 2.4653074999999998e-05,
    tableIndex: 10,
    pmKr: 381.9590603762541,
    delta: 740,
    hi: 1.9381871139565212,
    kisp: 1.0664327516989969,
    lstvD: 70.35672483010033,
    omega: 0.0014691221715611845,
    w0: 1.9853002318394384e-06,
    l0: 0.08052951738634789,
    lcCalc: 0.38344415032404683,
    ls: 0.37526915032404684,
    lkam: 0.04154888700191533,
    ld: 0.3337202633221315,
    lambdaD: 4.1440738024180295,
    pmKrU: 401.05701339506686,
    lsu: 0.4127960653564516,
    lcU: 0.4127960653564516,
    ldU: 0.37124717835453624,
    lambdaDU: 4.610075788402446,
    lambda1: 0.9137170404053115,
    fi: 1.2316041139796776,
    pm: 440.01195661147534,
    ettaD: 0.47914149328454253,
    transfer: {w0: 1.9853002318394384e-06, pm: 440.01195661147534, ls: 0.4127960653564516},
    warnings: [],
};

test('formatWorkbookValue keeps six significant digits and marks gaps', () => {
    assert.equal(formatWorkbookValue(440.01195661147534), '440.012');
    assert.equal(formatWorkbookValue(1566.7199999999998), '1566.72');
    assert.equal(formatWorkbookValue(1.9853002318394384e-06), '0.00000198530');
    assert.equal(formatWorkbookValue(0), '0.00000');
    assert.equal(formatWorkbookValue(Number.NaN), '—');
    assert.equal(formatWorkbookValue(Number.POSITIVE_INFINITY), '—');
    assert.equal(formatWorkbookValue(undefined), '—');
    assert.equal(formatWorkbookValue(null), '—');
    assert.equal(formatWorkbookValue('440'), '—');
});

test('workbookResultSections starts with the transfer block and keeps cell references', () => {
    const sections = workbookResultSections(goldenResult);
    assert.equal(sections.length, 6);
    assert.deepEqual(sections[0].title, 'Для введення в програму');
    assert.deepEqual(sections[0].rows.map(([term, value]) => [term, value]), [
        ['W0 (F72), м³', '0.00000198530'],
        ['pm (F73), МПа', '440.012'],
        ['Lс.у (F74), м', '0.412796'],
    ]);
    assert.match(sections[2].title, /Таблиця Слухоцького \(рядок 10\)/);
    assert.deepEqual(sections[2].rows[0].slice(0, 2), ['pm.кр (F25), МПа', '381.959']);
    assert.deepEqual(sections[5].rows.at(-1).slice(0, 2), ['ηд (F67)', '0.479141']);
    for (const section of sections) {
        assert.ok(section.title);
        for (const [term, value] of section.rows) {
            assert.ok(term);
            assert.match(value, /^(—|\d)/);
        }
    }
});

test('every result row carries hover help without source-file references', () => {
    const sections = workbookResultSections(goldenResult);
    for (const [index, section] of sections.entries()) {
        for (const [term, , help] of section.rows) {
            assert.ok(help && help.title && help.code && help.description, term);
            assert.match(help.code, /^[A-Z]+\d+$/, term);
            assert.doesNotMatch(help.title + help.description, /—/, term);
            if (index !== 0) assert.doesNotMatch(help.description, /\.xls|\.doc|Диплом\.xls|Diplom/i, term);
        }
    }
    // The transfer block is allowed to reference the ballistics program.
    assert.match(sections[0].rows[0][2].description, /програм/);
});

test('workbookResultSections tolerates missing or partial results', () => {
    assert.deepEqual(workbookResultSections(null), []);
    assert.deepEqual(workbookResultSections(undefined), []);
    const partial = workbookResultSections({transfer: {w0: 1.5e-06}});
    assert.equal(partial.length, 6);
    assert.deepEqual(partial[0].rows[0].slice(0, 2), ['W0 (F72), м³', '0.00000150000']);
    assert.equal(partial[0].rows[1][1], '—');
    assert.match(partial[2].title, /рядок —/);
});

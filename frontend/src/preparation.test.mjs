import {test} from 'node:test';
import assert from 'node:assert/strict';
import {parsePreparation, parseWorkbookPreparation, parseKTSelection, ktSelectValue, displayPreparation, preparationFields, workbookPreparationFields, preparationResultPanel, formaOptions, ktOptions, KT_CUSTOM_VALUE} from './preparation.mjs';

test('historical workbook editor exposes exactly seven inputs and confirmed units', () => {
    assert.deepEqual(workbookPreparationFields.map(({key}) => key), ['d', 'q', 'vd', 'lc', 'ns', 'f1', 'af']);
    assert.deepEqual(workbookPreparationFields.map(({unit}) => unit), ['мм', 'кг', 'м/с', 'м', null, null, null]);
    assert.match(workbookPreparationFields.find(({key}) => key === 'lc').help, /D10.*формули Excel його не читають/);
    for (const field of workbookPreparationFields) {
        assert.ok(field.label);
        assert.ok(field.help);
    }
});

test('preparation result panel has a title only, without instruction text', () => {
    assert.deepEqual(preparationResultPanel, {title: 'Результати підготовки'});
    assert.deepEqual(Object.keys(preparationResultPanel), ['title']);
});

test('workbook editor parser accepts finite locale numbers without inferred units', () => {
    assert.equal(parseWorkbookPreparation(''), null);
    assert.equal(parseWorkbookPreparation('  '), null);
    assert.equal(parseWorkbookPreparation('1,25e2'), 125);
    assert.equal(parseWorkbookPreparation('-0.5'), -0.5);
    assert.equal(parseWorkbookPreparation('0e-999'), 0);
    for (const text of ['-', '1,', '1e-', '1 000', '1mm', 'NaN', 'Infinity', '1e999', '1e-999']) {
        assert.throws(() => parseWorkbookPreparation(text), text);
    }
});

test('nullable preparation inputs, comma, dot, exponent and SI conversion', () => {
    assert.equal(preparationFields.length, 22);
    for (const {key} of preparationFields) {
        assert.equal(parsePreparation('  ', key), null);
        if (key !== 'forma') assert.equal(parsePreparation('0', key), 0);
        for (const text of ['0,5', '.5', '5E-1', '+0.5']) {
            if (key === 'forma') continue;
            const expected = key === 'd' ? .0005
                : key === 'svod' ? .00025
                : ['pmax', 'pfor', 'pvos'].includes(key) ? 500000
                : key === 'f1' ? 500000
                : .5;
            assert.equal(parsePreparation(text, key), expected);
        }
        if (key !== 'forma') {
            assert.equal(parsePreparation('-2.5e2', key),
                key === 'd' ? -.25
                : key === 'svod' ? -.125
                : ['pmax', 'pfor', 'pvos', 'f1'].includes(key) ? -250e6
                : -250);
        }
    }
    assert.equal(displayPreparation(.0123, 'd'), '12.3');
    assert.equal(displayPreparation(null, 'd'), '');
    assert.equal(parsePreparation('5e-324', 'q'), Number.MIN_VALUE);
});

test('scientific notation remains scientific after display round-trip', () => {
    for (const {key} of preparationFields) {
        if (key === 'forma') continue;
        const stored = parsePreparation('6.5e-10', key);
        assert.equal(displayPreparation(stored, key), '6.5e-10', key);
    }
    assert.equal(displayPreparation(parsePreparation('17.1e-3', 'w0'), 'w0'), '1.71e-2');
});

test('preserves each valid field input format and falls back for mismatches', () => {
    for (const {key} of preparationFields) {
        const raw = key === 'forma' ? '6' : '17.10E-3';
        const stored = parsePreparation(raw, key);
        assert.equal(displayPreparation(stored, key, raw), raw, key);
    }
    assert.equal(displayPreparation(2, 'q', '3.0'), '2');
});

test('FORMA select names follow the BAS grain-shape descriptions', () => {
    assert.equal(formaOptions.length, 7);
    assert.deepEqual(formaOptions[5], {value: 6, label: 'Трубка'});
    assert.deepEqual(formaOptions[6], {value: 7, label: '7-ми канальн.'});
});

test('all 22 controls have plain labels and separate help and unit metadata', () => {
    assert.deepEqual(preparationFields.map(field => field.key), [
        'pmax', 'vd', 'd', 'q', 'ns', 'hi', 'af', 'pfor', 'kt', 'tn', 'forma',
        'omega', 'w0', 'ro', 'tg15', 'ku1', 'f1', 'u15', 'svod', 'pvos', 'tvos', 'lc',
    ]);
    for (const field of preparationFields) {
        assert.ok(field.technicalId, field.key);
        assert.ok(field.label, field.key);
        assert.ok(!field.label.includes(`(${field.technicalId})`), field.key);
        if (field.unit) {
            assert.ok(!field.label.endsWith(`, ${field.unit}`), field.key);
            assert.equal(field.control, 'text', field.key);
        }
        if (field.description !== undefined) {
            assert.ok(field.description.trim(), field.key);
            assert.doesNotMatch(field.description, /\bBAS\b/);
        }
        assert.ok(field.source, field.key);
        assert.ok(['text', 'select'].includes(field.control), field.key);
    }
    const byKey = Object.fromEntries(preparationFields.map(field => [field.key, field]));
    assert.equal(byKey.pmax.label, 'Максимальний тиск у стволі');
    assert.equal(byKey.d.unit, 'мм');
    assert.equal(byKey.d.description, undefined);
    assert.equal(byKey.q.description, undefined);
    assert.ok(byKey.kt.description);
    assert.equal(byKey.kt.unit, null);
    assert.equal(byKey.forma.unit, null);
    assert.deepEqual(preparationFields.filter(field => field.control === 'select').map(field => field.key), ['kt', 'forma']);
    assert.deepEqual(ktOptions, [
        {value: 0.0025, label: 'Нітрогліцериновий порох'},
        {value: 0.0018, label: 'Піроксиліновий порох'},
    ]);
});

test('KT keeps numeric presets and supports arbitrary legacy numeric values', () => {
    for (const option of ktOptions) {
        assert.equal(ktSelectValue(option.value), String(option.value));
        assert.equal(parseKTSelection(String(option.value), ''), option.value);
    }
    assert.equal(ktSelectValue(null), '');
    assert.equal(ktSelectValue(0.0012345), KT_CUSTOM_VALUE);
    assert.equal(parseKTSelection(KT_CUSTOM_VALUE, '1.23450e-3'), 0.0012345);
    assert.throws(() => parseKTSelection('unknown', '0.0018'), /Виберіть варіант KT/);
    assert.throws(() => parseKTSelection(KT_CUSTOM_VALUE, '1e'), /Завершіть число/);
    const legacyValue = 0.00123456789;
    const raw = '1.234567890E-3';
    assert.equal(parsePreparation(raw, 'kt'), legacyValue);
    assert.equal(displayPreparation(legacyValue, 'kt', raw), raw);
    assert.equal(displayPreparation(legacyValue, 'kt'), String(legacyValue));
});

test('reject partial, malformed, overflow and underflow without prefix parsing', () => {
    for (const text of ['-', '+', '1,', '1.', '1e', '1e-', '1e+', '1.2,3', '1 000', '1mm', 'NaN', 'Infinity', '0x10', '1e999', '1e-999']) {
        assert.throws(() => parsePreparation(text, 'q'), text);
    }
    assert.throws(() => parsePreparation('5e-324', 'd'));
    assert.equal(parsePreparation('0e-999', 'd'), 0);
    assert.match(displayPreparation(Number.MAX_VALUE, 'd'), /e311$/);
});

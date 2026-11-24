import {test} from 'node:test';
import assert from 'node:assert/strict';
import {parsePreparation, displayPreparation, preparationFields, formaOptions} from './preparation.mjs';

test('nullable preparation inputs, comma, dot, exponent and SI conversion', () => {
    assert.equal(preparationFields.length, 22);
    for (const [key] of preparationFields) {
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
    for (const [key] of preparationFields) {
        if (key === 'forma') continue;
        const stored = parsePreparation('6.5e-10', key);
        assert.equal(displayPreparation(stored, key), '6.5e-10', key);
    }
    assert.equal(displayPreparation(parsePreparation('17.1e-3', 'w0'), 'w0'), '1.71e-2');
});

test('preserves each valid field input format and falls back for mismatches', () => {
    for (const [key] of preparationFields) {
        const raw = key === 'forma' ? '6' : '17.10E-3';
        const stored = parsePreparation(raw, key);
        assert.equal(displayPreparation(stored, key, raw), raw, key);
    }
    assert.equal(displayPreparation(2, 'q', '3.0'), '2');
});

test('FORMA select names follow the BAS grain-shape descriptions', () => {
    assert.equal(formaOptions.length, 7);
    assert.deepEqual(formaOptions[5], [6, 'Трубка']);
    assert.deepEqual(formaOptions[6], [7, '7-ми канальн.']);
});

test('reject partial, malformed, overflow and underflow without prefix parsing', () => {
    for (const text of ['-', '+', '1,', '1.', '1e', '1e-', '1e+', '1.2,3', '1 000', '1mm', 'NaN', 'Infinity', '0x10', '1e999', '1e-999']) {
        assert.throws(() => parsePreparation(text, 'q'), text);
    }
    assert.throws(() => parsePreparation('5e-324', 'd'));
    assert.equal(parsePreparation('0e-999', 'd'), 0);
    assert.match(displayPreparation(Number.MAX_VALUE, 'd'), /e311$/);
});

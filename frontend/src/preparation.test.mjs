import {test} from 'node:test';
import assert from 'node:assert/strict';
import {parsePreparation, displayPreparation, preparationFields} from './preparation.mjs';

test('seven nullable inputs, comma, dot, exponent and SI conversion', () => {
    assert.equal(preparationFields.length, 7);
    for (const [key] of preparationFields) {
        assert.equal(parsePreparation('  ', key), null);
        assert.equal(parsePreparation('0', key), 0);
        for (const text of ['0,5', '.5', '5E-1', '+0.5']) {
            assert.equal(parsePreparation(text, key), key === 'd' ? .0005 : .5);
        }
        assert.equal(parsePreparation('-2.5e2', key), key === 'd' ? -.25 : -250);
    }
    assert.equal(displayPreparation(.0123, 'd'), '12.3');
    assert.equal(displayPreparation(null, 'd'), '');
    assert.equal(parsePreparation('5e-324', 'q'), Number.MIN_VALUE);
});

test('reject partial, malformed, overflow and underflow without prefix parsing', () => {
    for (const text of ['-', '+', '1,', '1.', '1e', '1e-', '1e+', '1.2,3', '1 000', '1mm', 'NaN', 'Infinity', '0x10', '1e999', '1e-999']) {
        assert.throws(() => parsePreparation(text, 'q'), text);
    }
    assert.throws(() => parsePreparation('5e-324', 'd'));
    assert.equal(parsePreparation('0e-999', 'd'), 0);
    assert.match(displayPreparation(Number.MAX_VALUE, 'd'), /e311$/);
});

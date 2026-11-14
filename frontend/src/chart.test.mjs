import assert from 'node:assert/strict';
import test from 'node:test';
import {readFile} from 'node:fs/promises';

// Import the browser module without adding a package-wide module convention.
const source = await readFile(new URL('./chart.js', import.meta.url), 'utf8');
const {normalizationDivisor, seriesPreference, verticalLimit} = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`);

test('automatic normalization handles zero, negative and constant series', () => {
    for (const [values, expected] of [[[0,0],1],[[-3,-3],3],[[2,2],2],[[-7,2],7]]) {
        assert.equal(normalizationDivisor({values}), expected);
    }
    assert.equal(verticalLimit, 1.25);
});
test('reference normalization never follows the actual maximum', () => {
    const series = {reference:2, values:[-100,0,100]};
    const before = structuredClone(series);
    assert.equal(normalizationDivisor(series),2);
    assert.equal(series.values[2]/normalizationDivisor(series),50);
    assert.deepEqual(series,before);
    assert.equal(normalizationDivisor({values:[2],reference:null}),2);
    for (const reference of [0,-2,NaN,Infinity,-Infinity]) assert.throws(() => normalizationDivisor({values:[1],reference}));
});
test('preferences follow IDs and preserve all-hidden legacy settings', () => {
    const settings = {visible:[false,true],scales:[2,3]};
    const a = {id:'a'}, b = {id:'b'};
    assert.deepEqual(seriesPreference(settings,a,0),{visible:false,scale:2});
    assert.deepEqual(seriesPreference(settings,b,1),{visible:true,scale:3});
    assert.equal(seriesPreference(settings,b,0).scale,3);
    assert.equal(seriesPreference(settings,a,1).visible,false);
    assert.deepEqual(seriesPreference(JSON.parse(JSON.stringify(settings)),a,1),{visible:false,scale:2});
});

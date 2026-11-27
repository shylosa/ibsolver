import assert from 'node:assert/strict';
import test from 'node:test';
import {readFile} from 'node:fs/promises';

// Import the browser module without adding a package-wide module convention.
const source = await readFile(new URL('./chart.js', import.meta.url), 'utf8');
const {normalizationDivisor, seriesPreference, verticalLimit, formatAxisValue, percentTickStep, formatXAxisTick, valueRange, nearestIndex, clampRange, TimeSeriesChart} = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`);

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

test('Excel-style axis labels use series identities', () => {
    assert.equal(formatAxisValue(300000000), '3,00E+08');
    assert.equal(formatAxisValue(0), '0,00');
    assert.equal(formatAxisValue(2.5), '2,50');
});

test('axis range is the normalized percent of the series divisor', () => {
    assert.deepEqual(valueRange({values:[0,100,250]}, 250, 1), [0,105]);
    const travelRange = valueRange({values:[0,.01,.016]}, .016, 1);
    assert.equal(travelRange[0], 0);
    assert(Math.abs(travelRange[1] - 105) < 1e-9);
    assert.deepEqual(valueRange({values:[100]}, 100, 2), [0,210]);
    const negative = valueRange({values:[-50,50]}, 100, 1);
    assert(Math.abs(negative[0] + 55) < 1e-9);
    assert(Math.abs(negative[1] - 55) < 1e-9);
});

test('chart draws one percent axis without secondary or unit labels', () => {
    // Right-axis tick labels.
    assert.doesNotMatch(source, /left \+ pw \+ 10/);
    // Rotated right-side unit label and rotated left-side unit label.
    assert.doesNotMatch(source, /translate\(w - 7/);
    // The only rotated left label is the axis name carrying the percent sign.
    assert.match(source, /c\.translate\(13, top \+ ph \/ 2\)[\s\S]*?fillText\('% від max'/);
    // Tick labels carry no percent sign; only the axis name does.
    assert.doesNotMatch(source, /\}%`/);
    // No chart title and no standing explanation line above the cursor.
    assert.doesNotMatch(source, /chartTitle/);
    assert.doesNotMatch(source, /Кожна крива/);
});

test('y ticks step over multiples of 25, x labels use one decimal digit', () => {
    assert.equal(percentTickStep(105), 25);
    assert.equal(percentTickStep(8 * 25), 25);
    assert.equal(percentTickStep(1000), 200);
    assert.equal(percentTickStep(1), 25);
    assert.equal(formatXAxisTick(0), '0,0');
    assert.equal(formatXAxisTick(4), '4,0');
    assert.equal(formatXAxisTick(15.2), '15,2');
    assert.equal(formatXAxisTick(100), '100,0');
    assert.equal(formatXAxisTick(Number.NaN), '—');
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

test('cursor zoom, pan limits and reset range on nonuniform times', () => {
    const times = [0, .1, .9, 2, 10];
    assert.equal(nearestIndex(times, 1.25 + .25 * 5), 3);
    assert.equal(nearestIndex(times, -.1), 0);
    assert.equal(nearestIndex(times, 11), 4);
    assert.deepEqual(clampRange(times, -9, 3), [0,3]);
    assert.deepEqual(clampRange(times, 9, 3), [7,10]);
    const chart = {data:{time:times}, index:2, requestDraw(){}};
    TimeSeriesChart.prototype.reset.call(chart);
    assert.deepEqual(chart.range,[0,10]); assert.equal(chart.index,null);
    const target = {canvas:{getBoundingClientRect:()=>({left:0,top:0,width:800,height:400})}};
    for (const [clientX,clientY] of [[0,50],[100,0],[799,50],[100,399]]) {
        assert.equal(TimeSeriesChart.prototype.fraction.call(target,{clientX,clientY}),null);
    }
    assert.equal(TimeSeriesChart.prototype.fraction.call(target,{clientX:423,clientY:100}),.5);
});

test('wheel zoom listener is not registered on the chart canvas', t => {
    globalThis.ResizeObserver = class { observe() {} };
    t.after(() => { delete globalThis.ResizeObserver; });
    const handlers = {};
    const canvas = {
        getContext: () => ({}),
        getBoundingClientRect: () => ({left:0, top:0, width:800, height:400}),
        addEventListener: (name, callback) => { handlers[name] = callback; },
    };
    new TimeSeriesChart(canvas, {}, {}, () => {});
    assert.equal(handlers.wheel, undefined);
});

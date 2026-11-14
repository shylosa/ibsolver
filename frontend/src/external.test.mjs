import test from 'node:test';
import assert from 'node:assert/strict';
import {externalMetadata} from './external.mjs';

test('external metadata preserves source labels, unknown units and actual X bounds', () => {
    const r = {id: 0, configurationId: 7, source: '<source>.csv', formatVersion: 'csv-v1', checksum: 'original-hash', chart: {time: [-2, 3], series: [{name: 'Unspecified', unit: ''}, {name: 'Measured', unit: 'custom'}]}};
    const before = JSON.stringify(r);
    const rows = new Map(externalMetadata(r));
    assert.equal(rows.get('Джерело'), '<source>.csv');
    assert.equal(rows.get('Серії'), 'Unspecified []; Measured [custom]');
    assert.equal(rows.get('Вісь X'), '-2 … 3 с (з CSV)');
    assert.equal(rows.get('SHA-256 файла'), 'original-hash');
    assert.equal(rows.get('Імпорт'), 'Ще не збережено');
    assert.equal(JSON.stringify(r), before);
});

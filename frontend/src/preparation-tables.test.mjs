import {test} from 'node:test';
import assert from 'node:assert/strict';
import {preparationTables} from './preparation-tables.mjs';

test('historical preparation tables preserve workbook structure and empty cells', () => {
    assert.equal(preparationTables.length, 3);
    assert.deepEqual(preparationTables.map(table => table.id), [
        'slukhotsky-table-1',
        'slukhotsky-table-2',
        'slukhotsky-table-3',
    ]);
    assert.deepEqual(preparationTables.map(table => [table.title, table.columns.length, table.rows.length]), [
        ['Таблица 1 (Слухоцкого)', 8, 12],
        ['Таблица 2', 5, 11],
        ['Таблица 3', 5, 6],
    ]);
    for (const table of preparationTables) {
        const expectedCells = table.columns.length - (table.groupColumn ? 1 : 0);
        assert.ok(table.columns.every(column => column.label.trim()));
        assert.ok(table.rows.every(row => row.cells.length === expectedCells));
        assert.ok(table.rows.every(row => row.cells.every(cell => typeof cell === 'string')));
    }

    const [table1, table2, table3] = preparationTables;
    assert.equal(table1.columns[2].label, ' Delta,  ');
    assert.equal(table1.columns[2].unit, 'кг/м3  ');
    assert.equal(table1.columns[1].unit, '  МПа');
    assert.equal(table3.columns[0].label, ' Марка пороха');
    assert.deepEqual(table1.rows.filter(row => row.group).map(row => [row.group.label, row.group.rowSpan]), [
        ['Гаубицы', 3],
        ['Пушки средней мощности', 4],
        ['Пушки высокой мощности', 5],
    ]);
    assert.equal(table1.rows[0].cells[0], '500');
    assert.equal(table1.rows[11].cells[6], '1200');
    assert.equal(table2.rows[3].cells[3], '>=1,1');
    assert.equal(table2.rows[8].cells[2], '');
    assert.equal(table2.rows[9].cells[4], '');
    assert.equal(table2.rows[8].cells[3], '-');
    assert.deepEqual(table2.rows[8].mergedCells, [{column: 1, colSpan: 2}]);
    assert.equal(table3.rows[1].cells[1], '2800-2900');
    assert.equal(table3.rows[2].cells[4], '0,65 - 0,75');
});

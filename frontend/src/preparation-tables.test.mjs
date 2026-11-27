import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {preparationTables} from './preparation-tables.mjs';
import {renderPreparationTable} from './preparation-table-view.mjs';

class TestElement {
    constructor(tagName, fragment = false) {
        this.tagName = tagName;
        this.fragment = fragment;
        this.children = [];
        this.dataset = {};
        this.attributes = {};
    }

    append(...nodes) {
        for (const node of nodes) this.children.push(...(node.fragment ? node.children : [node]));
    }

    replaceChildren(...nodes) {
        this.children = [];
        this.append(...nodes);
    }

    setAttribute(name, value) {
        this.attributes[name] = value;
    }
}

function findElements(root, tagName) {
    return root.children.flatMap(child => [
        ...(child.tagName === tagName ? [child] : []),
        ...findElements(child, tagName),
    ]);
}

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
        assert.ok(table.columns.every(column => column.headerHidden || column.label.trim()));
        assert.ok(table.rows.every(row => row.cells.length === expectedCells));
        assert.ok(table.rows.every(row => row.cells.every(cell => typeof cell === 'string')));
    }

    const [table1, table2, table3] = preparationTables;
    assert.equal(table1.columns[6].label, 'V0 при cq=15000');
    assert.equal(table1.columns[6].headerColSpan, 2);
    assert.equal(table1.columns[7].headerHidden, true);
    assert.equal(table1.columns[2].label, ' Delta,  ');
    assert.equal(table1.columns[2].unit, 'кг/м3  ');
    assert.equal(table1.columns[1].unit, 'МПа');
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

test('table 1 headers pair each column label with its unit in one line', () => {
    const previousDocument = globalThis.document;
    globalThis.document = {
        createDocumentFragment: () => new TestElement('#fragment', true),
        createElement: tagName => new TestElement(tagName),
    };
    const container = new TestElement('div');
    try {
        renderPreparationTable(container, preparationTables[0]);
    } finally {
        if (previousDocument === undefined) delete globalThis.document;
        else globalThis.document = previousDocument;
    }

    const table = findElements(container, 'table')[0];
    const headerRows = findElements(table, 'thead')[0].children;
    const cells = headerRows[0].children;
    assert.equal(headerRows.length, 1);
    assert.equal(cells.length, preparationTables[0].columns.length - 1);
    assert.equal(cells[0].children[0].textContent, 'Ce, ');
    assert.equal(cells[0].children[1].className, 'reference-table-unit');
    assert.equal(cells[0].children[1].textContent, 'МДж/м3 ');
    assert.equal(cells[1].children[0].textContent, 'pm.кр, ');
    assert.equal(cells[3].children.length, 1);
    assert.equal(cells[6].colSpan, 2);
    assert.equal(cells[6].children[0].textContent, 'V0 при cq=15000');
    assert.equal(cells.every(cell => cell.scope === 'col'), true);

    const style = readFileSync(new URL('./style.css', import.meta.url), 'utf8');
    assert.match(style, /\.reference-table-label \{ display: inline; \}/);
    assert.match(style, /\.reference-table-unit \{[^}]*font-size: 12px;/);
    assert.match(style, /\.reference-table\[data-table-id="slukhotsky-table-1"\] \{ min-width: 1060px; \}/);
    assert.match(style, /\.reference-table\[data-table-id="slukhotsky-table-1"\] th \{ white-space: nowrap; \}/);
});

test('preparation screen embeds tables without visible unknown-unit warnings', () => {
    const main = readFileSync(new URL('./main.js', import.meta.url), 'utf8');
    const style = readFileSync(new URL('./style.css', import.meta.url), 'utf8');
    const tableView = readFileSync(new URL('./preparation-table-view.mjs', import.meta.url), 'utf8');
    const preparationScreenTransition = main.slice(
        main.indexOf('function setPreparationScreen(active)'),
        main.indexOf('async function save()'),
    );

    assert.doesNotMatch(main, /Одиницю не встановлено/);
    assert.doesNotMatch(main, /Одиниці вимірювання цих коефіцієнтів у книзі не вказані/);
    assert.doesNotMatch(main, /workbook-unknown-unit/);
    assert.doesNotMatch(main, /preparation-tables-dialog/);
    assert.doesNotMatch(main, /Початкові дані|Вибрані коефіцієнти/);
    assert.doesNotMatch(main, /preparation-reference-toggle|preparation-reference-icon|preparation-reference-body"[^>]*hidden/);
    assert.doesNotMatch(main, /preparation-configuration|Конфігурація: \$\{state\.active\.name\}/);
    assert.doesNotMatch(style, /workbook-unit-note|workbook-unknown-unit|#preparation-tables-dialog/);
    assert.doesNotMatch(main, /preparation-reference-title|>Довідкові таблиці<\/h2>/);
    assert.doesNotMatch(style, /\.preparation-reference-title/);
    assert.doesNotMatch(tableView, /reference-table-title|align-self:\s*center/);

    assert.match(main, /id="preparation-reference-section"/);
    assert.match(main, /id="preparation-reference-section"[^>]*aria-label="Таблиці для довідки"/);
    assert.match(main, /id="preparation-reference-body"[^>]*>\s*<div id="preparation-table-tabs"/);
    assert.doesNotMatch(main, /id="preparation-reference-body"[^>]*hidden/);
    assert.match(main, /id="preparation-nav"[^>]*class="header-nav-button"/);
    assert.match(main, /id="preparation-nav"[\s\S]*?\$\('preparation-nav'\)\.textContent = preparationScreen \? 'Робочий екран' : 'Підготовчі дані'/);
    assert.match(main, /id="workbook-input-fields"/);
    assert.doesNotMatch(main, /id="workbook-coefficient-fields"/);
    assert.match(main, /tab\.textContent = table\.title/);
    assert.match(main, /document\.getElementById\('workbook-input-fields'\)\.append\(element\)/);
    assert.match(main, /id="preparation-table-tabs"[^>]*role="tablist"/);
    assert.match(main, /id="preparation-table-content"[^>]*role="tabpanel"/);
    assert.match(main, /<select id="mode" hidden aria-hidden="true" tabindex="-1">/);
    assert.doesNotMatch(main, /Режим<\/label>/);
    assert.match(main, /document\.body\.classList\.toggle\('preparation-screen-active', active\)/);
    assert.match(main, /\$\('mode'\)\.value = state\.active\.mode/);
    assert.doesNotMatch(preparationScreenTransition, /\$\('mode'\)\.value\s*=/);
    assert.match(style, /html \{[^}]*overflow-y: scroll;[^}]*scrollbar-gutter: stable;/);
    assert.match(style, /body \{[^}]*overflow-x: hidden;[^}]*overflow-y: visible;/);
    assert.match(style, /\.reference-table-scroll \{[^}]*overflow-x: auto;/);
    assert.match(style, /\.reference-table-content \{[^}]*min-height:\s*540px;/);
    assert.doesNotMatch(style, /\.reference-table-content \{[^}]*overflow-y:/);
    assert.match(style, /\.workbook-input-fields \{[^}]*repeat\(4,\s*minmax\(0,\s*1fr\)\)/);
    assert.doesNotMatch(style, /\.preparation-screen \{[^}]*overflow:\s*hidden/);
});

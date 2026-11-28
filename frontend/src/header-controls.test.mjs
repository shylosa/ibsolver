import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';

const main = readFileSync(new URL('./main.js', import.meta.url), 'utf8');
const style = readFileSync(new URL('./style.css', import.meta.url), 'utf8');
const toolbar = main.match(/<header>[\s\S]*?<\/header>/)?.[0] ?? '';
const inputsForm = main.match(/<form id="inputs"[\s\S]*?<\/form>/)?.[0] ?? '';
const workbookForm = main.match(/<form id="workbook-preparation-form"[\s\S]*?<\/form>/)?.[0] ?? '';

test('one shared toolbar owns all primary commands', () => {
    assert.ok(toolbar);
    for (const id of ['reload', 'save', 'calculate', 'cancel', 'toggle-parameters', 'preparation-nav']) {
        assert.equal((toolbar.match(new RegExp(`id="${id}"`, 'g')) || []).length, 1, id);
    }
    for (const id of ['save', 'calculate', 'cancel']) {
        assert.doesNotMatch(inputsForm, new RegExp(`id="${id}"`), id);
        assert.doesNotMatch(workbookForm, new RegExp(`id="${id}"`), id);
    }
    assert.doesNotMatch(main, /quick-calculate|workspace-back|preparation-save/);
    assert.ok(toolbar.indexOf('header-primary-actions') < toolbar.indexOf('header-navigation-actions'));
    assert.ok(toolbar.indexOf('header-data-actions') < toolbar.indexOf('header-calculation-actions'));
    assert.ok(toolbar.indexOf('header-calculation-actions') < toolbar.indexOf('header-navigation-actions'));
    assert.ok(toolbar.indexOf('header-navigation-actions') < toolbar.indexOf('id="toggle-parameters"'));
    assert.ok(toolbar.indexOf('id="toggle-parameters"') < toolbar.indexOf('id="preparation-nav"'));
});

test('screen navigation reuses a fixed-size right-aligned button without changing mode', () => {
    assert.match(toolbar, /id="preparation-nav"[^>]*class="header-nav-button"/);
    assert.doesNotMatch(toolbar, /id="preparation-nav"[^>]*class="[^"]*\bstage\b/);
    assert.match(main, /\$\('preparation-nav'\)\.textContent = preparationScreen \? 'Робочий екран' : 'Підготовчі дані'/);
    assert.match(main, /\$\('preparation-nav'\)\.addEventListener\('click', \(\) => operation\(async \(\) => \{[\s\S]*?setPreparationScreen\(showPreparation\)/);
    assert.match(style, /\.header-nav-button \{[^}]*width:\s*160px;[^}]*height:\s*32px;/);
    assert.match(style, /\.header-actions \{[^}]*justify-content:\s*space-between;/);
    assert.match(style, /\.header-navigation-actions \{[^}]*margin-left:\s*auto;/);
    assert.doesNotMatch(style, /\.header-nav-button \{[^}]*margin-left:\s*auto;/);

    const navigationHandler = main.slice(
        main.indexOf("$('preparation-nav').addEventListener('click'"),
        main.indexOf("$('configurations').addEventListener('change'"),
    );
    assert.doesNotMatch(navigationHandler, /\$\('mode'\)\.value\s*=|SaveConfiguration|SaveSettings/);
    assert.match(main, /\$\('mode'\)\.value = state\.active\.mode/);
});

test('mode remains internal and hidden while loaded configuration mode drives the UI', () => {
    assert.match(main, /<select id="mode" hidden aria-hidden="true" tabindex="-1">/);
    assert.doesNotMatch(main, /<label[^>]*>Режим/);
    assert.match(main, /\$\('mode'\)\.value = state\.active\.mode/);
    assert.match(main, /\$\('configurations'\)\.addEventListener\('change'/);
    assert.match(main, /display\(await SaveSettings\(\{...state\.settings, activeConfigurationId: id\}\)\)/);
    assert.match(main, /\$\('mode'\)\.value = state\.active\.mode/);
    assert.match(main, /mode === 'oscillator'/);
});

test('toolbar commands follow screen, mode, dirty, busy, and parameter-collapse state', () => {
    assert.match(main, /\$\('save'\)\.disabled = busy \|\| !dirty/);
    assert.match(main, /\$\('calculate'\)\.hidden = preparationScreen;/);
    assert.match(main, /\$\('cancel'\)\.hidden = preparationScreen \|\| !\(numerical \|\| preparation\)/);
    assert.match(main, /\$\('toggle-parameters'\)\.hidden = preparationScreen/);
    for (const handler of [
        "$('save').addEventListener('click', () => operation(save))",
        "$('calculate').addEventListener('click', () => operation(runCalculate))",
        "$('cancel').addEventListener('click'",
        "$('toggle-parameters').addEventListener('click', () => toggleParameters())",
    ]) assert.ok(main.includes(handler), handler);
});

test('manual database reload remains available alongside startup restoration', () => {
    assert.match(main, /\$\('reload'\)\.addEventListener\('click', \(\) => operation\(async \(\) => \{[\s\S]*?display\(await LoadState\(\)\)/);
    assert.match(main, /operation\(async \(\) => \{\s*display\(await LoadState\(\)\)/);
});

test('parameters panel stays visible on the preparation screen even when collapsed on the workspace', () => {
    assert.match(style, /\.parameters\.collapsed \{ display: none !important; \}/);
    assert.match(style, /\.preparation-screen-active \.parameters\.collapsed \{ display: block !important; \}/);
    assert.match(main, /\$\('toggle-parameters'\)\.hidden = preparationScreen/);
});

test('field help buttons stay out of the Tab order so Tab moves between input fields', () => {
    const occurrences = main.match(/helpButton\.tabIndex = -1/g) || [];
    assert.equal(occurrences.length, 2, 'workspace fields and workbook fields');
    assert.doesNotMatch(main, /helpButton\.tabIndex = 0/);
});

test('every workspace input carries a parameter-name badge inside the field', () => {
    assert.match(main, /valueField\.className = unit \? 'field-with-unit' : 'field-input-wrap'/);
    assert.match(main, /badge\.className = 'field-param-badge';\s*\n\s*badge\.textContent = technicalId;/);
    assert.match(main, /valueField\.append\(badge\);\s*\n\s*controls\.append\(valueField\);/);
    assert.match(style, /\.field-param-badge \{[^}]*position: absolute;[^}]*pointer-events: none;/);
    assert.match(style, /\.field-with-unit \{[^}]*position: relative;/);
    assert.match(style, /\.field-input-wrap \{[^}]*position: relative;/);
});

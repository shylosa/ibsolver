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
    for (const id of ['reload', 'save', 'calculate', 'demo', 'cancel', 'toggle-parameters', 'preparation-nav']) {
        assert.equal((toolbar.match(new RegExp(`id="${id}"`, 'g')) || []).length, 1, id);
    }
    for (const id of ['save', 'calculate', 'demo', 'cancel']) {
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

test('toolbar commands follow screen, mode, dirty, busy, and parameter-collapse state', () => {
    assert.match(main, /\$\('save'\)\.disabled = busy \|\| !dirty/);
    assert.match(main, /\$\('calculate'\)\.hidden = preparationScreen \|\| mode === 'demo'/);
    assert.match(main, /\$\('demo'\)\.hidden = preparationScreen \|\| mode !== 'demo'/);
    assert.match(main, /\$\('cancel'\)\.hidden = preparationScreen \|\| !\(numerical \|\| preparation\)/);
    assert.match(main, /\$\('toggle-parameters'\)\.hidden = preparationScreen/);
    for (const handler of [
        "$('save').addEventListener('click', () => operation(save))",
        "$('calculate').addEventListener('click', () => operation(runCalculate))",
        "$('demo').addEventListener('click'",
        "$('cancel').addEventListener('click'",
        "$('toggle-parameters').addEventListener('click', () => toggleParameters())",
    ]) assert.ok(main.includes(handler), handler);
});

test('manual database reload remains available alongside startup restoration', () => {
    assert.match(main, /\$\('reload'\)\.addEventListener\('click', \(\) => operation\(async \(\) => \{[\s\S]*?display\(await LoadState\(\)\)/);
    assert.match(main, /operation\(async \(\) => \{\s*display\(await LoadState\(\)\)/);
});

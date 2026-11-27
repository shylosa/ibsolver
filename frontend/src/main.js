import './style.css';
import {LoadState, SaveConfiguration, SaveSettings, Calculate, CalculateWorkbook, CancelCalculation, ImportConfiguration, ExportConfiguration, ExportCSV, CloseWindow, PreviewExternalImport, CancelExternalImport, SaveExternalImport, OpenExternalResult, DeleteExternalResult, ExportExternalCSV, RenameConfiguration, DeleteConfiguration, DuplicateConfiguration} from '../wailsjs/go/main/App';
import {TimeSeriesChart} from './chart';
import {EventsOn} from '../wailsjs/runtime/runtime';
import {preparationFields, workbookPreparationFields, preparationResultPanel, KT_CUSTOM_VALUE, parsePreparation, parseWorkbookPreparation, parseKTSelection, ktSelectValue, displayPreparation} from './preparation.mjs';
import {preparationTables} from './preparation-tables.mjs';
import {renderPreparationTable} from './preparation-table-view.mjs';
import {externalMetadata} from './external.mjs';
import {workbookResultSections} from './workbook-results.mjs';
import {workbookTransferMatches, workbookTransferRows} from './workbook-transfer.mjs';
import {DebouncedTask} from './debounce.mjs';

document.querySelector('#app').innerHTML = `
<header><div class="header-actions">
        <div class="header-primary-actions">
            <div class="header-data-actions">
                <button id="reload" type="button">Відновити з БД</button>
                <button id="save" type="button" disabled>Зберегти</button>
            </div>
            <div class="header-calculation-actions">
                <button id="calculate" type="button" class="primary">Розрахувати</button>
                <button id="cancel" type="button" disabled>Скасувати</button>
            </div>
        </div>
        <div class="header-navigation-actions">
            <button id="toggle-parameters" class="toggle-parameters-btn" aria-expanded="true" title="Згорнути або розгорнути панель параметрів для звільнення робочого простору">▲ Згорнути параметри</button>
            <button id="preparation-nav" class="header-nav-button" type="button">Підготовчі дані</button>
        </div>
    </div></header>
<section class="parameters" aria-label="Параметри конфігурації">
    <div class="configuration-row"><label>Конфігурація<select id="configurations"></select></label>
        <select id="mode" hidden aria-hidden="true" tabindex="-1"><option value="preparation">Підготовчі дані</option><option value="oscillator">Осцилятор · RK45</option></select>
        <button id="new" title="Створити порожню конфігурацію">Нова</button><button id="rename" title="Змінити назву активної конфігурації">Перейменувати</button><button id="duplicate" title="Створити копію активної конфігурації">Дублювати</button><button id="delete" title="Видалити активну конфігурацію разом з її результатами">Видалити…</button>
        <button id="import-json">Імпорт JSON</button><button id="export-json">Експорт JSON</button><span id="save-state" role="status"></span></div>
    <form id="inputs" novalidate>    <div id="preparation-fields">
        <div id="preparation-main" class="preparation-grid">
            <section class="preparation-column"><h3>Лімітні параметри</h3><div class="preparation-column-fields"></div></section>
            <section class="preparation-column"><h3>Характеристики ствола</h3><div class="preparation-column-fields"></div></section>
            <section class="preparation-column"><h3>Характеристики заряду</h3><div class="preparation-column-fields"></div></section>
            <section class="preparation-column"><h3>Характеристики пороху</h3><div class="preparation-column-fields"></div></section>
            <section class="preparation-column"><h3>Характеристики запалювача</h3><div class="preparation-column-fields"></div></section>
        </div>
        </div><div id="oscillator-fields"><div class="numerical-fields">
        <label>Маса m, кг<input id="n-mass" type="number" min="1e-12" step="any" required></label>
        <label>Жорсткість k, Н/м<input id="n-stiffness" type="number" min="1e-12" step="any" required></label>
        <label>Опір c, кг/с<input id="n-damping" type="number" min="0" step="any" required></label>
        <label>Зміщення x₀, м<input id="n-position" type="number" step="any" required></label>
        <label>Швидкість v₀, м/с<input id="n-velocity" type="number" step="any" required></label>
        <label>Тривалість, с<input id="n-duration" type="number" min="1e-12" max="10000" step="any" required></label>
        <label>Кількість точок<input id="n-samples" type="number" min="2" max="20000" step="1" required></label>
    </div>
    <details><summary>Точність і кроки RK45</summary><div class="solver-fields">
        <label>Абс. допуск (м; м/с)<input id="s-absTolerance" type="number" min="1e-15" step="any" required></label>
        <label>Відносний допуск<input id="s-relTolerance" type="number" min="1e-15" step="any" required></label>
        <label>Початковий крок, с<input id="s-initialStep" type="number" min="1e-15" step="any" required></label>
        <label>Мінімальний крок, с<input id="s-minStep" type="number" min="1e-15" step="any" required></label>
        <label>Максимальний крок, с<input id="s-maxStep" type="number" min="1e-15" step="any" required></label>
        <label>Ліміт спроб кроку<input id="s-maxSteps" type="number" min="1" max="1000000" step="1" required></label>
    </div><p class="hint">Допуски керують оцінкою локальної похибки, а не гарантують глобальну точність. Абсолютний допуск застосовується окремо до x та v у їхніх одиницях.</p></details>
    <p class="hint">m·x″ + c·x′ + k·x = 0. Математичний приклад у SI, не спеціалізований інженерний розрахунок.</p></div>
    </form>
</section>
<section id="preparation-screen" class="preparation-screen" hidden aria-labelledby="preparation-screen-title">
    <div class="preparation-screen-heading">
        <h1 id="preparation-screen-title" tabindex="-1">Підготовчі дані</h1>
    </div>
    <div class="preparation-screen-grid">
        <form id="workbook-preparation-form" class="workbook-preparation-form" novalidate>
            <section class="workbook-field-group" aria-label="Сім полів підготовчих даних">
                <div id="workbook-input-fields" class="workbook-input-fields"></div>
                <button id="workbook-calculate" type="button" class="primary">Розрахувати</button>
            </section>
        </form>
        <aside id="preparation-results" class="preparation-results" aria-label="${preparationResultPanel.title}">
            <div class="section-title">${preparationResultPanel.title}</div>
            <p id="preparation-result-state" class="preparation-result-state">Результатів ще немає</p>
            <dl id="preparation-result-body" hidden></dl>
            <ul id="preparation-result-warnings" class="preparation-result-warnings" hidden></ul>
        </aside>
    </div>
    <section id="preparation-reference-section" class="preparation-reference-section" aria-label="Таблиці для довідки">
        <div id="preparation-reference-body" class="preparation-reference-body">
            <div id="preparation-table-tabs" class="reference-tabs" role="tablist" aria-label="Довідкові таблиці"></div>
            <section id="preparation-table-content" class="reference-table-content" role="tabpanel" tabindex="0"></section>
        </div>
    </section>
</section>
<main id="workspace"><aside id="results"><div class="section-title">Результати <span id="result-kind" class="badge">RK45</span></div>
    <label class="history">Зовнішні результати<select id="external-results" aria-label="Збережені зовнішні результати"></select></label><div class="history-actions"><button id="external-metadata">Метадані</button><button id="external-delete">Видалити…</button></div>
    <p id="result-state"></p><dl id="summary"></dl><h3 id="statistics-title" hidden>Інтегратор RK45</h3><dl id="statistics" hidden></dl></aside>
    <div id="splitter" role="separator" aria-label="Ширина результатів" aria-orientation="vertical" aria-valuemin="180" aria-valuemax="480" tabindex="0"></div>
    <section class="chart-workspace">
        <div class="chart-toolbar"><div id="legend" aria-label="Видимі ряди"></div><div class="chart-actions"><button id="import-csv">Імпорт CSV</button><button id="export-csv" disabled>CSV</button></div></div>
        <div class="canvas-wrap"><canvas id="chart" aria-label="Спільне поле нормалізованих часових рядів"></canvas></div><div id="cursor" aria-live="off"></div></section></main>
<dialog id="discard-dialog"><p>Не вдалося зберегти поточну чернетку. Поверніться до редагування або явно відкиньте незбережені зміни.</p><form method="dialog"><button value="return" autofocus>Повернутися до редагування</button><button value="discard">Відкинути незбережені зміни</button></form></dialog>
<dialog id="rename-dialog"><form method="dialog"><h3>Перейменувати конфігурацію</h3><label>Назва<input id="rename-name" maxlength="120" required></label><div class="actions"><button id="rename-confirm" value="save" class="primary">Перейменувати</button><button value="cancel" formnovalidate autofocus>Скасувати</button></div></form></dialog>
<dialog id="delete-dialog"><form method="dialog"><h3 id="delete-title">Видалити конфігурацію?</h3><div class="actions"><button id="delete-confirm" value="delete" class="danger">Видалити остаточно</button><button value="cancel" formnovalidate autofocus>Скасувати</button></div></form></dialog>
<dialog id="external-dialog"><h3 id="external-dialog-title"></h3><p>Числові часові ряди з CSV. Фізичний зміст визначає джерело даних.</p><dl id="external-details"></dl><form method="dialog"><label>Назва<input id="external-name" maxlength="120" required></label><div class="actions"><button id="external-confirm" value="save">Зберегти результат</button><button value="cancel" formnovalidate autofocus>Закрити / скасувати</button></div></form></dialog>
<dialog id="transfer-dialog"><h3>Передати результати на робочий екран</h3><p>Будуть змінені лише перелічені поля підготовчих даних. Інші параметри конфігурації та збережені результати розрахунків не змінюються; основний розрахунок не запускається.</p><table class="transfer-table"><thead><tr><th scope="col">Поле</th><th scope="col">Поточне значення</th><th scope="col">Нове значення</th></tr></thead><tbody id="transfer-rows"></tbody></table><form method="dialog"><button value="transfer" class="primary" autofocus>Передати на робочий екран</button><button value="cancel" formnovalidate>Скасувати</button></form></dialog>
<footer><span id="status" role="status">Відкриття бази даних…</span><span>SQLite · локальне збереження</span></footer>`;

const $ = id => document.getElementById(id);
const preparationGroups = {
    pmax: 0, vd: 0,
    d: 1, q: 1, ns: 1, hi: 1, af: 1, pfor: 1, lc: 1,
    kt: 2, tn: 2, forma: 2, omega: 2, w0: 2,
    ro: 3, tg15: 3, ku1: 3, f1: 3, u15: 3, svod: 3,
    pvos: 4, tvos: 4,
};
let fieldHelpItems = [];
let resultHelpItems = [];
for (const field of preparationFields) {
    const {key, technicalId, label, unit, description, options} = field;
    const element = document.createElement('div');
    element.className = 'preparation-field';
    const heading = document.createElement('div');
    heading.className = 'field-heading';
    const title = document.createElement('label');
    title.className = 'field-title';
    title.htmlFor = `p-${key}`;
    const helpButton = document.createElement('button');
    helpButton.id = `p-${key}-help-button`;
    helpButton.className = 'field-help-button';
    helpButton.type = 'button';
    helpButton.textContent = 'ⓘ';
    helpButton.setAttribute('aria-label', `Довідка: ${label}`);
    helpButton.setAttribute('aria-expanded', 'false');
    const helpPopup = document.createElement('div');
    helpPopup.id = `p-${key}-help`;
    helpPopup.className = 'field-help-popup';
    helpPopup.setAttribute('role', 'tooltip');
    helpPopup.hidden = true;
    const helpHeading = document.createElement('div');
    helpHeading.className = 'field-help-heading';
    const helpTitle = document.createElement('strong');
    helpTitle.textContent = unit ? `${label}, ${unit}` : label;
    const helpCode = document.createElement('code');
    helpCode.textContent = technicalId;
    helpHeading.append(helpTitle, helpCode);
    helpPopup.append(helpHeading);
    if (description) {
        const helpDescription = document.createElement('p');
        helpDescription.textContent = description;
        helpPopup.append(helpDescription);
    }
    helpButton.setAttribute('aria-controls', helpPopup.id);
    helpButton.setAttribute('aria-describedby', helpPopup.id);
    // Both pieces of the label target the same input when clicked.
    const lastSpace = label.lastIndexOf(' ');
    const lastWord = title.cloneNode();
    lastWord.textContent = label.slice(lastSpace + 1);
    const helpAnchor = document.createElement('span');
    helpAnchor.className = 'field-help-anchor';
    helpAnchor.append(lastWord, helpButton);
    if (lastSpace !== -1) {
        title.textContent = label.slice(0, lastSpace);
        heading.append(title, ' ');
    }
    heading.append(helpAnchor);

    const control = document.createElement(field.control === 'select' ? 'select' : 'input');
    control.id = `p-${key}`;
    control.setAttribute('aria-describedby', `p-${key}-message`);
    if (field.control === 'select') {
        control.add(new Option('Не задано', ''));
        for (const option of options) {
            control.add(new Option(`${option.value} - ${option.label}`, option.value));
        }
        if (key === 'kt') {
            control.add(new Option('Інше числове значення…', KT_CUSTOM_VALUE));
        }
    } else {
        control.type = 'text';
        control.inputMode = 'decimal';
        control.placeholder = 'Не задано';
    }
    const controls = document.createElement('div');
    controls.className = 'preparation-controls';
    if (field.control === 'text' && unit) {
        const valueField = document.createElement('div');
        valueField.className = 'field-with-unit';
        const suffix = document.createElement('span');
        suffix.className = 'field-unit';
        suffix.id = `p-${key}-unit`;
        suffix.textContent = unit;
        control.setAttribute('aria-describedby', `${suffix.id} p-${key}-message`);
        valueField.append(control, suffix);
        controls.append(valueField);
    } else {
        controls.append(control);
    }
    if (key === 'kt') {
        const customValue = document.createElement('input');
        customValue.id = 'p-kt-custom';
        customValue.type = 'text';
        customValue.inputMode = 'decimal';
        customValue.placeholder = 'Введіть значення KT';
        customValue.setAttribute('aria-label', 'Власне числове значення KT');
        customValue.setAttribute('aria-describedby', 'p-kt-message');
        customValue.dataset.preparationKey = 'kt';
        customValue.hidden = true;
        controls.append(customValue);
    }
    const message = document.createElement('span');
    message.id = `p-${key}-message`;
    message.className = 'field-message';
    element.append(heading, controls, message);
    document.querySelectorAll('#preparation-main .preparation-column-fields')[preparationGroups[key]].append(element);
    document.body.append(helpPopup);
    fieldHelpItems.push({button: helpButton, popup: helpPopup});
}

for (const field of workbookPreparationFields) {
    const element = document.createElement('div');
    element.className = 'workbook-preparation-field';
    const heading = document.createElement('div');
    heading.className = 'workbook-field-heading';
    const label = document.createElement('label');
    label.className = 'workbook-field-label';
    label.htmlFor = `workbook-${field.key}`;
    label.textContent = field.label;
    const helpButton = document.createElement('button');
    helpButton.id = `workbook-${field.key}-help-button`;
    helpButton.className = 'field-help-button';
    helpButton.type = 'button';
    helpButton.textContent = 'ⓘ';
    helpButton.setAttribute('aria-label', `Довідка: ${field.label}`);
    helpButton.setAttribute('aria-expanded', 'false');
    const helpPopup = document.createElement('div');
    helpPopup.id = `workbook-${field.key}-help`;
    helpPopup.className = 'field-help-popup';
    helpPopup.setAttribute('role', 'tooltip');
    helpPopup.hidden = true;
    const helpHeading = document.createElement('div');
    helpHeading.className = 'field-help-heading';
    const helpTitle = document.createElement('strong');
    helpTitle.textContent = field.unit ? `${field.label}, ${field.unit}` : field.label;
    const helpCode = document.createElement('code');
    helpCode.textContent = field.key;
    helpHeading.append(helpTitle, helpCode);
    const helpDescription = document.createElement('p');
    helpDescription.textContent = field.help;
    helpPopup.append(helpHeading, helpDescription);
    helpButton.setAttribute('aria-controls', helpPopup.id);
    helpButton.setAttribute('aria-describedby', helpPopup.id);
    heading.append(label, helpButton);

    const input = document.createElement('input');
    input.id = `workbook-${field.key}`;
    input.type = 'text';
    input.inputMode = 'decimal';
    input.placeholder = 'Не задано';
    input.autocomplete = 'off';
    input.dataset.preparationWorkbookKey = field.key;
    input.setAttribute('aria-describedby', `${field.unit ? `workbook-${field.key}-unit ` : ''}workbook-${field.key}-message`);
    const unit = document.createElement('span');
    const valueField = document.createElement('div');
    valueField.className = 'workbook-value-field';
    valueField.append(input);
    if (field.unit) {
        unit.id = `workbook-${field.key}-unit`;
        unit.className = 'field-unit';
        unit.textContent = field.unit;
        valueField.append(unit);
    }
    const message = document.createElement('span');
    message.id = `workbook-${field.key}-message`;
    message.className = 'field-message';
    element.append(heading, valueField, message);
    document.getElementById('workbook-input-fields').append(element);
    document.body.append(helpPopup);
    fieldHelpItems.push({button: helpButton, popup: helpPopup});
}

function positionFieldHelp(anchor, popup) {
    const rect = anchor.getBoundingClientRect();
    const margin = 8, gap = 6;
    const popupRect = popup.getBoundingClientRect();
    const maxLeft = Math.max(margin, window.innerWidth - popupRect.width - margin);
    const left = Math.max(margin, Math.min(rect.left, maxLeft));
    const below = rect.bottom + gap;
    const above = rect.top - popupRect.height - gap;
    const top = below + popupRect.height <= window.innerHeight - margin || above < margin
        ? Math.max(margin, Math.min(below, window.innerHeight - popupRect.height - margin))
        : above;
    popup.style.left = `${left}px`;
    popup.style.top = `${top}px`;
}
function showFieldHelp(anchor, popup) {
    clearTimeout(popup.hideTimer);
    popup.hidden = false;
    // Add .open on the next frame so the opacity transition runs from 0.
    requestAnimationFrame(() => { if (anchor.getAttribute('aria-expanded') === 'true') popup.classList.add('open'); });
    anchor.setAttribute('aria-expanded', 'true');
    positionFieldHelp(anchor, popup);
}
function hideFieldHelp(anchor, popup) {
    anchor.setAttribute('aria-expanded', 'false');
    popup.classList.remove('open');
    clearTimeout(popup.hideTimer);
    popup.hideTimer = setTimeout(() => { popup.hidden = true; }, 170);
}
function wireFieldHelp(anchor, popup) {
    let hovered = false, focused = false;
    anchor.addEventListener('mouseenter', () => { hovered = true; showFieldHelp(anchor, popup); });
    anchor.addEventListener('mouseleave', () => {
        hovered = false;
        hideFieldHelp(anchor, popup);
    });
    anchor.addEventListener('focus', () => { focused = true; showFieldHelp(anchor, popup); });
    anchor.addEventListener('blur', () => {
        focused = false;
        if (!hovered) hideFieldHelp(anchor, popup);
    });
    anchor.addEventListener('keydown', event => {
        if (event.key === 'Escape') {
            event.preventDefault();
            hideFieldHelp(anchor, popup);
        }
    });
}
for (const {button, popup} of fieldHelpItems) wireFieldHelp(button, popup);
function repositionFieldHelp() {
    for (const {button, popup} of fieldHelpItems) {
        if (!popup.hidden) positionFieldHelp(button, popup);
    }
}
window.addEventListener('resize', repositionFieldHelp);
window.addEventListener('scroll', repositionFieldHelp, true);
let state, busy = false, dirty = false, revision = 0, preparationScreen = false;
let draftQueue = Promise.resolve(), externalChart = null;
const delayedSave = new DebouncedTask(() => autosave(), 450);
const chart = new TimeSeriesChart($('chart'), $('legend'), $('cursor'), () => persistSettings());

function status(message, error = false) { $('status').textContent = message; $('status').classList.toggle('error', error); }
function errorMessage(error) {
    const message = String(error);
    for (const [kind, text] of Object.entries({
        canceled: 'Розрахунок скасовано.',
        step_limit: 'Вичерпано ліміт спроб кроку. Перевірте тривалість, кроки та допуски.',
        step_too_small: 'Інтегратор не може продовжити з заданим кроком і допусками.',
        non_finite: 'Обчислення дали нескінченне або нечислове значення. Перевірте параметри.',
        invalid_input: 'Перевірте вхідні значення, додатні допуски та порядок кроків інтегратора.',
    })) {
        if (message.includes(`ode: ${kind} `)) return text;
    }
    return message;
}
function setBusy(value) {
    busy = value;
    document.querySelectorAll('button, select, .parameters input, #preparation-screen input, #legend input').forEach(el => {
        if (!el.closest('dialog') && el.id !== 'toggle-parameters') el.disabled = value || el.id === 'cancel';
    });
    $('splitter').setAttribute('aria-disabled', String(value));
    $('export-csv').disabled = value || !hasChart();
    $('external-metadata').disabled = $('external-delete').disabled = value || !externalChart;
    if (state) updateMode();
    updateTransferAvailability();
}
// A workbook run has no time series, so only runs with charted series count.
function hasChart() { return Boolean(externalChart || visibleRun()?.chart?.series?.length); }
async function operation(action) {
    if (busy) return;
    delayedSave.flush();
    setBusy(true);
    let failed = false;
    try { await draftQueue; await action(); } catch (error) {
        failed = true;
        if (state) { $('configurations').value = String(state.active.id); $('mode').value = state.active.mode; }
        status(errorMessage(error), true);
    }
    finally { setBusy(false); if (failed) focusInvalid(); }
}
function width(value) {
    state.settings.resultsWidth = Math.max(180, Math.min(480, value));
    document.querySelector('main').style.setProperty('--results-width', `${state.settings.resultsWidth}px`);
    $('splitter').setAttribute('aria-valuenow', state.settings.resultsWidth);
}
function readConfiguration() {
    const parameters = structuredClone(state.active.parameters);
    const mode = state.active.mode;
    if (mode === 'preparation') {
        if (!preparationScreen) {
            const values = {}, inputFormats = {}, errors = [];
            for (const {key} of preparationFields) {
                try {
                    values[key] = readPreparationField(key);
                    inputFormats[key] = preparationInputFormat(key, values[key]);
                } catch (error) { errors.push([key, error]); }
            }
            if (errors.length) {
                const [, error] = errors[0];
                throw error;
            }
            parameters.preparation = {...values, inputFormats};
        }
    } else {
        const numerical = parameters.oscillator;
        for (const [group, prefix] of [['model', 'n-'], ['solver', 's-']]) {
            for (const key of Object.keys(numerical[group])) {
                const field = $(prefix + key), value = field.valueAsNumber;
                if (!validNumber(field, value)) throw new Error('Перевірте параметри чисельного прикладу та інтегратора.');
                numerical[group][key] = value;
            }
        }
        const s = numerical.solver;
        if (s.minStep > s.initialStep || s.initialStep > s.maxStep) throw new Error('Потрібно: мінімальний крок ≤ початковий ≤ максимальний.');
    }
    if (preparationScreen) {
        const values = {}, inputFormats = {};
        for (const {key} of workbookPreparationFields) {
            values[key] = readWorkbookPreparationField(key);
            inputFormats[key] = $(`workbook-${key}`).value;
        }
        parameters.preparationWorkbook = {...values, inputFormats};
    }
    return {...state.active, parameters};
}
function updateMode() {
    if (!state) return;
    const mode = state.active.mode, numerical = mode === 'oscillator', preparation = mode === 'preparation';
    $('preparation-nav').textContent = preparationScreen ? 'Робочий екран' : 'Підготовчі дані';
    $('preparation-nav').disabled = busy;
    $('save').disabled = busy || !dirty;
    $('rename').disabled = busy;
    $('duplicate').disabled = busy;
    $('delete').disabled = busy || state.configurations.length <= 1;
    $('calculate').hidden = preparationScreen;
    $('calculate').disabled = busy;
    $('cancel').hidden = preparationScreen || !(numerical || preparation);
    $('toggle-parameters').hidden = preparationScreen;
    for (const [id, active] of [['oscillator-fields', numerical], ['preparation-fields', preparation]]) {
        const el = $(id);
        if (el) {
            el.hidden = !active;
            el.querySelectorAll('input, select, button').forEach(field => { field.disabled = busy || !active; });
        }
    }
}
function preparationInputFormat(key, value) {
    const previous = state?.active.parameters.preparation?.inputFormats?.[key];
    if (typeof previous === 'string') {
        try {
            if (parsePreparation(previous, key) === value) return previous;
        } catch {}
    }
    if (key === 'kt' && $('p-kt').value !== KT_CUSTOM_VALUE) return $('p-kt').value;
    return key === 'kt' ? $('p-kt-custom').value : $('p-' + key).value;
}
function readPreparationField(key) {
    const control = $('p-' + key);
    const field = key === 'kt' && control.value === KT_CUSTOM_VALUE ? $('p-kt-custom') : control;
    const message = $('p-' + key + '-message');
    try {
        const stored = state?.active.parameters.preparation?.[key];
        const enteredValue = key === 'kt'
            ? parseKTSelection(control.value, $('p-kt-custom').value)
            : parsePreparation(field.value, key);
        const value = stored != null && (
            enteredValue === stored || field.value === displayPreparation(stored, key)
        ) ? stored : enteredValue;
        field.setAttribute('aria-invalid', 'false');
        control.setAttribute('aria-invalid', 'false');
        if (key === 'kt') $('p-kt-custom').setAttribute('aria-invalid', 'false');
        message.classList.remove('error');
        message.textContent = '';
        return value;
    } catch (error) {
        field.setAttribute('aria-invalid', 'true');
        message.textContent = error.message;
        message.classList.add('error');
        throw error;
    }
}
function readWorkbookPreparationField(key) {
    const field = $(`workbook-${key}`);
    const message = $(`workbook-${key}-message`);
    try {
        const value = parseWorkbookPreparation(field.value);
        field.setAttribute('aria-invalid', 'false');
        message.classList.remove('error');
        message.textContent = '';
        return value;
    } catch (error) {
        field.setAttribute('aria-invalid', 'true');
        message.textContent = error.message;
        message.classList.add('error');
        throw error;
    }
}
function visibleRun() { return state?.run?.mode === state?.active.mode ? state.run : null; }
// Explicit checks also work while operation() has disabled form controls.
function validNumber(field, value) {
    const valid = Number.isFinite(value) && (!field.min || value >= Number(field.min)) && (!field.max || value <= Number(field.max)) && (field.step !== '1' || Number.isInteger(value));
    field.setAttribute('aria-invalid', String(!valid));
    field.title = valid ? '' : 'Введіть скінченне число в допустимих межах поля.';
    return valid;
}
function resultState(stale = state.stale) {
    if (state.run && state.run.mode !== state.active.mode) {
        return `Збережений запуск іншого режиму: ${state.run.mode}, №${state.run.id}, ${new Date(state.run.createdAt).toLocaleString('uk-UA')}. Виберіть відповідний режим для перегляду.`;
    }
    return state.run ? (stale ? 'Попередній запуск · параметри змінено' : state.run.summary.kind === 'BALLISTICS' ? 'Збережений балістичний розрахунок' : state.run.summary.kind === 'NUMERICAL' ? 'Збережений чисельний результат' : state.run.summary.kind === 'WORKBOOK' ? 'Збережений підготовчий розрахунок' : 'Збережені демонстраційні дані') : 'Результатів ще немає';
}
// Workbook runs carry no chart; every result section renders as a bordered
// column card, with the transfer block F72:F74 first. Hovering a whole result
// row highlights it and shows its help popup (same styling as input fields).
let resultHelpSeq = 0;
let transferButton = null;
function clearResultHelp() {
    for (const item of resultHelpItems) item.popup.remove();
    fieldHelpItems = fieldHelpItems.filter(item => !resultHelpItems.includes(item));
    resultHelpItems = [];
}
function attachResultHelp(rowElement, term, help) {
    const popup = document.createElement('div');
    popup.id = `workbook-result-help-${resultHelpSeq++}`;
    popup.className = 'field-help-popup';
    popup.setAttribute('role', 'tooltip');
    popup.hidden = true;
    const heading = document.createElement('div');
    heading.className = 'field-help-heading';
    const title = document.createElement('strong');
    title.textContent = help.title;
    const code = document.createElement('code');
    code.textContent = help.code;
    heading.append(title, code);
    const description = document.createElement('p');
    description.textContent = help.description;
    popup.append(heading, description);
    rowElement.setAttribute('aria-describedby', popup.id);
    rowElement.tabIndex = 0;
    document.body.append(popup);
    wireFieldHelp(rowElement, popup);
    const item = {button: rowElement, popup};
    fieldHelpItems.push(item);
    resultHelpItems.push(item);
}
// Live values of the seven workbook inputs as entered right now; throws when
// any field holds an unfinished or invalid number.
function liveWorkbookValues() {
    const values = {};
    for (const {key} of workbookPreparationFields) values[key] = parseWorkbookPreparation($(`workbook-${key}`).value);
    return values;
}
// The transfer button is enabled only while the shown WORKBOOK run was
// calculated from exactly the seven inputs currently in the form.
function updateTransferAvailability() {
    if (!transferButton) return;
    const snapshot = visibleRun()?.snapshot?.preparationWorkbook;
    let matches = false;
    if (snapshot) {
        try { matches = workbookTransferMatches(snapshot, liveWorkbookValues()); } catch { matches = false; }
    }
    transferButton.disabled = !matches;
    transferButton.title = matches ? '' : 'Доступно, коли показаний результат відповідає поточним значенням семи полів підготовчих даних.';
}
function renderWorkbookResults(run) {
    const workbook = run?.summary?.kind === 'WORKBOOK' ? run.summary.workbook : null;
    const empty = $('preparation-result-state');
    const body = $('preparation-result-body'), warnings = $('preparation-result-warnings');
    clearResultHelp();
    transferButton = null;
    body.replaceChildren(...workbookResultSections(workbook).map((section, index) => {
        const group = document.createElement('div'), title = document.createElement('dt');
        group.className = 'preparation-result-column';
        title.className = 'preparation-result-section';
        title.textContent = section.title;
        group.append(title);
        for (const [term, value, help] of section.rows) {
            const row = document.createElement('div');
            row.className = 'preparation-result-row';
            const dt = document.createElement('dt'), dd = document.createElement('dd');
            dt.textContent = term;
            dd.textContent = value;
            row.append(dt, dd);
            if (help) attachResultHelp(row, term, help);
            group.append(row);
        }
        if (index === 0 && workbook) {
            const container = document.createElement('div');
            container.className = 'preparation-transfer-actions';
            transferButton = document.createElement('button');
            transferButton.id = 'workbook-transfer';
            transferButton.type = 'button';
            transferButton.textContent = 'Передати на робочий екран';
            // The button is recreated on every render, so its listener is
            // attached here and never from module scope.
            transferButton.addEventListener('click', () => operation(runWorkbookTransfer));
            container.append(transferButton);
            group.append(container);
        }
        return group;
    }));
    body.hidden = !workbook;
    empty.hidden = Boolean(workbook);
    updateTransferAvailability();
    const list = workbook?.warnings ?? [];
    warnings.replaceChildren(...list.map(message => {
        const item = document.createElement('li');
        item.textContent = message;
        return item;
    }));
    warnings.hidden = list.length === 0;
}
function display(next) {
    state = next; dirty = false; externalChart = null; revision++;
    document.querySelectorAll('.parameters input, .parameters select, #preparation-screen input').forEach(field => { field.removeAttribute('aria-invalid'); field.title = ''; });
    $('configurations').replaceChildren(...state.configurations.map(c => new Option(c.name, c.id, false, c.id === state.active.id)));
    $('configurations').title = state.active.name;
    $('mode').value = state.active.mode;
    updateMode();
    for (const [group, prefix] of [['model', 'n-'], ['solver', 's-']]) {
        for (const [key, value] of Object.entries(state.active.parameters.oscillator?.[group] || {})) $(prefix + key).value = value;
    }
    for (const {key} of preparationFields) {
        const preparation = state.active.parameters.preparation;
        const value = preparation?.[key];
        const inputFormat = preparation?.inputFormats?.[key];
        if (key === 'kt') {
            $('p-kt').value = ktSelectValue(value);
            $('p-kt-custom').value = displayPreparation(value, key, inputFormat);
            $('p-kt-custom').hidden = $('p-kt').value !== KT_CUSTOM_VALUE;
        } else {
            $('p-' + key).value = displayPreparation(value, key, inputFormat);
        }
        try { readPreparationField(key); } catch {}
    }
    renderWorkbookPreparationFields();
    $('save-state').textContent = 'Збережено';
    width(state.settings.resultsWidth);
    renderHistory();
    const run = visibleRun();
    $('result-state').textContent = resultState();
    $('result-state').classList.toggle('stale', state.stale);
    const numerical = run?.summary.kind === 'NUMERICAL';
    const ballistics = run?.summary.kind === 'BALLISTICS';
    const workbook = run?.summary.kind === 'WORKBOOK';
    $('result-kind').textContent = state.active.mode === 'preparation' ? (workbook ? 'КНИГА' : ballistics ? 'БАЛІСТИКА' : 'ДАНІ') : 'RK45';
    $('summary').replaceChildren();
    $('statistics').replaceChildren();
    $('statistics').hidden = $('statistics-title').hidden = !numerical;
    if (run) {
        let durationTerm = 'Тривалість, с';
        let durationVal = '—';
        if (run.summary && typeof run.summary.duration === 'number') {
            if (run.summary.duration < 0.1) {
                durationTerm = 'Тривалість, мс';
                durationVal = (run.summary.duration * 1000).toFixed(2);
            } else {
                durationVal = run.summary.duration.toPrecision(6);
            }
        }
        const rows = [
            [run.summary.configurationName ? 'Назва запуску' : 'Поточна назва', run.summary.configurationName || state.active.name],
            ['Режим запуску', run.mode === 'preparation' ? (workbook ? 'Підготовчі дані · книга' : 'Балістика (OZVB)') : 'Осцилятор'],
            ['Точок / серію', String(run.summary.samples ?? 0)],
            [durationTerm, durationVal],
            ['Серій', String(run.chart?.series?.length ?? 0)],
            ['Збережено', new Date(run.createdAt).toLocaleString('uk-UA')]
        ];
        if (ballistics && run.summary.ballistics) {
            const b = run.summary.ballistics;
            rows.push(
                ['Макс. тиск Pm, МПа', b.maxPressure != null ? b.maxPressure.toFixed(1) : '—'],
                ['Дульна швидкість Vd, м/с', b.muzzleVelocity != null ? b.muzzleVelocity.toFixed(1) : '—'],
                ['Час руху снаряда td, мс', b.muzzleTime != null ? (b.muzzleTime * 1000).toFixed(2) : '—'],
                ['Шлях снаряда Ld, м', b.barrelTravel != null ? b.barrelTravel.toFixed(3) : '—'],
                ['Маса заряду ω, кг', b.chargeMass != null ? b.chargeMass.toFixed(4) : '—'],
                ['Обʼєм камори W₀, л', b.chamberVolume != null ? (b.chamberVolume * 1000).toFixed(3) : '—'],
                ['Положення кінця горіння ηk', b.burntFraction != null ? b.burntFraction.toFixed(3) : '—'],
                ['Розрахункова маса ствола, кг', b.barrelMass != null ? b.barrelMass.toFixed(1) : '—'],
            );
        } else if (numerical) {
            const n = run.summary.numerical, snapshot = run.snapshot.oscillator;
            rows.push(
                ['x(T), м', n.finalPosition.toPrecision(6)],
                ['v(T), м/с', n.finalVelocity.toPrecision(6)],
                ['E(0), Дж', n.initialEnergy.toPrecision(6)],
                ['E(T), Дж', n.finalEnergy.toPrecision(6)],
                ['Кроки / відхилено', `${n.statistics.accepted} / ${n.statistics.rejected}`],
                ['Обчислень RHS', String(n.statistics.evaluations)],
                ['Допуски абс. / відн.', `${snapshot.solver.absTolerance} / ${snapshot.solver.relTolerance}`],
                ['Знімок m, кг', String(snapshot.model.mass)],
                ['Знімок k, Н/м', String(snapshot.model.stiffness)],
                ['Знімок c, кг/с', String(snapshot.model.damping)],
                ['Знімок x₀, м', String(snapshot.model.position)],
                ['Знімок v₀, м/с', String(snapshot.model.velocity)]
            );
        }
        for (const [key, value] of rows) {
            const term = document.createElement('dt'), description = document.createElement('dd');
            term.textContent = key;
            description.textContent = value;
            (['Кроки / відхилено', 'Обчислень RHS', 'Допуски абс. / відн.'].includes(key) ? $('statistics') : $('summary')).append(term, description);
        }
    }
    renderWorkbookResults(run);
    chart.set(run?.chart?.series?.length ? run.chart : null, state.settings);
    status(state.warning || 'Робочий стан відновлено з бази даних.');
}
function renderWorkbookPreparationFields() {
    const saved = state.active.parameters.preparationWorkbook;
    for (const {key} of workbookPreparationFields) {
        const field = $(`workbook-${key}`);
        const value = saved?.[key];
        const format = saved?.inputFormats?.[key];
        let text = value == null ? '' : String(value);
        if (typeof format === 'string') {
            try {
                if (parseWorkbookPreparation(format) === value) text = format;
            } catch {}
        }
        field.value = text;
        try { readWorkbookPreparationField(key); } catch {}
    }
}
function setPreparationScreen(active) {
    preparationScreen = active;
    $('preparation-screen').hidden = !active;
    $('inputs').hidden = active;
    document.body.classList.toggle('preparation-screen-active', active);
    updateMode();
    if (active) {
        renderWorkbookPreparationFields();
        $('preparation-screen-title').focus({preventScroll: true});
    } else {
        chart.requestDraw();
    }
}
async function save() {
    delayedSave.cancel();
    await draftQueue;
    if (!dirty) return;
    try { await queueSave(); } catch (error) { focusInvalid(); throw error; }
    status('Конфігурацію збережено.');
}
function focusInvalid() {
    const field = document.querySelector('#preparation-screen input[aria-invalid="true"], .parameters input[aria-invalid="true"], .parameters select[aria-invalid="true"]');
    if (field) { const details = field.closest('details'); if (details) details.open = true; field.focus(); }
}
async function flushForTransition() {
    try { if (dirty) await save(); }
    catch (error) {
        status(errorMessage(error), true);
        const dialog = $('discard-dialog');
        const choice = new Promise(resolve => dialog.addEventListener('close', () => resolve(dialog.returnValue), {once: true}));
        dialog.showModal();
        if (await choice !== 'discard') throw error;
        await draftQueue;
        display(await LoadState());
    }
}
function markDirty(field) {
    dirty = true; revision++; $('save').disabled = busy;
    $('save-state').textContent = 'Є незбережені зміни';
    if (!externalChart && visibleRun()) {
        let changed = true;
        try {
            const mode = state.active.mode;
            const key = preparationScreen ? 'preparationWorkbook' : mode;
            changed = JSON.stringify(readConfiguration().parameters[key]) !== JSON.stringify(state.run.snapshot[key]);
        } catch {}
        $('result-state').textContent = changed ? 'Попередній запуск · параметри редагуються' : resultState(false);
        $('result-state').classList.toggle('stale', changed);
    }
    updateTransferAvailability();
}
// All writes share one queue. A response acknowledges only its captured revision.
function queueSave() {
    let configuration;
    try { configuration = readConfiguration(); }
    catch (error) { $('save-state').textContent = 'Є незбережені зміни · перевірте поля'; throw error; }
    const version = revision, configurationID = configuration.id;
    const settings = structuredClone(state.settings);
    $('save-state').textContent = 'Збереження…';
    const request = draftQueue.then(async () => {
        const next = await SaveConfiguration(configuration, settings);
        if (version !== revision || state.active.id !== configurationID) return;
        state = {...next, settings: state.settings}; dirty = false;
        $('save-state').textContent = 'Збережено'; $('save').disabled = true;
        if (!externalChart) {
            $('result-state').textContent = resultState();
            $('result-state').classList.toggle('stale', state.stale);
        }
        status(state.warning || 'Конфігурацію збережено.');
    });
    draftQueue = request.catch(error => {
        if (version === revision && state.active.id === configurationID) {
            dirty = true; $('save-state').textContent = 'Не збережено'; $('save').disabled = busy;
            status(errorMessage(error), true);
        }
    });
    return request;
}
function autosave() {
    if (busy || !state || !dirty) return;
    try { queueSave().catch(() => {}); } catch (error) { status(errorMessage(error), true); }
}
for (const field of document.querySelectorAll('.parameters input, #preparation-screen input')) {
    field.addEventListener('input', () => {
        markDirty(field);
        const workbookKey = field.dataset.preparationWorkbookKey;
        if (workbookKey) { try { readWorkbookPreparationField(workbookKey); } catch {} }
        const preparationKey = field.dataset.preparationKey || (field.id.startsWith('p-') ? field.id.slice(2) : null);
        if (preparationKey && preparationFields.some(item => item.key === preparationKey)) { try { readPreparationField(preparationKey); } catch {} }
        else if (field.type === 'number') validNumber(field, field.valueAsNumber);
        delayedSave.schedule();
    });
}
for (const {key} of preparationFields) {
    $('p-' + key).addEventListener('blur', () => { try { readPreparationField(key); } catch {} });
    $('p-' + key).addEventListener('change', event => {
        if (key === 'kt') $('p-kt-custom').hidden = event.currentTarget.value !== KT_CUSTOM_VALUE;
        markDirty(event.currentTarget);
        try { readPreparationField(key); } catch {}
        delayedSave.schedule();
    });
}
for (const {key} of workbookPreparationFields) {
    $(`workbook-${key}`).addEventListener('blur', () => { try { readWorkbookPreparationField(key); } catch {} });
}
$('import-json').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    const next = await ImportConfiguration();
    if (next) { display(next); status('Конфігурацію імпортовано з новим ідентифікатором.'); }
}));
$('export-json').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    if (await ExportConfiguration(state.active.id)) status('Конфігурацію експортовано.');
}));
$('export-csv').addEventListener('click', () => operation(async () => {
    const saved = externalChart ? await ExportExternalCSV(state.active.id, externalChart.id) : await ExportCSV(state.active.id);
    if (saved) status('Збережені часові ряди експортовано без перетворень графіка.');
}));
$('inputs').addEventListener('submit', event => { event.preventDefault(); operation(save); });
$('workbook-preparation-form').addEventListener('submit', event => { event.preventDefault(); operation(save); });
$('save').addEventListener('click', () => operation(save));
$('preparation-nav').addEventListener('click', () => operation(async () => {
    const showPreparation = !preparationScreen;
    await flushForTransition();
    setPreparationScreen(showPreparation);
}));
$('configurations').addEventListener('change', event => {
    const id = Number(event.target.value);
    operation(async () => {
        await flushForTransition();
        display(await SaveSettings({...state.settings, activeConfigurationId: id}));
    });
});
$('mode').addEventListener('change', event => {
    const mode = event.target.value;
    operation(async () => {
        await flushForTransition();
        const parameters = structuredClone(state.active.parameters);
        if (mode === 'preparation') parameters.preparation ||= Object.fromEntries(preparationFields.map(({key}) => [key, null]));
        display(await SaveConfiguration({...state.active, mode, parameters}, state.settings));
        status('Режим змінено. Параметри режимів збережено.');
    }).finally(() => { $('mode').value = state.active.mode; });
});
function unusedConfigurationName(base) {
    const names = new Set(state.configurations.map(c => c.name));
    if (!names.has(base)) return base;
    for (let i = 2; ; i++) {
        const candidate = `${base} ${i}`;
        if (!names.has(candidate)) return candidate;
    }
}
$('new').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    const parameters = {
        preparation: Object.fromEntries(preparationFields.map(({key}) => [key, null])),
        preparationWorkbook: Object.fromEntries(workbookPreparationFields.map(({key}) => [key, null])),
    };
    display(await SaveConfiguration({id: 0, name: unusedConfigurationName('Нова конфігурація'), mode: 'preparation', parameters}, state.settings));
    status('Створено порожню конфігурацію.');
}));
$('duplicate').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    display(await DuplicateConfiguration(state.active.id));
    status('Конфігурацію продубльовано; відкрито копію.');
}));
$('rename').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    $('rename-name').value = state.active.name;
    const dialog = $('rename-dialog');
    const choice = new Promise(resolve => dialog.addEventListener('close', () => resolve(dialog.returnValue), {once: true}));
    dialog.showModal();
    if (await choice !== 'save') return;
    const name = $('rename-name').value.trim();
    if (!name || [...name].length > 120) throw new Error('Вкажіть назву конфігурації (до 120 символів).');
    display(await RenameConfiguration(state.active.id, name));
    status('Назву конфігурації змінено.');
}));
$('delete').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    $('delete-title').textContent = `Видалити конфігурацію «${state.active.name}»?`;
    const dialog = $('delete-dialog');
    const choice = new Promise(resolve => dialog.addEventListener('close', () => resolve(dialog.returnValue), {once: true}));
    dialog.showModal();
    if (await choice !== 'delete') return;
    display(await DeleteConfiguration(state.active.id));
    status('Конфігурацію та всі її результати видалено.');
}));
async function runCalculate() {
    if (dirty) await save();
    const isPrep = state.active.mode === 'preparation';
    status(isPrep ? 'Виконується балістичний розрахунок…' : 'Виконується чисельне інтегрування RK45…');
    $('calculate').textContent = 'Обчислення…';
    $('cancel').disabled = false;
    try {
        display(await Calculate(state.active.id));
        status(isPrep ? 'Балістичний розрахунок завершено та збережено.' : 'Чисельний приклад розраховано та збережено.');
    }
    finally {
        $('calculate').textContent = 'Розрахувати';
        $('cancel').disabled = true;
    }
}
$('calculate').addEventListener('click', () => operation(runCalculate));

async function runWorkbookCalculate() {
    if (dirty) await save();
    status('Виконується підготовчий розрахунок за книгою Диплом.xls…');
    display(await CalculateWorkbook(state.active.id));
    status('Підготовчий розрахунок завершено та збережено. Часових рядів він не створює.');
}
$('workbook-calculate').addEventListener('click', () => operation(runWorkbookCalculate));

// Transfer of the workbook block F72:F74 onto the working screen is an
// explicit, confirmed action: it rewrites only w0, pmax and lc, keeps every
// other parameter and stored run untouched, and never starts a calculation.
// Wired from the button inside renderWorkbookResults; the button does not
// exist at module scope.
async function runWorkbookTransfer() {
    const run = visibleRun();
    const snapshot = run?.summary?.kind === 'WORKBOOK' ? run.snapshot.preparationWorkbook : null;
    // Entered values gate the transfer; the dialog's "current" column shows
    // the working-screen fields being replaced, not the preparation inputs.
    let entered = null;
    try { entered = liveWorkbookValues(); } catch { entered = null; }
    const current = state?.active?.parameters?.preparation ?? null;
    const rows = entered && workbookTransferMatches(snapshot, entered)
        ? workbookTransferRows(run?.summary?.workbook?.transfer, current) : [];
    if (!rows.length) {
        status('Передачу скасовано: результат не відповідає поточним підготовчим даним.', true);
        return;
    }
    const body = $('transfer-rows');
    body.replaceChildren(...rows.map(row => {
        const tr = document.createElement('tr');
        const field = document.createElement('th');
        field.scope = 'row';
        field.textContent = `${row.technicalId} - ${row.label}, ${row.unit}`;
        const currentCell = document.createElement('td');
        const next = document.createElement('td');
        currentCell.textContent = row.currentText;
        next.textContent = row.newText;
        tr.append(field, currentCell, next);
        return tr;
    }));
    const dialog = $('transfer-dialog');
    dialog.returnValue = 'cancel';
    const choice = new Promise(resolve => dialog.addEventListener('close', () => resolve(dialog.returnValue), {once: true}));
    dialog.showModal();
    if (await choice !== 'transfer') { status('Передачу скасовано. Значення не змінено.'); return; }
    for (const row of rows) {
        const field = $('p-' + row.key);
        field.value = displayPreparation(row.newStored, row.key);
        try { readPreparationField(row.key); } catch {}
        markDirty(field);
    }
    setPreparationScreen(false);
    await save();
    status('Значення W0, pm, LS передано на робочий екран. Основний розрахунок не запущено.');
}

function toggleParameters(force) {
    const section = document.querySelector('.parameters');
    if (!section) return;
    const isCollapsed = force !== undefined ? force : !section.classList.contains('collapsed');
    section.classList.toggle('collapsed', isCollapsed);
    const toggleBtn = $('toggle-parameters');
    if (toggleBtn) {
        toggleBtn.textContent = isCollapsed ? '▼ Розгорнути параметри' : '▲ Згорнути параметри';
        toggleBtn.setAttribute('aria-expanded', String(!isCollapsed));
    }
    localStorage.setItem('ibsolver_params_collapsed', isCollapsed ? '1' : '0');
    if (chart && typeof chart.requestDraw === 'function') chart.requestDraw();
}
$('toggle-parameters').addEventListener('click', () => toggleParameters());

const preparationTableTabs = $('preparation-table-tabs');
for (const [index, table] of preparationTables.entries()) {
    const tab = document.createElement('button');
    tab.type = 'button';
    tab.id = `preparation-table-tab-${table.id}`;
    tab.dataset.tableId = table.id;
    tab.setAttribute('role', 'tab');
    tab.setAttribute('aria-label', table.title);
    tab.setAttribute('aria-controls', 'preparation-table-content');
    tab.setAttribute('aria-selected', String(index === 0));
    tab.tabIndex = index === 0 ? 0 : -1;
    tab.textContent = table.title;
    preparationTableTabs.append(tab);
}
function selectPreparationTable(id) {
    const table = preparationTables.find(item => item.id === id);
    if (!table) throw new Error(`Невідома довідкова таблиця: ${id}`);
    for (const tab of preparationTableTabs.querySelectorAll('[role="tab"]')) {
        const selected = tab.dataset.tableId === id;
        tab.setAttribute('aria-selected', String(selected));
        tab.tabIndex = selected ? 0 : -1;
    }
    $('preparation-table-content').setAttribute('aria-labelledby', `preparation-table-tab-${id}`);
    renderPreparationTable($('preparation-table-content'), table);
}
preparationTableTabs.addEventListener('click', event => {
    const tab = event.target.closest('[role="tab"]');
    if (tab && preparationTableTabs.contains(tab)) selectPreparationTable(tab.dataset.tableId);
});
preparationTableTabs.addEventListener('keydown', event => {
    const tabs = [...preparationTableTabs.querySelectorAll('[role="tab"]')];
    const index = tabs.indexOf(document.activeElement);
    if (index < 0) return;
    const next = event.key === 'ArrowRight' ? (index + 1) % tabs.length
        : event.key === 'ArrowLeft' ? (index - 1 + tabs.length) % tabs.length
        : event.key === 'Home' ? 0
        : event.key === 'End' ? tabs.length - 1
        : -1;
    if (next < 0) return;
    event.preventDefault();
    tabs[next].focus();
    selectPreparationTable(tabs[next].dataset.tableId);
});
selectPreparationTable(preparationTables[0].id);


$('cancel').addEventListener('click', async () => {
    $('cancel').disabled = true;
    try { await CancelCalculation(); status('Запит на скасування надіслано.'); }
    catch (error) { status(String(error), true); }
});
$('reload').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    display(await LoadState());
    status(`Базу даних перезчитано о ${new Date().toLocaleTimeString('uk-UA')} · конфігурацій: ${state.configurations.length} · активна: «${state.active.name}»`);
}));
function persistSettings() {
    if (!state || busy) return;
    operation(async () => {
        const settings = structuredClone(state.settings);
        await SaveSettings(settings);
        status('Налаштування графіка збережено.');
    });
}
const splitter = $('splitter');
splitter.addEventListener('pointerdown', event => {
    if (!state || busy) return;
    splitter.setPointerCapture(event.pointerId);
});
splitter.addEventListener('pointermove', event => {
    if (state && splitter.hasPointerCapture(event.pointerId)) width(Math.round(event.clientX - document.querySelector('main').getBoundingClientRect().left));
});
splitter.addEventListener('pointerup', event => {
    if (!splitter.hasPointerCapture(event.pointerId)) return;
    splitter.releasePointerCapture(event.pointerId); persistSettings();
});
splitter.addEventListener('keydown', event => {
    if (!state || busy || !['ArrowLeft', 'ArrowRight'].includes(event.key)) return;
    event.preventDefault(); width(state.settings.resultsWidth + (event.key === 'ArrowRight' ? 10 : -10)); persistSettings();
});
operation(async () => {
    display(await LoadState());
    if (localStorage.getItem('ibsolver_params_collapsed') === '1') {
        toggleParameters(true);
    }
});

EventsOn('close-request', () => operation(async () => {
    await flushForTransition();
    await CloseWindow();
}));

function renderHistory() {
    $('external-results').replaceChildren(new Option('— Збережений запуск IBSolver —', ''),
        ...(state.externalResults || []).map(r => new Option(`${r.name} · ${new Date(r.importedAt).toLocaleString('uk-UA')}`, r.id)));
    $('external-results').value = externalChart ? String(externalChart.id) : '';
}
function showExternal(result) {
    externalChart = result;
    chart.set(result.chart, state.settings);
    $('external-results').value = String(result.id);
    $('result-kind').textContent = 'ЗОВНІШНІ';
    $('result-state').textContent = 'Збережений зовнішній результат · не обчислено IBSolver';
    $('result-state').classList.remove('stale');
    $('statistics').hidden = $('statistics-title').hidden = true;
    renderWorkbookResults(null);
    fillMetadata($('summary'), [['Назва', result.name], ...externalMetadata(result)]);
}
function fillMetadata(element, rows) {
    element.replaceChildren();
    for (const [key, value] of rows) {
        const dt = document.createElement('dt'), dd = document.createElement('dd');
        dt.textContent = key; dd.textContent = value; element.append(dt, dd);
    }
}
function externalDialog(result, preview) {
    $('external-dialog-title').textContent = preview ? `Імпорт до «${state.active.name}»` : result.name;
    fillMetadata($('external-details'), externalMetadata(result));
    $('external-name').value = result.name.slice(0, 120);
    $('external-name').readOnly = !preview;
    $('external-confirm').hidden = !preview;
    const dialog = $('external-dialog');
    dialog.returnValue = 'cancel';
    const closed = new Promise(resolve => dialog.addEventListener('close', () => resolve(dialog.returnValue), {once: true}));
    dialog.showModal();
    return closed;
}
$('external-results').addEventListener('change', event => {
    const id = Number(event.target.value);
    operation(async () => {
        if (id) showExternal(await OpenExternalResult(state.active.id, id));
        else { await flushForTransition(); display(await LoadState()); }
    }).finally(() => { $('external-results').value = externalChart ? String(externalChart.id) : ''; });
});
$('external-metadata').addEventListener('click', () => operation(() => externalDialog(externalChart, false)));
$('external-delete').addEventListener('click', () => operation(async () => {
    const id = externalChart.id;
    await flushForTransition();
    const next = await DeleteExternalResult(state.active.id, id);
    if (next) { display(next); status('Зовнішній результат видалено.'); }
}));
$('import-csv').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    const preview = await PreviewExternalImport(state.active.id);
    if (!preview) return;
    try {
        if (await externalDialog(preview, true) !== 'save') { status('Імпорт скасовано.'); return; }
        const result = await SaveExternalImport(state.active.id, $('external-name').value);
        state.externalResults ||= [];
        state.externalResults.unshift(result);
        renderHistory(); showExternal(result);
        status('Зовнішній результат збережено в SQLite.');
    } finally { await CancelExternalImport(); }
}));

import './style.css';
import {LoadState, SaveConfiguration, SaveSettings, CreateDemo} from '../wailsjs/go/main/App';
import {TimeSeriesChart} from './chart';

document.querySelector('#app').innerHTML = `
<header><div class="brand">IB<span>Solver</span><small>ІНЖЕНЕРНИЙ РОБОЧИЙ ПРОСТІР</small></div>
    <span class="stage">Етап 1 · Основа застосунку</span><button id="reload">Відновити з БД</button></header>
<section class="parameters" aria-label="Параметри конфігурації">
    <div class="configuration-row"><label>Конфігурація<select id="configurations"></select></label>
        <label class="name-field">Назва<input id="name" maxlength="120" required></label><button id="new">Нова</button>
        <span id="save-state" role="status"></span></div>
    <form id="inputs"><label>Тривалість, с<input id="duration" type="number" min="0.000001" max="10000" step="any" required></label>
        <label>Кількість точок<input id="samples" type="number" min="2" max="20000" step="1" required></label>
        <label>Амплітуда, ум. од.<input id="amplitude" type="number" min="0.000001" max="1000000" step="any" required></label>
        <label>Частота, Гц<input id="frequency" type="number" min="0.000001" max="1000" step="any" required></label>
        <button type="submit" class="primary">Зберегти</button><button id="calculate" type="button" disabled title="Чисельне ядро буде реалізовано на наступних етапах">Розрахувати</button>
        <button id="demo" type="button">Створити DEMO</button></form>
    <p class="hint">Параметри лише для демонстрації графіків. Фізичну модель ще не реалізовано.</p>
</section>
<main><aside id="results"><div class="section-title">Результати <span class="badge">DEMO</span></div>
    <p id="result-state"></p><dl id="summary"></dl><div class="section-title curves-title">Криві та масштаби</div>
    <div id="legend"></div><p class="hint">Нормалізована шкала −1…1.<br>Кожна серія має власний масштаб.</p></aside>
    <div id="splitter" role="separator" aria-label="Ширина результатів" aria-orientation="vertical" aria-valuemin="180" aria-valuemax="480" tabindex="0"></div>
    <section class="chart-workspace"><div class="chart-title"><span>Часові ряди</span><span id="chart-label">Збережених даних немає</span></div>
        <div class="canvas-wrap"><canvas id="chart" aria-label="Графік часових рядів"></canvas></div><div id="cursor" aria-live="off"></div></section></main>
<footer><span id="status" role="status">Відкриття бази даних…</span><span>SQLite · локальне збереження</span></footer>`;

const $ = id => document.getElementById(id);
let state, busy = false, dirty = false, revision = 0;
let draftQueue = Promise.resolve();
const chart = new TimeSeriesChart($('chart'), $('legend'), $('cursor'), () => persistSettings());

function status(message, error = false) { $('status').textContent = message; $('status').classList.toggle('error', error); }
function setBusy(value) {
    busy = value;
    document.querySelectorAll('button, select, .parameters input, #legend input').forEach(el => { el.disabled = value || el.id === 'calculate'; });
    $('splitter').setAttribute('aria-disabled', String(value));
}
async function operation(action) {
    if (busy) return;
    setBusy(true);
    try { await draftQueue; await action(); } catch (error) {
        if (state) $('configurations').value = String(state.active.id);
        status(String(error), true);
    }
    finally { setBusy(false); }
}
function width(value) {
    state.settings.resultsWidth = Math.max(180, Math.min(480, value));
    document.querySelector('main').style.setProperty('--results-width', `${state.settings.resultsWidth}px`);
    $('splitter').setAttribute('aria-valuenow', state.settings.resultsWidth);
}
function readConfiguration() {
    const name = $('name').value.trim();
    const inputs = Object.fromEntries(['duration', 'samples', 'amplitude', 'frequency'].map(key => [key, $(key).valueAsNumber]));
    if (!name || [...name].length > 120) throw new Error('Вкажіть назву конфігурації (до 120 символів).');
    for (const [key, value] of Object.entries(inputs)) {
        const field = $(key);
        if (!Number.isFinite(value) || value < Number(field.min) || value > Number(field.max) || (key === 'samples' && !Number.isInteger(value))) {
            throw new Error('Перевірте параметри конфігурації. Незавершене введення ще не збережено.');
        }
    }
    return {...state.active, name, inputs};
}
function display(next) {
    state = next; dirty = false;
    $('configurations').replaceChildren(...state.configurations.map(c => new Option(c.name, c.id, false, c.id === state.active.id)));
    $('name').value = state.active.name;
    for (const [key, value] of Object.entries(state.active.inputs)) $(key).value = value;
    $('save-state').textContent = 'Збережено';
    width(state.settings.resultsWidth);
    const run = state.run;
    $('result-state').textContent = run ? (state.stale ? 'Попередній запуск · параметри змінено' : 'Збережені демонстраційні дані') : 'Результатів ще немає';
    $('result-state').classList.toggle('stale', state.stale);
    $('chart-label').textContent = run ? `DEMO · запуск №${run.id}` : 'Збережених даних немає';
    $('summary').replaceChildren();
    if (run) {
        for (const [key, value] of [['Точок / серію', run.summary.samples], ['Тривалість', `${run.summary.duration} с`], ['Серій', run.chart.series.length], ['Збережено', new Date(run.createdAt).toLocaleString('uk-UA')]]) {
            const term = document.createElement('dt'), description = document.createElement('dd'); term.textContent = key; description.textContent = value; $('summary').append(term, description);
        }
    }
    chart.set(run?.chart || null, state.settings);
    status(state.warning || 'Робочий стан відновлено з бази даних.');
}
async function save() {
    const configuration = readConfiguration();
    display(await SaveConfiguration(configuration, state.settings));
    status('Конфігурацію збережено.');
}
function markDirty() {
    dirty = true; revision++; $('save-state').textContent = 'Незбережені зміни';
    if (state?.run) { $('result-state').textContent = 'Попередній запуск · параметри редагуються'; $('result-state').classList.add('stale'); }
}
// Serialize valid input snapshots; delayed responses must never overwrite newer typing.
function autosave() {
    if (busy || !state) return;
    let configuration;
    try { configuration = readConfiguration(); } catch (error) { status(String(error), true); return; }
    const version = revision;
    const settings = structuredClone(state.settings);
    draftQueue = draftQueue.then(async () => {
        const next = await SaveConfiguration(configuration, settings);
        if (version !== revision) return;
        state = {...next, settings: state.settings}; dirty = false;
        $('save-state').textContent = 'Збережено';
        const option = [...$('configurations').options].find(item => Number(item.value) === state.active.id);
        if (option) option.textContent = state.active.name;
        $('result-state').textContent = state.run ? (state.stale ? 'Попередній запуск · параметри змінено' : 'Збережені демонстраційні дані') : 'Результатів ще немає';
        $('result-state').classList.toggle('stale', state.stale);
        status(state.warning || 'Конфігурацію збережено.');
    }).catch(error => status(String(error), true));
}
for (const id of ['name', 'duration', 'samples', 'amplitude', 'frequency']) {
    $(id).addEventListener('input', () => { markDirty(); autosave(); });
}
$('inputs').addEventListener('submit', event => { event.preventDefault(); operation(save); });
$('configurations').addEventListener('change', event => {
    const id = Number(event.target.value);
    operation(async () => {
        if (dirty) await save();
        display(await SaveSettings({...state.settings, activeConfigurationId: id}));
    });
});
$('new').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    display(await SaveConfiguration({id: 0, name: `Конфігурація ${state.configurations.length + 1}`, inputs: state.active.inputs}, state.settings));
    $('name').focus(); $('name').select();
}));
$('demo').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    display(await CreateDemo(state.active.id)); status('DEMO створено та збережено. Це не фізичний розрахунок.');
}));
$('reload').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    display(await LoadState());
}));
function persistSettings() {
    if (!state || busy) return;
    operation(async () => {
        await SaveSettings(state.settings);
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
operation(async () => display(await LoadState()));

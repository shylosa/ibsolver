import './style.css';
import {LoadState, SaveConfiguration, SaveSettings, CreateDemo, Calculate, CancelCalculation, ImportConfiguration, ExportConfiguration, ExportCSV, CloseWindow, PreviewExternalImport, CancelExternalImport, SaveExternalImport, OpenExternalResult, DeleteExternalResult, ExportExternalCSV} from '../wailsjs/go/main/App';
import {TimeSeriesChart} from './chart';
import {EventsOn} from '../wailsjs/runtime/runtime';
import {preparationFields, parsePreparation, displayPreparation} from './preparation.mjs';
import {externalMetadata} from './external.mjs';
import {DebouncedTask} from './debounce.mjs';

document.querySelector('#app').innerHTML = `
<header><div class="brand">IB<span>Solver</span><small>ІНЖЕНЕРНИЙ РОБОЧИЙ ПРОСТІР</small></div>
    <span class="stage">Підготовчі дані</span><button id="reload">Відновити з БД</button></header>
<section class="parameters" aria-label="Параметри конфігурації">
    <div class="configuration-row"><label>Конфігурація<select id="configurations"></select></label>
        <label>Режим<select id="mode"><option value="preparation">Підготовчі дані</option><option value="oscillator">Осцилятор · RK45</option><option value="demo">DEMO · інтерфейс</option></select></label>
        <label class="name-field">Назва<input id="name" maxlength="120" required></label><button id="new">Нова</button>
        <button id="import-json">Імпорт JSON</button><button id="export-json">Експорт JSON</button><span id="save-state" role="status"></span></div>
    <form id="inputs" novalidate><div id="preparation-fields">
        <div class="group-title">Вихідні дані</div><div id="preparation-main" class="preparation-grid"></div>
        <details id="coefficients"><summary>Вибрані коефіцієнти · одиниці не встановлено</summary><div id="preparation-coefficients" class="preparation-grid coefficients"></div></details>
        <p class="preparation-note">Предметні межі й обов’язковість не встановлено. Значення не пройшли фізичної перевірки.</p>
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
    <div id="demo-fields" hidden><div class="demo-fields"><label>Тривалість, с<input id="duration" type="number" min="0.000001" max="10000" step="any" required></label>
        <label>Кількість точок<input id="samples" type="number" min="2" max="20000" step="1" required></label>
        <label>Амплітуда, ум. од.<input id="amplitude" type="number" min="0.000001" max="1000000" step="any" required></label>
        <label>Частота, Гц<input id="frequency" type="number" min="0.000001" max="1000" step="any" required></label>
        </div><p class="hint">DEMO: математичні криві для перевірки інтерфейсу, не фізичний розрахунок.</p></div>
    <div class="actions"><button id="save" type="submit" disabled>Зберегти</button><button id="calculate" type="button" class="primary">Розрахувати</button>
        <button id="demo" type="button" class="primary" hidden>Створити DEMO</button><span id="preparation-explanation">Розрахунковий модуль ще не реалізовано.</span><button id="cancel" type="button" disabled>Скасувати</button></div></form>
</section>
<main><aside id="results"><div class="section-title">Результати <span id="result-kind" class="badge">RK45</span></div>
    <label class="history">Зовнішні результати<select id="external-results" aria-label="Збережені зовнішні результати"></select></label><div class="history-actions"><button id="external-metadata">Метадані</button><button id="external-delete">Видалити…</button></div>
    <p id="result-state"></p><dl id="summary"></dl><h3 id="statistics-title" hidden>Інтегратор RK45</h3><dl id="statistics" hidden></dl></aside>
    <div id="splitter" role="separator" aria-label="Ширина результатів" aria-orientation="vertical" aria-valuemin="180" aria-valuemax="480" tabindex="0"></div>
    <section class="chart-workspace"><div class="chart-title"><span>Часові ряди</span><span id="chart-label">Збережених даних немає</span><button id="chart-reset" title="Колесо — масштаб; перетягування — зсув; подвійний клік — увесь діапазон">Увесь графік</button><button id="import-csv">Імпорт CSV</button><button id="export-csv" disabled>CSV</button></div>
        <div id="legend" aria-label="Видимі ряди"></div>
        <div class="canvas-wrap"><canvas id="chart" aria-label="Спільне поле нормалізованих часових рядів"></canvas></div><div id="cursor" aria-live="off"></div></section></main>
<dialog id="discard-dialog"><p>Не вдалося зберегти поточну чернетку. Поверніться до редагування або явно відкиньте незбережені зміни.</p><form method="dialog"><button value="return" autofocus>Повернутися до редагування</button><button value="discard">Відкинути незбережені зміни</button></form></dialog>
<dialog id="external-dialog"><h3 id="external-dialog-title"></h3><p>Числові часові ряди з CSV. Фізичний зміст визначає джерело даних.</p><dl id="external-details"></dl><form method="dialog"><label>Назва<input id="external-name" maxlength="120" required></label><div class="actions"><button id="external-confirm" value="save">Зберегти результат</button><button value="cancel" formnovalidate autofocus>Закрити / скасувати</button></div></form></dialog>
<footer><span id="status" role="status">Відкриття бази даних…</span><span>SQLite · локальне збереження</span></footer>`;

const $ = id => document.getElementById(id);
for (const [key, label, unit] of preparationFields) {
    const element = document.createElement('label');
    element.className = unit ? '' : 'unknown-unit';
    element.innerHTML = `${label}<span class="unit">${unit || 'Одиницю не встановлено'}</span><input id="p-${key}" type="text" inputmode="decimal" placeholder="Не задано" aria-describedby="p-${key}-message"><span id="p-${key}-message" class="field-message">Не задано</span>`;
    $(unit ? 'preparation-main' : 'preparation-coefficients').append(element);
}
$('p-d').closest('label').insertAdjacentHTML('beforeend', '<details class="field-help"><summary>Про одиниці d</summary>Введення в мм; збереження в метрах (мм / 1000). Числова точність — binary64.</details>');
let state, busy = false, dirty = false, revision = 0;
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
    document.querySelectorAll('button, select, .parameters input, #legend input').forEach(el => { if (!el.closest('dialog')) el.disabled = value || el.id === 'cancel'; });
    $('splitter').setAttribute('aria-disabled', String(value));
    $('export-csv').disabled = value || !(visibleRun() || externalChart);
    $('external-metadata').disabled = $('external-delete').disabled = value || !externalChart;
    $('chart-reset').disabled = value || !(visibleRun() || externalChart);
    $('save').disabled = value || !dirty;
    if (state) updateMode();
}
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
    const name = $('name').value.trim();
    if (!name || [...name].length > 120) throw new Error('Вкажіть назву конфігурації (до 120 символів).');
    const parameters = structuredClone(state.active.parameters);
    const mode = state.active.mode;
    if (mode === 'demo') {
        parameters.demo = Object.fromEntries(['duration', 'samples', 'amplitude', 'frequency'].map(key => {
            const field = $(key), value = field.valueAsNumber;
            if (!validNumber(field, value)) throw new Error('Перевірте параметри DEMO.');
            return [key, value];
        }));
    } else if (mode === 'preparation') {
        const values = {}, errors = [];
        for (const [key] of preparationFields) {
            try { values[key] = readPreparationField(key); } catch (error) { errors.push([key, error]); }
        }
        if (errors.length) {
            const [key, error] = errors[0];
            if (['ns', 'f1', 'af'].includes(key)) $('coefficients').open = true;
            throw error;
        }
        parameters.preparation = values;
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
    return {...state.active, name, parameters};
}
function updateMode() {
    const mode = state.active.mode, numerical = mode === 'oscillator', preparation = mode === 'preparation';
    $('calculate').hidden = mode === 'demo';
    $('calculate').disabled = busy;
    $('preparation-explanation').hidden = true;
    $('demo').hidden = mode !== 'demo';
    $('cancel').hidden = !(numerical || preparation);
    for (const [id, active] of [['oscillator-fields', numerical], ['demo-fields', mode === 'demo'], ['preparation-fields', preparation]]) {
        $(id).hidden = !active;
        $(id).querySelectorAll('input').forEach(field => { field.disabled = busy || !active; });
    }
}
function readPreparationField(key) {
    const field = $('p-' + key), message = $('p-' + key + '-message');
    try {
        const stored = state?.active.parameters.preparation?.[key];
        const value = stored != null && field.value === displayPreparation(stored, key) ? stored : parsePreparation(field.value, key);
        field.setAttribute('aria-invalid', 'false');
        message.classList.remove('error');
        message.textContent = value === null ? 'Не задано' : 'Число введено';
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
    return state.run ? (stale ? 'Попередній запуск · параметри змінено' : state.run.summary.kind === 'BALLISTICS' ? 'Збережений балістичний розрахунок' : state.run.summary.kind === 'NUMERICAL' ? 'Збережений чисельний результат' : 'Збережені демонстраційні дані') : 'Результатів ще немає';
}
function display(next) {
    state = next; dirty = false; externalChart = null; revision++;
    document.querySelectorAll('.parameters input').forEach(field => { field.removeAttribute('aria-invalid'); field.title = ''; });
    $('configurations').replaceChildren(...state.configurations.map(c => new Option(c.name, c.id, false, c.id === state.active.id)));
    $('name').value = state.active.name;
    $('mode').value = state.active.mode;
    updateMode();
    for (const [key, value] of Object.entries(state.active.parameters.demo || {})) $(key).value = value;
    for (const [group, prefix] of [['model', 'n-'], ['solver', 's-']]) {
        for (const [key, value] of Object.entries(state.active.parameters.oscillator?.[group] || {})) $(prefix + key).value = value;
    }
    for (const [key] of preparationFields) {
        $('p-' + key).value = displayPreparation(state.active.parameters.preparation?.[key], key);
        readPreparationField(key);
    }
    $('save-state').textContent = 'Збережено';
    width(state.settings.resultsWidth);
    renderHistory();
    const run = visibleRun();
    $('result-state').textContent = resultState();
    $('result-state').classList.toggle('stale', state.stale);
    const numerical = run?.summary.kind === 'NUMERICAL';
    const ballistics = run?.summary.kind === 'BALLISTICS';
    $('result-kind').textContent = state.active.mode === 'preparation' ? (ballistics ? 'БАЛІСТИКА' : 'ДАНІ') : numerical || !run ? 'RK45' : 'DEMO';
    $('chart-label').textContent = run ? `${ballistics ? 'Внутрішня балістика · OZVB' : numerical ? 'Чисельний приклад · RK45' : 'DEMO'} · запуск №${run.id}` : 'Збережених даних немає';
    $('summary').replaceChildren();
    $('statistics').replaceChildren();
    $('statistics').hidden = $('statistics-title').hidden = !numerical;
    if (run) {
        const durationFormatted = run.summary.duration < 0.1 ? `${(run.summary.duration * 1000).toPrecision(4)} мс` : `${run.summary.duration.toPrecision(6)} с`;
        const rows = [[run.summary.configurationName ? 'Назва запуску' : 'Поточна назва', run.summary.configurationName || state.active.name], ['Режим запуску', run.mode === 'preparation' ? 'Балістика (OZVB)' : run.mode === 'oscillator' ? 'Осцилятор' : 'DEMO'], ['Точок / серію', run.summary.samples], ['Тривалість', durationFormatted], ['Серій', run.chart.series.length], ['Збережено', new Date(run.createdAt).toLocaleString('uk-UA')]];
        if (ballistics && run.summary.ballistics) {
            const b = run.summary.ballistics;
            rows.push(
                ['Макс. тиск Pm', `${b.maxPressure.toFixed(1)} МПа`],
                ['Дульна швидкість Vd', `${b.muzzleVelocity.toFixed(1)} м/с`],
                ['Час руху снаряда td', `${(b.muzzleTime * 1000).toFixed(2)} мс`],
                ['Шлях снаряда Ld', `${b.barrelTravel.toFixed(3)} м`],
                ['Маса заряду ω', `${b.chargeMass.toFixed(4)} кг`],
                ['Обʼєм камори W₀', `${(b.chamberVolume * 1000).toFixed(3)} л`],
                ['Положення кінця горіння ηk', `${b.burntFraction.toFixed(3)}`],
                ['Розрахункова маса ствола', `${b.barrelMass.toFixed(1)} кг`],
            );
        } else if (numerical) {
            const n = run.summary.numerical, snapshot = run.snapshot.oscillator;
            rows.push(['x(T)', `${n.finalPosition.toPrecision(6)} м`], ['v(T)', `${n.finalVelocity.toPrecision(6)} м/с`], ['E(0)', `${n.initialEnergy.toPrecision(6)} Дж`], ['E(T)', `${n.finalEnergy.toPrecision(6)} Дж`], ['Кроки / відхилено', `${n.statistics.accepted} / ${n.statistics.rejected}`], ['Обчислень RHS', n.statistics.evaluations], ['Допуски абс. / відн.', `${snapshot.solver.absTolerance} / ${snapshot.solver.relTolerance}`], ['Знімок m / k / c', `${snapshot.model.mass} кг / ${snapshot.model.stiffness} Н/м / ${snapshot.model.damping} кг/с`], ['Знімок x₀ / v₀', `${snapshot.model.position} м / ${snapshot.model.velocity} м/с`]);
        }
        for (const [key, value] of rows) {
            const term = document.createElement('dt'), description = document.createElement('dd'); term.textContent = key; description.textContent = value; (['Кроки / відхилено', 'Обчислень RHS', 'Допуски абс. / відн.'].includes(key) ? $('statistics') : $('summary')).append(term, description);
        }
    }
    chart.set(run?.chart || null, state.settings);
    status(state.warning || 'Робочий стан відновлено з бази даних.');
}
async function save() {
    delayedSave.cancel();
    await draftQueue;
    if (!dirty) return;
    try { await queueSave(); } catch (error) { focusInvalid(); throw error; }
    status('Конфігурацію збережено.');
}
function focusInvalid() {
    const field = document.querySelector('.parameters input[aria-invalid="true"]');
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
    if (!externalChart && visibleRun() && field.id !== 'name') {
        let changed = true;
        try {
            const mode = state.active.mode;
            changed = JSON.stringify(readConfiguration().parameters[mode]) !== JSON.stringify(state.run.snapshot[mode]);
        } catch {}
        $('result-state').textContent = changed ? 'Попередній запуск · параметри редагуються' : resultState(false);
        $('result-state').classList.toggle('stale', changed);
    }
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
        const option = [...$('configurations').options].find(item => Number(item.value) === state.active.id);
        if (option) option.textContent = state.active.name;
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
for (const field of document.querySelectorAll('.parameters input')) {
    field.addEventListener('input', () => {
        markDirty(field);
        if (field.id.startsWith('p-')) { try { readPreparationField(field.id.slice(2)); } catch {} }
        else if (field.type === 'number') validNumber(field, field.valueAsNumber);
        delayedSave.schedule();
    });
}
$('name').addEventListener('keydown', event => {
    if (event.key === 'Enter') { event.preventDefault(); $('inputs').requestSubmit(); }
});
for (const [key] of preparationFields) $('p-' + key).addEventListener('blur', () => { try { readPreparationField(key); } catch {} });
$('chart-reset').addEventListener('click', () => chart.reset());
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
        if (mode === 'preparation') parameters.preparation ||= Object.fromEntries(preparationFields.map(([key]) => [key, null]));
        display(await SaveConfiguration({...state.active, mode, parameters}, state.settings));
        status('Режим змінено. Параметри режимів збережено.');
    }).finally(() => { $('mode').value = state.active.mode; });
});
$('new').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    display(await SaveConfiguration({id: 0, name: `Конфігурація ${state.configurations.length + 1}`, mode: 'preparation', parameters: {...state.active.parameters, preparation: Object.fromEntries(preparationFields.map(([key]) => [key, null]))}}, state.settings));
    $('name').focus(); $('name').select();
}));
$('demo').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    display(await CreateDemo(state.active.id)); status('DEMO створено та збережено. Це не фізичний розрахунок.');
}));
$('calculate').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    status('Виконується чисельне інтегрування RK45…');
    $('calculate').textContent = 'Обчислення…';
    $('cancel').disabled = false;
    try { display(await Calculate(state.active.id)); status('Чисельний приклад розраховано та збережено.'); }
    finally { $('calculate').textContent = 'Розрахувати'; $('cancel').disabled = true; }
}));
$('cancel').addEventListener('click', async () => {
    $('cancel').disabled = true;
    try { await CancelCalculation(); status('Запит на скасування надіслано.'); }
    catch (error) { status(String(error), true); }
});
$('reload').addEventListener('click', () => operation(async () => {
    await flushForTransition();
    display(await LoadState());
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
operation(async () => display(await LoadState()));

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
    $('chart-label').textContent = `Зовнішній CSV · ${result.name}`;
    $('statistics').hidden = $('statistics-title').hidden = true;
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

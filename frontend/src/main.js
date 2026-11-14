import './style.css';
import {LoadState, SaveConfiguration, SaveSettings, CreateDemo, Calculate, CancelCalculation} from '../wailsjs/go/main/App';
import {TimeSeriesChart} from './chart';

document.querySelector('#app').innerHTML = `
<header><div class="brand">IB<span>Solver</span><small>ІНЖЕНЕРНИЙ РОБОЧИЙ ПРОСТІР</small></div>
    <span class="stage">Етап 3 · Інженерні часові ряди</span><button id="reload">Відновити з БД</button></header>
<section class="parameters" aria-label="Параметри конфігурації">
    <div class="configuration-row"><label>Конфігурація<select id="configurations"></select></label>
        <label>Режим<select id="mode"><option value="oscillator">Осцилятор · RK45</option><option value="demo">DEMO · інтерфейс</option></select></label>
        <label class="name-field">Назва<input id="name" maxlength="120" required></label><button id="new">Нова</button>
        <span id="save-state" role="status"></span></div>
    <form id="inputs"><div id="oscillator-fields"><div class="numerical-fields">
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
    <div class="actions"><button type="submit">Зберегти</button><button id="calculate" type="button" class="primary">Розрахувати</button>
        <button id="demo" type="button" class="primary" hidden>Створити DEMO</button><button id="cancel" type="button" disabled>Скасувати</button></div></form>
</section>
<main><aside id="results"><div class="section-title">Результати <span id="result-kind" class="badge">RK45</span></div>
    <p id="result-state"></p><dl id="summary"></dl></aside>
    <div id="splitter" role="separator" aria-label="Ширина результатів" aria-orientation="vertical" aria-valuemin="180" aria-valuemax="480" tabindex="0"></div>
    <section class="chart-workspace"><div class="chart-title"><span>Часові ряди</span><span id="chart-label">Збережених даних немає</span></div>
        <div id="legend" aria-label="Видимі ряди"></div>
        <div class="canvas-wrap"><canvas id="chart" aria-label="Спільне поле нормалізованих часових рядів"></canvas></div><div id="cursor" aria-live="off"></div></section></main>
<footer><span id="status" role="status">Відкриття бази даних…</span><span>SQLite · локальне збереження</span></footer>`;

const $ = id => document.getElementById(id);
let state, busy = false, dirty = false, revision = 0;
let draftQueue = Promise.resolve();
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
    document.querySelectorAll('button, select, .parameters input, #legend input').forEach(el => { el.disabled = value || el.id === 'cancel'; });
    $('splitter').setAttribute('aria-disabled', String(value));
    if (state) updateMode();
}
async function operation(action) {
    if (busy) return;
    setBusy(true);
    try { await draftQueue; await action(); } catch (error) {
        if (state) { $('configurations').value = String(state.active.id); $('mode').value = state.active.mode; }
        status(errorMessage(error), true);
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
    if (!name || [...name].length > 120) throw new Error('Вкажіть назву конфігурації (до 120 символів).');
    const parameters = structuredClone(state.active.parameters);
    const mode = state.active.mode;
    if (mode === 'demo') {
        parameters.demo = Object.fromEntries(['duration', 'samples', 'amplitude', 'frequency'].map(key => {
            const field = $(key), value = field.valueAsNumber;
            if (!validNumber(field, value)) throw new Error('Перевірте параметри DEMO.');
            return [key, value];
        }));
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
    const numerical = state.active.mode === 'oscillator';
    $('oscillator-fields').hidden = !numerical;
    $('demo-fields').hidden = numerical;
    $('calculate').hidden = !numerical;
    $('demo').hidden = numerical;
    $('cancel').hidden = !numerical;
    for (const [id, active] of [['oscillator-fields', numerical], ['demo-fields', !numerical]]) {
        $(id).querySelectorAll('input').forEach(field => { field.disabled = busy || !active; });
    }
}
// Explicit checks also work while operation() has disabled form controls.
function validNumber(field, value) {
    return Number.isFinite(value) && (!field.min || value >= Number(field.min)) && (!field.max || value <= Number(field.max)) && (field.step !== '1' || Number.isInteger(value));
}
function resultState() {
    return state.run ? (state.stale ? 'Попередній запуск · параметри змінено' : state.run.summary.kind === 'NUMERICAL' ? 'Збережений чисельний результат' : 'Збережені демонстраційні дані') : 'Результатів ще немає';
}
function display(next) {
    state = next; dirty = false;
    $('configurations').replaceChildren(...state.configurations.map(c => new Option(c.name, c.id, false, c.id === state.active.id)));
    $('name').value = state.active.name;
    $('mode').value = state.active.mode;
    updateMode();
    for (const [key, value] of Object.entries(state.active.parameters.demo)) $(key).value = value;
    for (const [group, prefix] of [['model', 'n-'], ['solver', 's-']]) {
        for (const [key, value] of Object.entries(state.active.parameters.oscillator[group])) $(prefix + key).value = value;
    }
    $('save-state').textContent = 'Збережено';
    width(state.settings.resultsWidth);
    const run = state.run;
    $('result-state').textContent = resultState();
    $('result-state').classList.toggle('stale', state.stale);
    const numerical = run?.summary.kind === 'NUMERICAL';
    $('result-kind').textContent = numerical || !run ? 'RK45' : 'DEMO';
    $('chart-label').textContent = run ? `${numerical ? 'Чисельний приклад · RK45' : 'DEMO'} · запуск №${run.id}` : 'Збережених даних немає';
    $('summary').replaceChildren();
    if (run) {
        const rows = [['Точок / серію', run.summary.samples], ['Тривалість', `${run.summary.duration} с`], ['Серій', run.chart.series.length], ['Збережено', new Date(run.createdAt).toLocaleString('uk-UA')]];
        if (numerical) {
            const n = run.summary.numerical, snapshot = run.snapshot.oscillator;
            rows.push(['x(T)', `${n.finalPosition.toPrecision(6)} м`], ['v(T)', `${n.finalVelocity.toPrecision(6)} м/с`], ['E(0)', `${n.initialEnergy.toPrecision(6)} Дж`], ['E(T)', `${n.finalEnergy.toPrecision(6)} Дж`], ['Кроки / відхилено', `${n.statistics.accepted} / ${n.statistics.rejected}`], ['Обчислень RHS', n.statistics.evaluations], ['Допуски абс. / відн.', `${snapshot.solver.absTolerance} / ${snapshot.solver.relTolerance}`], ['Знімок m / k / c', `${snapshot.model.mass} кг / ${snapshot.model.stiffness} Н/м / ${snapshot.model.damping} кг/с`], ['Знімок x₀ / v₀', `${snapshot.model.position} м / ${snapshot.model.velocity} м/с`]);
        }
        for (const [key, value] of rows) {
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
        $('result-state').textContent = resultState();
        $('result-state').classList.toggle('stale', state.stale);
        status(state.warning || 'Конфігурацію збережено.');
    }).catch(error => status(errorMessage(error), true));
}
for (const field of document.querySelectorAll('.parameters input')) {
    field.addEventListener('input', () => { markDirty(); autosave(); });
}
$('inputs').addEventListener('submit', event => { event.preventDefault(); operation(save); });
$('configurations').addEventListener('change', event => {
    const id = Number(event.target.value);
    operation(async () => {
        if (dirty) await save();
        display(await SaveSettings({...state.settings, activeConfigurationId: id}));
    });
});
$('mode').addEventListener('change', event => {
    const mode = event.target.value;
    operation(async () => {
        if (dirty) await save();
        display(await SaveConfiguration({...state.active, mode}, state.settings));
        status('Режим змінено. Параметри обох режимів збережено.');
    }).finally(() => { $('mode').value = state.active.mode; });
});
$('new').addEventListener('click', () => operation(async () => {
    if (dirty) await save();
    display(await SaveConfiguration({id: 0, name: `Конфігурація ${state.configurations.length + 1}`, mode: state.active.mode, parameters: state.active.parameters}, state.settings));
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

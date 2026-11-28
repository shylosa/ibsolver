const preparationFieldDefinitions = [
    {key: 'pmax', technicalId: 'PMAX', label: 'Максимальний тиск у стволі', unit: 'МПа', source: '2A65.BAS, рядки 70, 295'},
    {key: 'vd', technicalId: 'VDZ', label: 'Задана дульна швидкість снаряда', unit: 'м/с', source: '2A65.BAS, рядки 70, 296'},
    {key: 'd', technicalId: 'D', label: 'Калібр ствола', unit: 'мм', source: '2A65.BAS, рядки 80, 297'},
    {key: 'q', technicalId: 'Q', label: 'Маса снаряда', unit: 'кг', source: '2A65.BAS, рядки 80, 298'},
    {key: 'ns', technicalId: 'NS', label: 'Вплив нарізів на площу каналу', unit: null, description: 'Коефіцієнт, що враховує вплив нарізів на площу поперечного перерізу каналу ствола.', source: '2A65.BAS, рядки 80, 299'},
    {key: 'hi', technicalId: 'HI', label: 'Розширення камори', unit: null, source: '2A65.BAS, рядки 80, 300'},
    {key: 'af', technicalId: 'AFI', label: 'Постійний доданок коефіцієнта другорядних робіт', unit: null, source: '2A65.BAS, рядки 80, 301–302'},
    {key: 'pfor', technicalId: 'PFOR', label: 'Тиск форсування', unit: 'МПа', source: '2A65.BAS, рядки 80, 303'},
    {key: 'kt', technicalId: 'KT', label: 'Вплив початкової температури на горіння', unit: null, description: 'Емпіричний коефіцієнт, що враховує вплив початкової температури заряду на швидкість горіння.', control: 'select', options: [
        {value: 0.0025, label: 'Нітрогліцериновий порох'},
        {value: 0.0018, label: 'Піроксиліновий порох'},
    ], source: '2A65.BAS, рядки 90, 305–307; альтернативи в коментарі перед рядком 90'},
    {key: 'tn', technicalId: 'TN', label: 'Початкова температура заряду', unit: '°C', source: '2A65.BAS, рядки 90, 308'},
    {key: 'forma', technicalId: 'FORMA', label: 'Форма зерна пороху', unit: null, control: 'select', options: [
        {value: 1, label: 'Куб, шар, цилиндр'},
        {value: 2, label: 'Кв.бр, кр.пр.'},
        {value: 3, label: 'Кв. пласт.'},
        {value: 4, label: 'Лента'},
        {value: 5, label: 'Тр. кор.'},
        {value: 6, label: 'Трубка'},
        {value: 7, label: '7-ми канальн.'},
    ], source: '2A65.BAS, рядки 90, 309; форми 1–7'},
    {key: 'omega', technicalId: 'OMEGA', label: 'Маса заряду', unit: 'кг', source: '2A65.BAS, рядки 90, 310'},
    {key: 'w0', technicalId: 'W0', label: 'Геометричний обʼєм камори', unit: 'м³', source: '2A65.BAS, рядки 70, 90, 311'},
    {key: 'ro', technicalId: 'RO', label: 'Густина пороху', unit: 'кг/м³', source: '2A65.BAS, рядки 100, 312'},
    {key: 'tg15', technicalId: 'TG15', label: 'Температура горіння за +15 °C', unit: 'К', description: 'Температура горіння за сталого обʼєму.', source: '2A65.BAS, рядки 100, 313–314'},
    {key: 'ku1', technicalId: 'KU1', label: 'Зміна швидкості горіння під час вигоряння', unit: null, description: 'Коефіцієнт, що враховує збільшення швидкості горіння флегматизованого пороху в міру вигоряння зерна.', source: '2A65.BAS, рядки 100, 315–316'},
    {key: 'f1', technicalId: 'F15', label: 'Сила пороху за +15 °C', unit: 'МДж/кг', source: '2A65.BAS, рядки 100, 317'},
    {key: 'u15', technicalId: 'U15', label: 'Швидкість горіння за +15 °C', unit: 'м/(с·Па)', description: 'Коефіцієнт швидкості горіння пороху.', source: '2A65.BAS, рядки 100, 318–319'},
    {key: 'svod', technicalId: 'SVOD', label: 'Товщина горючого своду', unit: 'мм', description: 'Товщина горючого своду зерна (2 × e1).', source: '2A65.BAS, рядки 100, 320'},
    {key: 'pvos', technicalId: 'PVOS', label: 'Тиск газів запалювача', unit: 'МПа', source: '2A65.BAS, рядки 105, 321'},
    {key: 'tvos', technicalId: 'TVOS', label: 'Температура газів запалювача', unit: 'К', source: '2A65.BAS, рядки 105, 322'},
    {key: 'lc', technicalId: 'LS', label: 'Загальна довжина ствола', unit: 'м', description: 'Сума довжини камори та шляху снаряда в каналі ствола.', source: '2A65.BAS, рядки 80, 304'},
];

export const preparationFields = preparationFieldDefinitions.map(field => ({control: 'text', ...field}));
export const formaOptions = preparationFields.find(field => field.key === 'forma').options;
export const ktOptions = preparationFields.find(field => field.key === 'kt').options;
export const KT_CUSTOM_VALUE = 'custom';

export const preparationResultPanel = {
    title: 'Результати підготовки',
};

export const workbookPreparationFields = [
    {key: 'd', label: 'd', unit: 'мм', help: 'Калібр (підпис C7; одиниця мм у E7).'},
    {key: 'q', label: 'q', unit: 'кг', help: 'Маса снаряда (підпис C8; одиниця кг у E8).'},
    {key: 'vd', label: 'Vд', unit: 'м/с', help: 'Дульна швидкість (підпис C9; одиниця м/с у E9).'},
    {key: 'lc', label: 'lc', unit: 'м', help: 'Довжина ствола (підпис C10; одиниця м у E10). Поле D10 збережено як у книзі, але збережені формули Excel його не читають.'},
    {key: 'ns', label: 'ns', unit: null, help: 'Коефіцієнт враховує вплив глибини нарізів. Одиницю вимірювання в книзі не вказано.'},
    {key: 'f1', label: 'f1', unit: null, help: 'Коефіцієнт враховує тертя та роботу обертання. Одиницю вимірювання в книзі не вказано.'},
    {key: 'af', label: 'af', unit: null, help: 'Використовується у визначенні f1; примітка до книги вказує на ручний вибір за Таблицею 2. Зв’язок таблиці з полем не автоматизовано. Одиницю вимірювання в книзі не вказано.'},
];

export function parseWorkbookPreparation(text) {
    const raw = text.trim();
    if (!raw) return null;
    if (!/^[+-]?(?:\d+(?:[.,]\d+)?|[.,]\d+)(?:[eE][+-]?\d+)?$/.test(raw)) {
        const unfinished = /^[+-]?$/.test(raw) || /^[+-]?(?:\d+[.,]|(?:\d+(?:[.,]\d+)?|[.,]\d+)[eE][+-]?)$/.test(raw);
        throw new Error(unfinished ? 'Завершіть число' : 'Введіть число без одиниць і розділювачів тисяч');
    }
    const value = Number(raw.replace(',', '.'));
    const nonzero = /[1-9]/.test(raw.split(/[eE]/)[0]);
    if (!Number.isFinite(value) || (nonzero && value === 0)) {
        throw new Error('Число поза діапазоном binary64');
    }
    return value;
}

// Only complete decimal numbers are accepted; raw text stays in the editor.
export function parsePreparation(text, key) {
    if (key === 'forma') {
        if (!text.trim()) return null;
        if (!/^[1-7]$/.test(text.trim())) throw new Error('Виберіть форму зерна від 1 до 7');
        return Number(text.trim());
    }
    const raw = text.trim();
    if (!raw) return null;
    if (!/^[+-]?(?:\d+(?:[.,]\d+)?|[.,]\d+)(?:[eE][+-]?\d+)?$/.test(raw)) {
        const unfinished = /^[+-]?$/.test(raw) || /^[+-]?(?:\d+[.,]|(?:\d+(?:[.,]\d+)?|[.,]\d+)[eE][+-]?)$/.test(raw);
        throw new Error(unfinished ? 'Завершіть число' : 'Введіть число без одиниць і розділювачів тисяч');
    }
    const value = Number(raw.replace(',', '.'));
    const nonzero = /[1-9]/.test(raw.split(/[eE]/)[0]);
    let stored = value;
    if (key === 'd') stored = value / 1000;
    if (key === 'svod') stored = value / 2000;
    if (key === 'pmax' || key === 'pfor' || key === 'pvos') stored = value * 1e6;
    if (key === 'f1') stored = value * 1e6;
    if (!Number.isFinite(value) || !Number.isFinite(stored) || (nonzero && stored === 0)) {
        throw new Error('Число поза діапазоном binary64');
    }
    return stored;
}

export function ktSelectValue(value) {
    if (value == null) return '';
    return ktOptions.find(option => option.value === value)?.value.toString() ?? KT_CUSTOM_VALUE;
}

export function parseKTSelection(selection, customText) {
    if (selection === '') return null;
    if (selection === KT_CUSTOM_VALUE) return parsePreparation(customText, 'kt');
    const option = ktOptions.find(item => String(item.value) === selection);
    if (!option) throw new Error('Виберіть варіант KT або введіть числове значення');
    return option.value;
}

// ".87" is accepted as input, but the editor always displays the leading zero.
export function normalizeLeadingZero(text) {
    return text.replace(/^([+-]?)\./, '$10.');
}

export function displayPreparation(value, key, inputFormat) {
    if (typeof inputFormat === 'string') {
        try {
            if (parsePreparation(inputFormat, key) === value) return normalizeLeadingZero(inputFormat);
        } catch {}
    }
    if (value == null) return '';
    if (key === 'forma') return String(value);
    // Avoid overflowing the display conversion for finite stored metres.
    if (key === 'd') {
        const mm = value * 1000;
        if (!Number.isFinite(mm)) return `${value}e3`.replace(/e([+-]?\d+)e3$/, (_, exponent) => `e${Number(exponent) + 3}`);
        return String(mm);
    }
    if (key === 'svod') return String(value * 2000);
    if (key === 'pmax' || key === 'pfor' || key === 'pvos') return String(value / 1e6);
    if (key === 'f1') return String(value / 1e6);
    if (key === 'w0') return value.toExponential();
    return String(value);
}

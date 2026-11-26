const preparationFieldDefinitions = [
    {key: 'pmax', technicalId: 'PMAX', label: 'Заданий максимальний тиск у стволі', unit: 'МПа', description: 'Граничне значення тиску, задане для ствола.', source: '2A65.BAS, рядки 70, 295'},
    {key: 'vd', technicalId: 'VDZ', label: 'Задана дульна швидкість снаряда', unit: 'м/с', description: 'Швидкість снаряда, яку задано для дульного зрізу.', source: '2A65.BAS, рядки 70, 296'},
    {key: 'd', technicalId: 'D', label: 'Калібр ствола', unit: 'мм', description: 'Діаметр каналу ствола. Значення вводиться в міліметрах.', source: '2A65.BAS, рядки 80, 297'},
    {key: 'q', technicalId: 'Q', label: 'Маса снаряда', unit: 'кг', description: 'Маса снаряда, що рухається каналом ствола.', source: '2A65.BAS, рядки 80, 298'},
    {key: 'ns', technicalId: 'NS', label: 'Вплив нарізів на площу каналу', unit: null, description: 'Коефіцієнт, що враховує вплив нарізів на площу поперечного перерізу каналу ствола.', source: '2A65.BAS, рядки 80, 299'},
    {key: 'hi', technicalId: 'HI', label: 'Розширення камори', unit: null, description: 'Коефіцієнт розширення камори.', source: '2A65.BAS, рядки 80, 300'},
    {key: 'af', technicalId: 'AFI', label: 'Постійний доданок коефіцієнта другорядних робіт', unit: null, description: 'Постійна частина коефіцієнта, який враховує другорядні роботи.', source: '2A65.BAS, рядки 80, 301–302'},
    {key: 'pfor', technicalId: 'PFOR', label: 'Тиск форсування', unit: 'МПа', description: 'Задане значення тиску форсування.', source: '2A65.BAS, рядки 80, 303'},
    {key: 'kt', technicalId: 'KT', label: 'Вплив початкової температури на горіння', unit: null, description: 'Емпіричний коефіцієнт, що враховує вплив початкової температури заряду на швидкість горіння. У BAS наведено різні типові значення для нітрогліцеринового та піроксилінового пороху; інші числові значення також можна ввести.', control: 'select', options: [
        {value: 0.0025, label: 'Нітрогліцериновий порох'},
        {value: 0.0018, label: 'Піроксиліновий порох'},
    ], source: '2A65.BAS, рядки 90, 305–307; альтернативи в коментарі перед рядком 90'},
    {key: 'tn', technicalId: 'TN', label: 'Початкова температура заряду', unit: '°C', description: 'Початкова температура заряду.', source: '2A65.BAS, рядки 90, 308'},
    {key: 'forma', technicalId: 'FORMA', label: 'Форма зерна пороху', unit: null, description: 'Номер однієї з семи форм порохового зерна. Варіанти відповідають наявній класифікації.', control: 'select', options: [
        {value: 1, label: 'Куб, шар, цилиндр'},
        {value: 2, label: 'Кв.бр, кр.пр.'},
        {value: 3, label: 'Кв. пласт.'},
        {value: 4, label: 'Лента'},
        {value: 5, label: 'Тр. кор.'},
        {value: 6, label: 'Трубка'},
        {value: 7, label: '7-ми канальн.'},
    ], source: '2A65.BAS, рядки 90, 309; форми 1–7'},
    {key: 'omega', technicalId: 'OMEGA', label: 'Маса заряду', unit: 'кг', description: 'Маса порохового заряду.', source: '2A65.BAS, рядки 90, 310'},
    {key: 'w0', technicalId: 'W0', label: 'Геометричний обʼєм камори', unit: 'м³', description: 'Геометричний обʼєм камори. Для початкових параметрів BAS зазначає основні одиниці SI, крім товщини своду.', source: '2A65.BAS, рядки 70, 90, 311'},
    {key: 'ro', technicalId: 'RO', label: 'Густина пороху', unit: 'кг/м³', description: 'Густина пороху.', source: '2A65.BAS, рядки 100, 312'},
    {key: 'tg15', technicalId: 'TG15', label: 'Температура горіння за +15 °C', unit: 'К', description: 'Температура горіння пороху за сталого обʼєму, коли початкова температура заряду становить +15 °C.', source: '2A65.BAS, рядки 100, 313–314'},
    {key: 'ku1', technicalId: 'KU1', label: 'Зміна швидкості горіння під час вигоряння', unit: null, description: 'Коефіцієнт, що враховує збільшення швидкості горіння флегматизованого пороху в міру вигоряння зерна.', source: '2A65.BAS, рядки 100, 315–316'},
    {key: 'f1', technicalId: 'F15', label: 'Сила пороху за +15 °C', unit: 'МДж/кг', description: 'Сила пороху за нормальної початкової температури заряду +15 °C.', source: '2A65.BAS, рядки 100, 317'},
    {key: 'u15', technicalId: 'U15', label: 'Швидкість горіння за +15 °C', unit: 'м/(с·Па)', description: 'Коефіцієнт швидкості горіння пороху за нормальної початкової температури +15 °C.', source: '2A65.BAS, рядки 100, 318–319'},
    {key: 'svod', technicalId: 'SVOD', label: 'Товщина горючого своду', unit: 'мм', description: 'Товщина горючого своду зерна (2 × e1). Вводиться в міліметрах.', source: '2A65.BAS, рядки 100, 320'},
    {key: 'pvos', technicalId: 'PVOS', label: 'Тиск газів запалювача', unit: 'МПа', description: 'Тиск газів запалювача.', source: '2A65.BAS, рядки 105, 321'},
    {key: 'tvos', technicalId: 'TVOS', label: 'Температура газів запалювача', unit: 'К', description: 'Температура газів запалювача.', source: '2A65.BAS, рядки 105, 322'},
    {key: 'lc', technicalId: 'LS', label: 'Загальна довжина ствола', unit: 'м', description: 'Сума довжини камори та шляху снаряда в каналі ствола.', source: '2A65.BAS, рядки 80, 304'},
];

export const preparationFields = preparationFieldDefinitions.map(field => ({control: 'text', ...field}));
export const formaOptions = preparationFields.find(field => field.key === 'forma').options;
export const ktOptions = preparationFields.find(field => field.key === 'kt').options;
export const KT_CUSTOM_VALUE = 'custom';

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

export function displayPreparation(value, key, inputFormat) {
    if (typeof inputFormat === 'string') {
        try {
            if (parsePreparation(inputFormat, key) === value) return inputFormat;
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

export const preparationFields = [
    ['pmax', 'Максимальний тиск PMAX', 'МПа'], ['vd', 'Швидкість Vд', 'м/с'],
    ['d', 'Калібр D', 'мм'], ['q', 'Маса снаряда Q', 'кг'],
    ['ns', 'Коефіцієнт нарізів NS', null], ['hi', 'Коефіцієнт HI', null],
    ['af', 'Коефіцієнт AFI', null], ['pfor', 'Тиск спрацьовування PFOR', 'МПа'],
    ['kt', 'Коефіцієнт KT', null], ['tn', 'Температура TN', '°C'],
    ['forma', 'Форма зерна FORMA', null, 'forma'],
    ['omega', 'Маса заряду OMEGA', 'кг'], ['w0', 'Обʼєм камори W0', 'м³'],
    ['ro', 'Густина пороху RO', 'кг/м³'], ['tg15', 'Температура TG15', 'К'],
    ['ku1', 'Коефіцієнт KU1', null], ['f1', 'Сила пороху F15', 'МДж/кг'],
    ['u15', 'Коефіцієнт U15', null], ['svod', 'Свод зерна SVOD', 'мм'],
    ['pvos', 'Тиск воспламенителя PVOS', 'МПа'], ['tvos', 'Температура воспламенителя TVOS', 'К'],
    ['lc', 'Довжина канулу ствола LS', 'м'],
];

export const formaOptions = [
    [1, 'Куб, шар, цилиндр'],
    [2, 'Кв.бр, кр.пр.'],
    [3, 'Кв. пласт.'],
    [4, 'Лента'],
    [5, 'Тр. кор.'],
    [6, 'Трубка'],
    [7, '7-ми канальн.'],
];

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

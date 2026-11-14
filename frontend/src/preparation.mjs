export const preparationFields = [
    ['d', 'Параметр d', 'мм'], ['q', 'Параметр q', 'кг'],
    ['vd', 'Швидкість Vд', 'м/с'], ['lc', 'Параметр lc', 'м'],
    ['ns', 'Коефіцієнт ns', null], ['f1', 'Коефіцієнт f1', null], ['af', 'Коефіцієнт af', null],
];

// Only complete decimal numbers are accepted; raw text stays in the editor.
export function parsePreparation(text, key) {
    const raw = text.trim();
    if (!raw) return null;
    if (!/^[+-]?(?:\d+(?:[.,]\d+)?|[.,]\d+)(?:[eE][+-]?\d+)?$/.test(raw)) {
        const unfinished = /^[+-]?$/.test(raw) || /^[+-]?(?:\d+[.,]|(?:\d+(?:[.,]\d+)?|[.,]\d+)[eE][+-]?)$/.test(raw);
        throw new Error(unfinished ? 'Завершіть число' : 'Введіть число без одиниць і розділювачів тисяч');
    }
    const value = Number(raw.replace(',', '.'));
    const nonzero = /[1-9]/.test(raw.split(/[eE]/)[0]);
    const stored = key === 'd' ? value / 1000 : value;
    if (!Number.isFinite(value) || !Number.isFinite(stored) || (nonzero && stored === 0)) {
        throw new Error('Число поза діапазоном binary64');
    }
    return stored;
}

export function displayPreparation(value, key) {
    if (value == null) return '';
    // Avoid overflowing the display conversion for finite stored metres.
    if (key === 'd') {
        const mm = value * 1000;
        if (!Number.isFinite(mm)) return `${value}e3`.replace(/e([+-]?\d+)e3$/, (_, exponent) => `e${Number(exponent) + 3}`);
        return String(mm);
    }
    return String(value);
}

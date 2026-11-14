export function externalMetadata(result) {
    const data = result.chart;
    return [
        ['Тип', 'Зовнішній результат · не обчислено IBSolver'],
        ['Конфігурація №', result.configurationId],
        ['Джерело', result.source],
        ['Формат', result.formatVersion],
        ['Імпорт', result.id ? new Date(result.importedAt).toLocaleString('uk-UA') : 'Ще не збережено'],
        ['Точок / серію', data.time.length],
        ['Вісь X', `${data.time[0]} … ${data.time.at(-1)} с (з CSV)`],
        ['Серії', data.series.map(s => `${s.name} [${s.unit}]`).join('; ')],
        ['SHA-256 файла', result.checksum],
    ];
}

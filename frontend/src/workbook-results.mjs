// Display helpers for the WORKBOOK run summary (Диплом.xls preparation kernel).
// No DOM access; rows are plain data so the panel renderer and tests share them.

// formatWorkbookValue keeps six significant digits; scientific notation stays
// for very small quantities such as W0.
export function formatWorkbookValue(value) {
    if (typeof value !== 'number' || !Number.isFinite(value)) return '—';
    return value.toPrecision(6);
}

// Every row carries a hover help entry: title, workbook cell code and a plain
// description. Descriptions avoid source-file references; only the transfer
// block may point to the ballistics program it feeds.
function row(label, value, title, code, description) {
    return [label, value, {title, code, description}];
}

// workbookResultSections groups the kernel result for the results panel.
// Section titles and row labels follow the workbook sheet «Расчеты». The first
// section is the transfer block F72:F74 («Для введення в програму»).
export function workbookResultSections(result) {
    if (!result || typeof result !== 'object') return [];
    return [
        {
            title: 'Для введення в програму',
            rows: [
                row('W0 (F72), м³', formatWorkbookValue(result.transfer?.w0), 'W0 - обʼєм камори', 'F72', 'Обʼєм камори заряджання. Переноситься в програму внутрішньої балістики як параметр W0.'),
                row('pm (F73), МПа', formatWorkbookValue(result.transfer?.pm), 'pm - максимальний тиск', 'F73', 'Максимальний середньобалістичний тиск у каналі ствола. Переноситься в програму як параметр PMAX - масштаб кривої тиску.'),
                row('Lс.у (F74), м', formatWorkbookValue(result.transfer?.ls), 'LS - довжина ствола', 'F74', 'Уточнена довжина ствола для введення в програму (зменшена на 1,5d). Переноситься як параметр LS.'),
            ],
        },
        {
            title: 'Початкові величини',
            rows: [
                row('Eд (F13), Дж', formatWorkbookValue(result.ed), 'Eд - дульна енергія', 'F13', 'Дульна енергія снаряда: Eд = q·Vд²/2.'),
                row('Cq (F15), кг/м³', formatWorkbookValue(result.cq), 'cq - коефіцієнт маси снаряда', 'F15', 'Коефіцієнт маси снаряда: cq = q/d³.'),
                row('Ce (H17), МДж/м³', formatWorkbookValue(result.ceMJ), 'Ce - коефіцієнт могутності', 'H17', 'Коефіцієнт могутності: Ce = Eд/d³. За ним із таблиці Слухоцького визначають табличні величини.'),
                row('S (F20), м²', formatWorkbookValue(result.s), 'S - площа каналу ствола', 'F20', 'Площа поперечного перерізу каналу ствола з урахуванням глибини нарізів (коефіцієнт ns).'),
            ],
        },
        {
            title: 'Таблиця Слухоцького (рядок ' + String(result.tableIndex ?? '—') + ')',
            rows: [
                row('pm.кр (F25), МПа', formatWorkbookValue(result.pmKr), 'pm.кр - крешерний тиск', 'F25', 'Максимальний крешерний тиск у стволі з таблиці Слухоцького (лінійна інтерполяція за Ce).'),
                row('Delta (F26), кг/м³', formatWorkbookValue(result.delta), 'Δ - щільність заряджання', 'F26', 'Щільність заряджання з таблиці Слухоцького.'),
                row('χ (F27)', formatWorkbookValue(result.hi), 'χ - коефіцієнт уширення камори', 'F27', 'Коефіцієнт уширення камори з таблиці Слухоцького.'),
                row('ηω (F28), МДж/кг', formatWorkbookValue(result.kisp), 'ηω - використання маси заряду', 'F28', 'Коефіцієнт використання одиниці маси заряду з таблиці Слухоцького.'),
                row('Lств/d (F29)', formatWorkbookValue(result.lstvD), 'Lств/d - довжина ствола в калибрах', 'F29', 'Довжина ствола в калибрах із таблиці Слухоцького.'),
            ],
        },
        {
            title: 'Камора і ствол',
            rows: [
                row('ω (F31), кг', formatWorkbookValue(result.omega), 'ω - маса заряду', 'F31', 'Маса порохового заряду: ω = Eд/(ηω·10⁶).'),
                row('W0 (F33), м³', formatWorkbookValue(result.w0), 'W0 - обʼєм камори', 'F33', 'Обʼєм камори: W0 = ω/Δ.'),
                row('l0 (F35), м', formatWorkbookValue(result.l0), 'l0 - приведена довжина камори', 'F35', 'Приведена довжина камори: l0 = W0/S.'),
                row('Lств (F38), м', formatWorkbookValue(result.lcCalc), 'Lств - довжина ствола', 'F38', 'Довжина ствола з таблиці: Lств = (Lств/d)·d.'),
                row('LS (I39), м', formatWorkbookValue(result.ls), 'LS - довжина ствола для програми', 'I39', 'Довжина ствола для введення в програму: LS = Lств − 1,5d.'),
                row('lкам (F40), м', formatWorkbookValue(result.lkam), 'lкам - довжина камори', 'F40', 'Довжина камори: lкам = l0/χ.'),
                row('lд (F42), м', formatWorkbookValue(result.ld), 'lд - шлях снаряда по стволу', 'F42', 'Шлях снаряда по стволу: lд = LS − lкам.'),
                row('Λд (F44)', formatWorkbookValue(result.lambdaD), 'Λд - число обʼємів розширення', 'F44', 'Число обʼємів розширення: Λд = lд/l0.'),
            ],
        },
        {
            title: 'Уточнені значення',
            rows: [
                row('pm.кр (F50), МПа', formatWorkbookValue(result.pmKrU), 'pm.кр.у - уточнений крешерний тиск', 'F50', 'Уточнений крешерний тиск: збільшується на 5% при cq > 15000 кг/м³ і зменшується на 5% при cq < 13000 кг/м³.'),
                row('Lс.у (F51), м', formatWorkbookValue(result.lcU), 'Lс.у - уточнена довжина ствола', 'F51', 'Уточнена довжина ствола: збільшується на 10% при cq > 15000 кг/м³ і зменшується на 10% при cq < 13000 кг/м³.'),
                row('lд.у (F52), м', formatWorkbookValue(result.ldU), 'lд.у - уточнений шлях снаряда', 'F52', 'Уточнений шлях снаряда по стволу: lд.у = Lс.у − lкам.'),
                row('Λд.у (I52)', formatWorkbookValue(result.lambdaDU), 'Λд.у - уточнене число обʼємів розширення', 'I52', 'Уточнене число обʼємів розширення: Λд.у = lд.у/l0.'),
            ],
        },
        {
            title: 'Заключні величини',
            rows: [
                row('Λ1 (F55)', formatWorkbookValue(result.lambda1), 'Λ1 - допоміжний параметр', 'F55', 'Допоміжний параметр для перерахунку крешерного тиску: Λ1 = (Λд.у + 1/χ)/(Λд.у + 1).'),
                row('φ (F60)', formatWorkbookValue(result.fi), 'φ - коефіцієнт фіктивності', 'F60', 'Коефіцієнт фіктивності (облік вторинних робіт): φ = af + Λ1·ω/(3q).'),
                row('pm (F63), МПа', formatWorkbookValue(result.pm), 'pm - середньобалістичний тиск', 'F63', 'Максимальний середньобалістичний тиск: pm = 1,12·φ·pm.кр.у/(φ1·(1 + ½Λ1·ω/q)).'),
                row('ηд (F67)', formatWorkbookValue(result.ettaD), 'ηд - коефіцієнт заповнення діаграми', 'F67', 'Коефіцієнт заповнення індикаторної діаграми: ηд = φ·Eд/(pm·S·lд.у).'),
            ],
        },
    ];
}

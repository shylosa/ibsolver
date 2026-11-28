// Transfer plan for the workbook block F72:F74 («Для введення в програму»).
// Pure data and logic without DOM access, so the confirmation dialog in
// main.js and the tests share one implementation.
import {displayPreparation, parsePreparation} from './preparation.mjs';

// Targets map kernel transfer values onto working-screen preparation fields.
// The kernel returns W0 in m³, pm in MPa and LS in meters; the pmax editor
// field is also labeled MPa but stores pascals, hence the ×1e6 conversion.
export const workbookTransferTargets = [
    {key: 'w0', technicalId: 'W0', label: 'Геометричний обʼєм камори', unit: 'м³', stored: transfer => transfer.w0},
    {key: 'pmax', technicalId: 'PMAX', label: 'Максимальний тиск у стволі', unit: 'МПа', stored: transfer => transfer.pm * 1e6},
    {key: 'lc', technicalId: 'LS', label: 'Загальна довжина ствола', unit: 'м', stored: transfer => transfer.ls},
];

const workbookInputKeys = ['d', 'q', 'vd', 'lc', 'ns', 'f1', 'af'];

// The transfer is offered only while the shown workbook result was produced
// from exactly the seven inputs currently entered in the form. Input formats
// are cosmetic and deliberately ignored.
export function workbookTransferMatches(snapshot, current) {
    if (!snapshot || typeof snapshot !== 'object' || !current || typeof current !== 'object') return false;
    return workbookInputKeys.every(key => {
        const saved = snapshot[key], entered = current[key];
        return typeof saved === 'number' && Number.isFinite(saved) &&
            typeof entered === 'number' && Number.isFinite(entered) && saved === entered;
    });
}

// Rows for the confirmation dialog: one per target field with the current
// stored value (null when the field was not set) and the new stored value.
// Returns [] when the transfer block is missing or any new value is not a
// finite number — such results must never reach the working screen.
export function workbookTransferRows(transfer, current) {
    if (!transfer || typeof transfer !== 'object') return [];
    const rows = [];
    for (const target of workbookTransferTargets) {
        const newStored = target.stored(transfer);
        if (!Number.isFinite(newStored)) return [];
        const currentStored = current?.[target.key];
        const hasCurrent = typeof currentStored === 'number' && Number.isFinite(currentStored);
        rows.push({
            key: target.key,
            technicalId: target.technicalId,
            label: target.label,
            unit: target.unit,
            currentStored: hasCurrent ? currentStored : null,
            newStored,
            currentText: hasCurrent ? displayPreparation(currentStored, target.key) : '—',
            newText: displayPreparation(newStored, target.key),
        });
    }
    return rows;
}

// Values edited in the dialog's "new value" column are parsed with the same
// rules as the working-screen fields, so display text round-trips into stored
// units (MPa -> Pa for pmax, exponential text for w0, comma or dot decimals).
export function parseTransferValue(text, key) {
    const raw = typeof text === 'string' ? text.trim() : '';
    if (!raw) throw new Error('Введіть число');
    const stored = parsePreparation(raw, key);
    if (stored == null || !Number.isFinite(stored)) throw new Error('Введіть скінченне число');
    return stored;
}

// Validates every edited cell of the confirmation dialog. The thrown error
// carries `index` of the offending row so the dialog can highlight it.
export function parseTransferRows(rows, texts) {
    if (!Array.isArray(rows) || !Array.isArray(texts) || texts.length !== rows.length) {
        throw new Error('Невідповідна кількість значень для передачі');
    }
    return rows.map((row, index) => {
        try {
            return {key: row.key, stored: parseTransferValue(texts[index], row.key)};
        } catch (error) {
            error.index = index;
            throw error;
        }
    });
}

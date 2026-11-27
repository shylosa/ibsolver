const colors = ['#60a5fa', '#f472b6', '#facc15', '#fb7185', '#2dd4bf', '#a3e635'];
const millisecondsPerSecond = 1000;

export const verticalLimit = 1.25;

export function normalizationDivisor(series) {
    if (series.reference !== undefined && series.reference !== null) {
        if (!Number.isFinite(series.reference) || series.reference <= 0) {
            throw new Error('Опорне значення серії має бути скінченним і додатним.');
        }
        return series.reference;
    }
    return series.values.reduce((max, value) => Math.max(max, Math.abs(value)), 0) || 1;
}

export function seriesPreference(settings, series, index) {
    settings.series ||= {};
    return settings.series[series.id] ||= {visible: settings.visible?.[index] ?? true, scale: settings.scales?.[index] ?? 1};
}

export function formatAxisValue(value) {
    if (!Number.isFinite(value)) return '—';
    const abs = Math.abs(value);
    if (abs >= 1e6 || (abs > 0 && abs < 1e-3)) {
        return value.toExponential(2).replace('.', ',').replace(/e([+-]?)(\d+)$/, (_, sign, digits) => `E${sign}${digits.padStart(2, '0')}`);
    }
    if (abs >= 100) return value.toFixed(0);
    if (abs >= 10) return value.toFixed(1).replace('.', ',');
    return value.toFixed(2).replace('.', ',');
}

// Y grid steps stay on multiples of 25 (25, 50, 75, 100, ...); the step only
// doubles when the axis range grows past eight labels.
export function percentTickStep(span) {
    let step = 25;
    while (span / step > 8) step *= 2;
    return step;
}

// Horizontal axis labels always carry exactly one decimal digit.
export function formatXAxisTick(ms) {
    return Number.isFinite(ms) ? ms.toFixed(1).replace('.', ',') : '—';
}

export function valueRange(series, divisor, scale) {
    const values = (series.values || []).map(value => value / divisor * scale * 100).filter(Number.isFinite);
    if (!values.length) return [0, 1];
    let min = Math.min(0, ...values);
    let max = Math.max(0, ...values);
    if (max === min) max = min + 1;
    if (min >= 0) {
        max *= 1.05;
    } else {
        const span = max - min;
        min -= span * 0.05;
        max += span * 0.05;
    }
    return [min, max];
}

// This component only consumes numeric series and display preferences.
export class TimeSeriesChart {
    constructor(canvas, legend, cursor, onChange) {
        Object.assign(this, {canvas, legend, cursor, onChange, data: null, index: null});
        this.context = canvas.getContext('2d');
        this.observer = new ResizeObserver(() => this.draw());
        this.observer.observe(canvas);
        canvas.addEventListener('pointerdown', event => {
            if (event.button !== 0 || !this.data || this.fraction(event) === null) return;
            event.preventDefault();
            this.drag = {x: event.clientX, range: [...this.range]};
            canvas.setPointerCapture(event.pointerId);
        });
        canvas.addEventListener('pointermove', event => {
            if (!this.data) return;
            if (this.drag) {
                const delta = (this.drag.x - event.clientX) / Math.max(1, canvas.getBoundingClientRect().width - 76) * (this.drag.range[1] - this.drag.range[0]);
                this.range = clampRange(this.data.time, this.drag.range[0] + delta, this.drag.range[1] - this.drag.range[0]);
            }
            const fraction = this.fraction(event);
            this.index = fraction === null ? null : nearestIndex(this.data.time, this.range[0] + fraction * (this.range[1] - this.range[0]));
            this.requestDraw();
        });
        const stop = () => { this.drag = null; };
        canvas.addEventListener('pointerup', event => { stop(); if (canvas.hasPointerCapture(event.pointerId)) canvas.releasePointerCapture(event.pointerId); });
        canvas.addEventListener('pointercancel', stop);
        canvas.addEventListener('lostpointercapture', stop);
        canvas.addEventListener('pointerleave', () => { this.index = null; this.requestDraw(); });
        canvas.addEventListener('dblclick', () => this.reset());
    }

    fraction(event) {
        const rect = this.canvas.getBoundingClientRect();
        const x = event.clientX - rect.left, y = event.clientY - rect.top;
        return x < 68 || x > rect.width - 22 || y < 12 || y > rect.height - 44 ? null : (x - 68) / Math.max(1, rect.width - 90);
    }

    requestDraw() {
        if (this.frame) return;
        this.frame = requestAnimationFrame(() => { this.frame = null; this.draw(); });
    }

    reset() {
        this.range = this.data ? [this.data.time[0], this.data.time.at(-1)] : null;
        this.index = null;
        this.requestDraw();
    }

    set(data, settings) {
        if (data !== this.data) this.range = data ? [data.time[0], data.time.at(-1)] : null;
        this.data = data;
        this.settings = settings;
        this.divisors = (data?.series || []).map(normalizationDivisor);
        this.legend.replaceChildren();
        (data?.series || []).forEach((series, i) => {
            const preference = seriesPreference(settings, series, i);
            const row = document.createElement('div'); row.className = 'legend-item';
            const label = document.createElement('label');
            const check = document.createElement('input'); check.type = 'checkbox'; check.checked = preference.visible;
            check.addEventListener('change', () => { preference.visible = check.checked; this.draw(); this.onChange(); });
            const dot = document.createElement('span'); dot.className = 'dot'; dot.style.background = colors[i];
            row.title = series.reference == null ? 'Автоматична нормалізація: max|y|' : `Опорна нормалізація: ${series.reference} ${series.unit}`;
            label.append(check, dot, document.createTextNode(`${series.name} · ${series.unit}`));
            const scale = document.createElement('input');
            Object.assign(scale, {type: 'number', min: '0.1', max: '10', step: '0.1', value: preference.scale});
            scale.setAttribute('aria-label', `Масштаб: ${series.name}`);
            scale.addEventListener('change', () => {
                if (!scale.checkValidity() || !Number.isFinite(scale.valueAsNumber)) { scale.value = preference.scale; return; }
                preference.scale = scale.valueAsNumber; this.draw(); this.onChange();
            });
            row.append(label, document.createTextNode('×'), scale); this.legend.append(row);
        });
        this.index = null;
        this.draw();
    }

    draw() {
        const {canvas, context: c, data, settings} = this;
        const {width: w, height: h} = canvas.getBoundingClientRect();
        if (!w || !h) return;
        const ratio = window.devicePixelRatio || 1;
        if (canvas.width !== Math.round(w * ratio)) canvas.width = Math.round(w * ratio);
        if (canvas.height !== Math.round(h * ratio)) canvas.height = Math.round(h * ratio);
        c.setTransform(ratio, 0, 0, ratio, 0, 0);
        c.fillStyle = '#151d29'; c.fillRect(0, 0, w, h);
        if (!data) {
            this.cursor.textContent = '';
            return;
        }
        const left = 68, top = 12, pw = Math.max(1, w - 90), ph = Math.max(1, h - 56);
        const visible = data.series.map((series, index) => ({series, index, preference: seriesPreference(settings, series, index)})).filter(item => item.preference.visible);
        // One shared percent axis: every curve is plotted as a percent of its
        // own reference (or max|y|), multiplied by the legend scale.
        let rangeMin = 0, rangeMax = 1;
        visible.forEach(item => {
            const [min, max] = valueRange(item.series, this.divisors[item.index], item.preference.scale);
            rangeMin = Math.min(rangeMin, min);
            rangeMax = Math.max(rangeMax, max);
        });
        // Floating-point noise such as -1.42e-14% must not open the axis.
        if (rangeMin < 0 && rangeMin > -1e-9) rangeMin = 0;
        c.font = '11px Segoe UI, sans-serif';
        // Y grid starts at 0; intermediate ticks sit on multiples of the step,
        // the axis maximum gets a label only when it is far enough from the
        // last multiple.
        const yStep = percentTickStep(rangeMax - rangeMin);
        const firstStep = Math.ceil(rangeMin / yStep - 1e-9);
        const lastStep = Math.floor(rangeMax / yStep + 1e-9);
        const yTicks = [];
        for (let k = firstStep; k <= lastStep; k++) yTicks.push(k * yStep);
        if (rangeMax - yTicks.at(-1) > yStep / 2) yTicks.push(rangeMax);
        for (const value of yTicks) {
            const y = top + ph * (1 - (value - rangeMin) / (rangeMax - rangeMin));
            c.strokeStyle = '#354458'; c.lineWidth = 1;
            c.beginPath(); c.moveTo(left, y); c.lineTo(left + pw, y); c.stroke();
            c.fillStyle = '#b7c5d6'; c.textAlign = 'right';
            c.fillText(Number.isInteger(value) ? String(value) : formatAxisValue(value), left - 10, y + 4);
        }
        const times = data.time;
        const [min, max] = this.range;
        const xAt = i => left + (times[i] - min) / (max - min) * pw;
        const first = Math.max(0, nearestIndex(times, min) - 1), last = Math.min(times.length - 1, nearestIndex(times, max) + 1);
        // X grid: intermediate ticks at whole milliseconds, the final tick
        // sits at the end of the range and may be fractional.
        const minMs = min * millisecondsPerSecond, maxMs = max * millisecondsPerSecond;
        const xStepMs = Math.max(1, Math.ceil((maxMs - minMs) / 5));
        const xTicks = [];
        for (let t = Math.ceil(minMs - 1e-9); t < maxMs - 1e-9; t += xStepMs) xTicks.push(t);
        xTicks.push(maxMs);
        for (const tickMs of xTicks) {
            const x = left + (tickMs / millisecondsPerSecond - min) / (max - min) * pw;
            c.strokeStyle = '#354458'; c.lineWidth = 1;
            c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke();
            c.fillStyle = '#b7c5d6'; c.textAlign = 'center';
            c.fillText(formatXAxisTick(tickMs), x, top + ph + 21);
        }
        c.fillStyle = '#dce5ef'; c.textAlign = 'center'; c.font = '600 12px Segoe UI, sans-serif';
        c.fillText('Час, мс', left + pw / 2, h - 7);
        c.save(); c.translate(13, top + ph / 2); c.rotate(-Math.PI / 2);
        c.fillText('% від max', 0, 0); c.restore();
        c.strokeStyle = '#9aa9bb'; c.lineWidth = 1.5;
        c.strokeRect(left, top, pw, ph);
        c.save(); c.beginPath(); c.rect(left, top, pw, ph); c.clip();
        data.series.forEach((s, j) => {
            const preference = seriesPreference(settings, s, j);
            if (!preference.visible) return;
            c.strokeStyle = colors[j]; c.lineWidth = 1.5; c.beginPath();
            for (let i = first; i <= last; i++) {
                const percent = s.values[i] * preference.scale / this.divisors[j] * 100;
                const y = top + ph * (1 - (percent - rangeMin) / (rangeMax - rangeMin));
                if (i === first) c.moveTo(xAt(i), y); else c.lineTo(xAt(i), y);
            } c.stroke();
        });
        if (this.index !== null) {
            const x = xAt(this.index); c.strokeStyle = '#e5edf7'; c.setLineDash([4, 4]);
            c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke(); c.setLineDash([]);
        }
        c.restore();
        this.cursor.replaceChildren();
        if (this.index === null) return;
        const intro = document.createElement('span');
        intro.textContent = `t = ${(times[this.index] * millisecondsPerSecond).toFixed(3)} мс`;
        this.cursor.append(intro);
        if (this.index !== null) data.series.forEach((s, j) => {
            const preference = seriesPreference(settings, s, j);
            if (!preference.visible) return;
            const value = document.createElement('span'); value.style.color = colors[j];
            value.textContent = `${s.name}: ${s.values[this.index].toPrecision(5)} ${s.unit}`; this.cursor.append(value);
        });
    }
}

// Binary search on the immutable, strictly increasing time axis.
export function nearestIndex(times, target) {
    let low = 0, high = times.length - 1;
    while (low < high) { const mid = (low + high) >> 1; if (times[mid] < target) low = mid + 1; else high = mid; }
    return low > 0 && target - times[low - 1] < times[low] - target ? low - 1 : low;
}

export function clampRange(times, start, span) {
    const min = times[0], max = times.at(-1);
    span = Math.min(max - min, Math.max((max - min) / 1000, span));
    start = Math.max(min, Math.min(max - span, start));
    return [start, start + span];
}


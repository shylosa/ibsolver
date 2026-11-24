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

export function chartTitle(series) {
    const ids = new Set((series || []).map(item => item.id));
    if ([...ids].some(id => id.startsWith('preparation:'))) {
        if (ids.has('preparation:p') && ids.has('preparation:l')) return 'Тиск і переміщення від часу';
        if (ids.has('preparation:p')) return 'Давлення від часу';
        if (ids.has('preparation:l')) return 'Переміщення від часу';
        return 'Балістичні часові ряди';
    }
    if ([...ids].some(id => id.startsWith('oscillator:'))) return 'Коливання';
    if ([...ids].some(id => id.startsWith('demo:'))) return 'Демонстраційні криві';
    return 'Часові ряди';
}

export function valueRange(series, divisor, scale) {
    const values = (series.values || []).map(value => value * scale).filter(Number.isFinite);
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
        return x < 68 || x > rect.width - 22 || y < 42 || y > rect.height - 44 ? null : (x - 68) / Math.max(1, rect.width - 90);
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
        const left = 68, top = 42, pw = Math.max(1, w - 90), ph = Math.max(1, h - 86);
        const visible = data.series.map((series, index) => ({series, index, preference: seriesPreference(settings, series, index)})).filter(item => item.preference.visible);
        const units = [];
        visible.forEach(item => {
            if (!units.includes(item.series.unit)) units.push(item.series.unit);
        });
        if (!units.length) units.push('Нормоване значення');
        const ranges = visible.length ? units.map(unit => {
            const items = visible.filter(item => item.series.unit === unit);
            let min = 0, max = 0;
            items.forEach(item => {
                const range = valueRange(item.series, this.divisors[item.index], item.preference.scale);
                min = Math.min(min, range[0]);
                max = Math.max(max, range[1]);
            });
            return [min, max];
        }) : [[0,1]];
        c.font = '11px Segoe UI, sans-serif';
        c.fillStyle = '#e5edf7'; c.textAlign = 'center';
        c.font = '600 16px Segoe UI, sans-serif';
        c.fillText(chartTitle(visible.map(item => item.series)), left + pw / 2, 24);
        c.font = '11px Segoe UI, sans-serif';
        const ticks = ph < 80 ? 2 : 5;
        const yLabel = !visible.length ? 'Нормоване значення' : units.length === 1 ? `${visible[0].series.name.replace(/^[^ ]+ /, '')}, ${units[0]}` : `${units[0]} / ${units.slice(1).join(' / ')}`;
        const primaryRange = ranges[0];
        const secondaryRange = ranges[1];
        for (let i = 0; i <= ticks; i++) {
            const y = top + i * ph / ticks;
            c.strokeStyle = i === ticks ? '#718096' : '#354458'; c.lineWidth = i === ticks ? 1.5 : 1;
            c.beginPath(); c.moveTo(left, y); c.lineTo(left + pw, y); c.stroke();
            c.fillStyle = '#b7c5d6'; c.textAlign = 'right';
            const value = primaryRange[1] - (primaryRange[1] - primaryRange[0]) * i / ticks;
            c.fillText(formatAxisValue(value), left - 10, y + 4);
            if (secondaryRange) {
                c.textAlign = 'left';
                const secondaryValue = secondaryRange[1] - (secondaryRange[1] - secondaryRange[0]) * i / ticks;
                c.fillText(formatAxisValue(secondaryValue), left + pw + 10, y + 4);
            }
        }
        const times = data.time;
        const [min, max] = this.range;
        const xAt = i => left + (times[i] - min) / (max - min) * pw;
        const first = Math.max(0, nearestIndex(times, min) - 1), last = Math.min(times.length - 1, nearestIndex(times, max) + 1);
        for (let i = 0; i <= 5; i++) {
            const x = left + pw * i / 5;
            c.strokeStyle = '#354458'; c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke();
            c.fillStyle = '#b7c5d6'; c.textAlign = 'center';
            const tickTime = min + (max - min) * i / 5;
            c.fillText(formatAxisValue(tickTime * millisecondsPerSecond), x, top + ph + 21);
        }
        c.fillStyle = '#dce5ef'; c.textAlign = 'center'; c.font = '600 12px Segoe UI, sans-serif';
        c.fillText('Час, мс', left + pw / 2, h - 7);
        c.save(); c.translate(13, top + ph / 2); c.rotate(-Math.PI / 2);
        c.fillText(yLabel, 0, 0); c.restore();
        if (units.length > 1) {
            c.save(); c.translate(w - 7, top + ph / 2); c.rotate(-Math.PI / 2);
            c.fillStyle = '#dce5ef'; c.font = '600 11px Segoe UI, sans-serif'; c.textAlign = 'center';
            c.fillText(units[1], 0, 0); c.restore();
            c.fillStyle = '#dce5ef'; c.font = '600 11px Segoe UI, sans-serif'; c.textAlign = 'right';
            c.fillText(units[0], left - 10, top - 10);
        }
        c.strokeStyle = '#9aa9bb'; c.lineWidth = 1.5;
        c.strokeRect(left, top, pw, ph);
        c.save(); c.beginPath(); c.rect(left, top, pw, ph); c.clip();
        data.series.forEach((s, j) => {
            const preference = seriesPreference(settings, s, j);
            if (!preference.visible) return;
            c.strokeStyle = colors[j]; c.lineWidth = 1.5; c.beginPath();
            const axis = units.indexOf(s.unit);
            const range = ranges[Math.max(0, axis)];
            for (let i = first; i <= last; i++) {
                const value = s.values[i];
                const scaled = value * preference.scale;
                const y = top + ph * (1 - (scaled - range[0]) / (range[1] - range[0]));
                if (i === first) c.moveTo(xAt(i), y); else c.lineTo(xAt(i), y);
            } c.stroke();
        });
        if (this.index !== null) {
            const x = xAt(this.index); c.strokeStyle = '#e5edf7'; c.setLineDash([4, 4]);
            c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke(); c.setLineDash([]);
        }
        c.restore();
        this.cursor.replaceChildren();
        const intro = document.createElement('span');
        intro.textContent = this.index === null
            ? 'Кожна крива: y / опора · масштаб; опора — задана або max|y|. Значення курсора — вихідні.'
            : `t = ${(times[this.index] * millisecondsPerSecond).toFixed(3)} мс`;
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


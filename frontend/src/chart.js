const colors = ['#60c5fa', '#f1b75e', '#b49af8', '#6ed6b2', '#ed8598', '#c9d86a'];

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

// This component only consumes numeric series and display preferences.
export class TimeSeriesChart {
    constructor(canvas, legend, cursor, onChange) {
        Object.assign(this, {canvas, legend, cursor, onChange, data: null, index: null});
        this.context = canvas.getContext('2d');
        this.observer = new ResizeObserver(() => this.draw());
        this.observer.observe(canvas);
        canvas.addEventListener('wheel', event => {
            const fraction = this.fraction(event);
            if (fraction === null || !this.data) return;
            event.preventDefault();
            this.range = zoomRange(this.data.time, this.range, fraction, Math.exp(Math.max(-2, Math.min(2, event.deltaY * .002))));
            this.index = nearestIndex(this.data.time, this.range[0] + fraction * (this.range[1] - this.range[0]));
            this.requestDraw();
        }, {passive: false});
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
        return x < 54 || x > rect.width - 22 || y < 22 || y > rect.height - 44 ? null : (x - 54) / Math.max(1, rect.width - 76);
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
        c.fillStyle = '#131b26'; c.fillRect(0, 0, w, h);
        if (!data) {
            this.cursor.textContent = '';
            return;
        }
        const left = 54, top = 22, pw = Math.max(1, w - 76), ph = Math.max(1, h - 66);
        c.font = '11px Consolas, monospace';
        const ticks = ph < 80 ? 2 : 4;
        for (let i = 0; i <= ticks; i++) {
            const y = top + i * ph / ticks;
            c.strokeStyle = '#283446'; c.beginPath(); c.moveTo(left, y); c.lineTo(left + pw, y); c.stroke();
            c.fillStyle = '#91a0b3'; c.textAlign = 'right'; c.fillText(((1 - 2 * i / ticks) * verticalLimit).toFixed(2), left - 10, y + 4);
        }
        const times = data.time;
        const [min, max] = this.range;
        const xAt = i => left + (times[i] - min) / (max - min) * pw;
        const first = Math.max(0, nearestIndex(times, min) - 1), last = Math.min(times.length - 1, nearestIndex(times, max) + 1);
        for (let i = 0; i <= 5; i++) {
            const x = left + pw * i / 5;
            c.strokeStyle = '#283446'; c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke();
            c.fillStyle = '#91a0b3'; c.textAlign = 'center';
            c.fillText((min + (max - min) * i / 5).toPrecision(5), x, top + ph + 21);
        }
        c.fillText('Час, с', left + pw / 2, h - 7);
        c.save(); c.beginPath(); c.rect(left, top, pw, ph); c.clip();
        data.series.forEach((s, j) => {
            const preference = seriesPreference(settings, s, j);
            if (!preference.visible) return;
            c.strokeStyle = colors[j]; c.lineWidth = 1.5; c.beginPath();
            for (let i = first; i <= last; i++) {
                const value = s.values[i];
                const y = top + ph / 2 * (1 - Math.max(-1e9, Math.min(1e9, value / this.divisors[j] * preference.scale)) / verticalLimit);
                if (i === first) c.moveTo(xAt(i), y); else c.lineTo(xAt(i), y);
            } c.stroke();
        });
        if (this.index !== null) {
            const x = xAt(this.index); c.strokeStyle = '#c5d0df'; c.setLineDash([4, 4]);
            c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke(); c.setLineDash([]);
        }
        c.restore();
        this.cursor.replaceChildren();
        const intro = document.createElement('span');
        intro.textContent = this.index === null ? 'Кожна крива: y / опора · масштаб; опора — задана або max|y|. Значення курсора — вихідні.' : `t = ${times[this.index].toFixed(3)} с`;
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

export function zoomRange(times, range, fraction, factor) {
    const span = Math.min(times.at(-1) - times[0], Math.max((times.at(-1) - times[0]) / 1000, (range[1] - range[0]) * factor));
    return clampRange(times, range[0] + fraction * (range[1] - range[0]) - fraction * span, span);
}

const colors = ['#60c5fa', '#f1b75e', '#b49af8', '#6ed6b2', '#ed8598', '#c9d86a'];

// This component only consumes numeric series and display preferences.
export class TimeSeriesChart {
    constructor(canvas, legend, cursor, onChange) {
        Object.assign(this, {canvas, legend, cursor, onChange, data: null, index: null});
        this.context = canvas.getContext('2d');
        this.observer = new ResizeObserver(() => this.draw());
        this.observer.observe(canvas);
        canvas.addEventListener('pointermove', event => {
            if (!this.data) return;
            const rect = canvas.getBoundingClientRect();
            const fraction = Math.max(0, Math.min(1, (event.clientX - rect.left - 54) / (rect.width - 76)));
            const times = this.data.time;
            const target = times[0] + fraction * (times.at(-1) - times[0]);
            let low = 0, high = times.length - 1;
            while (low < high) { const mid = (low + high) >> 1; if (times[mid] < target) low = mid + 1; else high = mid; }
            this.index = low > 0 && target - times[low - 1] < times[low] - target ? low - 1 : low;
            this.draw();
        });
        canvas.addEventListener('pointerleave', () => { this.index = null; this.draw(); });
    }

    set(data, settings) {
        this.data = data;
        this.settings = settings;
        this.legend.replaceChildren();
        (data?.series || []).forEach((series, i) => {
            const row = document.createElement('div'); row.className = 'legend-item';
            const label = document.createElement('label');
            const check = document.createElement('input'); check.type = 'checkbox'; check.checked = settings.visible[i];
            check.addEventListener('change', () => { settings.visible[i] = check.checked; this.draw(); this.onChange(); });
            const dot = document.createElement('span'); dot.className = 'dot'; dot.style.background = colors[i];
            label.append(check, dot, document.createTextNode(series.name));
            const scale = document.createElement('input');
            Object.assign(scale, {type: 'number', min: '0.1', max: '10', step: '0.1', value: settings.scales[i]});
            scale.setAttribute('aria-label', `Масштаб: ${series.name}`);
            scale.addEventListener('change', () => {
                if (!scale.checkValidity() || !Number.isFinite(scale.valueAsNumber)) { scale.value = settings.scales[i]; return; }
                settings.scales[i] = scale.valueAsNumber; this.draw(); this.onChange();
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
        canvas.width = Math.round(w * ratio); canvas.height = Math.round(h * ratio);
        c.setTransform(ratio, 0, 0, ratio, 0, 0);
        c.fillStyle = '#131b26'; c.fillRect(0, 0, w, h);
        const left = 54, top = 22, pw = Math.max(1, w - 76), ph = Math.max(1, h - 66);
        c.font = '11px Consolas, monospace';
        for (let i = 0; i <= 4; i++) {
            const y = top + i * ph / 4;
            c.strokeStyle = '#283446'; c.beginPath(); c.moveTo(left, y); c.lineTo(left + pw, y); c.stroke();
            c.fillStyle = '#91a0b3'; c.textAlign = 'right'; c.fillText((1 - i / 2).toFixed(1), left - 10, y + 4);
        }
        if (!data) {
            c.fillStyle = '#91a0b3'; c.textAlign = 'center';
            c.fillText('Натисніть «Розрахувати», щоб отримати часові ряди.', w / 2, h / 2);
            this.cursor.textContent = 'Наведіть курсор на графік, щоб переглянути значення.';
            return;
        }
        const times = data.time;
        const xAt = i => left + (times[i] - times[0]) / (times.at(-1) - times[0]) * pw;
        for (let i = 0; i <= 5; i++) {
            const x = left + pw * i / 5;
            c.strokeStyle = '#283446'; c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke();
            c.fillStyle = '#91a0b3'; c.textAlign = 'center';
            c.fillText((times[0] + (times.at(-1) - times[0]) * i / 5).toFixed(2), x, top + ph + 21);
        }
        c.fillText('Час, с', left + pw / 2, h - 7);
        c.save(); c.beginPath(); c.rect(left, top, pw, ph); c.clip();
        const maxima = data.series.map(s => s.values.reduce((max, n) => Math.max(max, Math.abs(n)), 0) || 1);
        data.series.forEach((s, j) => {
            if (!settings.visible[j]) return;
            c.strokeStyle = colors[j]; c.lineWidth = 1.5; c.beginPath();
            s.values.forEach((value, i) => {
                const y = top + ph / 2 * (1 - value / maxima[j] * settings.scales[j]);
                if (i === 0) c.moveTo(xAt(i), y); else c.lineTo(xAt(i), y);
            }); c.stroke();
        });
        if (this.index !== null) {
            const x = xAt(this.index); c.strokeStyle = '#c5d0df'; c.setLineDash([4, 4]);
            c.beginPath(); c.moveTo(x, top); c.lineTo(x, top + ph); c.stroke(); c.setLineDash([]);
        }
        c.restore();
        this.cursor.replaceChildren();
        const intro = document.createElement('span');
        intro.textContent = this.index === null ? 'Кожна крива: y / max|y| · масштаб. Значення курсора — вихідні.' : `t = ${times[this.index].toFixed(3)} с`;
        this.cursor.append(intro);
        if (this.index !== null) data.series.forEach((s, j) => {
            if (!settings.visible[j]) return;
            const value = document.createElement('span'); value.style.color = colors[j];
            value.textContent = `${s.name}: ${s.values[this.index].toPrecision(5)} ${s.unit}`; this.cursor.append(value);
        });
    }
}

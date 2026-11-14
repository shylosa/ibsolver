// A flush captures the current draft before a configuration-changing operation.
export class DebouncedTask {
    constructor(task, delay = 450) { this.task = task; this.delay = delay; this.timer = null; }
    schedule() {
        this.cancel();
        this.timer = setTimeout(() => { this.timer = null; this.task(); }, this.delay);
    }
    flush() {
        if (this.timer === null) return;
        this.cancel();
        this.task();
    }
    cancel() { if (this.timer !== null) clearTimeout(this.timer); this.timer = null; }
}

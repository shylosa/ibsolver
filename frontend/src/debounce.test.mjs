import test from 'node:test';
import assert from 'node:assert/strict';
import {DebouncedTask} from './debounce.mjs';

test('debounce merges typing, flushes once and cancels pending callbacks', t => {
    t.mock.timers.enable({apis:['setTimeout']});
    let draft = '', writes = [];
    const task = new DebouncedTask(() => writes.push(draft));
    draft = '1'; task.schedule(); t.mock.timers.tick(300);
    draft = '1.23'; task.schedule(); t.mock.timers.tick(449);
    assert.deepEqual(writes, []);
    t.mock.timers.tick(1); assert.deepEqual(writes, ['1.23']);
    draft = '2'; task.schedule(); task.flush(); task.flush();
    t.mock.timers.tick(500); assert.deepEqual(writes, ['1.23','2']);
    task.schedule(); task.cancel(); t.mock.timers.tick(500);
    assert.equal(writes.length,2);
});

// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2025 happyDomain
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { describe, it, expect, vi } from "vitest";
import { Mutex } from "./mutex";

// Small helper to yield control back to the microtask queue.
const tick = () => Promise.resolve();

describe("Mutex", () => {
    it("resolves lock() immediately when unlocked", async () => {
        const mutex = new Mutex();

        const unlock = await mutex.lock();

        expect(typeof unlock).toBe("function");
    });

    it("blocks a second lock() until the first is unlocked", async () => {
        const mutex = new Mutex();

        const unlock1 = await mutex.lock();

        let secondAcquired = false;
        const secondLockPromise = mutex.lock().then((unlock) => {
            secondAcquired = true;
            return unlock;
        });

        await tick();
        await tick();
        expect(secondAcquired).toBe(false);

        unlock1();
        const unlock2 = await secondLockPromise;

        expect(secondAcquired).toBe(true);
        expect(typeof unlock2).toBe("function");
    });

    it("grants the lock to waiters in FIFO order", async () => {
        const mutex = new Mutex();
        const order: number[] = [];

        const unlock1 = await mutex.lock();

        const p2 = mutex.lock().then((unlock) => {
            order.push(2);
            return unlock;
        });
        const p3 = mutex.lock().then((unlock) => {
            order.push(3);
            return unlock;
        });
        const p4 = mutex.lock().then((unlock) => {
            order.push(4);
            return unlock;
        });

        unlock1();
        const unlock2 = await p2;
        unlock2();
        const unlock3 = await p3;
        unlock3();
        const unlock4 = await p4;
        unlock4();

        expect(order).toEqual([2, 3, 4]);
    });

    it("allows the lock to be re-acquired sequentially after each unlock", async () => {
        const mutex = new Mutex();

        const unlock1 = await mutex.lock();
        unlock1();

        const unlock2 = await mutex.lock();
        unlock2();

        const unlock3 = await mutex.lock();
        unlock3();

        // No assertion errors / hangs means the mutex correctly reset to unlocked each time.
        expect(true).toBe(true);
    });

    it("does not throw and remains usable when unlock() is called multiple times", async () => {
        const mutex = new Mutex();

        const unlock1 = await mutex.lock();
        unlock1();
        expect(() => unlock1()).not.toThrow();

        // Mutex should still be acquirable afterwards.
        const unlock2 = await mutex.lock();
        expect(typeof unlock2).toBe("function");
        unlock2();
    });

    it("only wakes a single waiter per unlock() call", async () => {
        const mutex = new Mutex();

        const unlock1 = await mutex.lock();

        const acquired: number[] = [];
        const p2 = mutex.lock().then((unlock) => {
            acquired.push(2);
            return unlock;
        });
        const p3 = mutex.lock().then((unlock) => {
            acquired.push(3);
            return unlock;
        });

        unlock1();
        const unlock2 = await p2;

        await tick();
        await tick();
        expect(acquired).toEqual([2]);

        unlock2();
        const unlock3 = await p3;
        unlock3();

        expect(acquired).toEqual([2, 3]);
    });

    it("serializes concurrent critical sections so they never overlap", async () => {
        const mutex = new Mutex();
        let active = 0;
        let maxActive = 0;
        const results: number[] = [];

        const task = async (id: number) => {
            const unlock = await mutex.lock();
            active++;
            maxActive = Math.max(maxActive, active);
            await tick();
            results.push(id);
            active--;
            unlock();
        };

        await Promise.all([task(1), task(2), task(3), task(4), task(5)]);

        expect(maxActive).toBe(1);
        expect(results.sort((a, b) => a - b)).toEqual([1, 2, 3, 4, 5]);
    });

    it("keeps independent Mutex instances from blocking each other", async () => {
        const mutexA = new Mutex();
        const mutexB = new Mutex();

        const unlockA = await mutexA.lock();

        let acquiredB = false;
        const unlockB = await mutexB.lock();
        acquiredB = true;

        expect(acquiredB).toBe(true);

        unlockA();
        unlockB();
    });

    it("propagates unlock() call count matching lock() call count under contention", async () => {
        const mutex = new Mutex();
        const unlockSpies: Array<() => void> = [];

        const wrap = async () => {
            const unlock = await mutex.lock();
            const spy = vi.fn(unlock);
            unlockSpies.push(spy);
            spy();
        };

        await Promise.all([wrap(), wrap(), wrap()]);

        expect(unlockSpies).toHaveLength(3);
        unlockSpies.forEach((spy) => expect(spy).toHaveBeenCalledTimes(1));
    });
});

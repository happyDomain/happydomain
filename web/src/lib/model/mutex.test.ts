import { describe, it, expect } from "vitest";
import { Mutex } from "./mutex";

describe("Mutex", () => {
    it("returns an unlock function from lock()", async () => {
        const m = new Mutex();
        const unlock = await m.lock();
        expect(typeof unlock).toBe("function");
        unlock();
    });

    it("allows a second lock after the first is released", async () => {
        const m = new Mutex();
        const unlock1 = await m.lock();
        unlock1();
        const unlock2 = await m.lock();
        expect(typeof unlock2).toBe("function");
        unlock2();
    });

    it("serializes concurrent acquisitions", async () => {
        const m = new Mutex();
        const order: number[] = [];

        const run = async (id: number) => {
            const unlock = await m.lock();
            order.push(id);
            // Yield to give other waiters a chance to interleave (they shouldn't).
            await Promise.resolve();
            order.push(id);
            unlock();
        };

        await Promise.all([run(1), run(2), run(3)]);

        // Each id should appear consecutively (1,1,2,2,3,3) — never interleaved.
        for (let i = 0; i < order.length; i += 2) {
            expect(order[i]).toBe(order[i + 1]);
        }
    });

    it("preserves FIFO order among waiters", async () => {
        const m = new Mutex();
        const finished: number[] = [];

        const unlock1 = await m.lock();
        const p2 = m.lock().then((unlock) => {
            finished.push(2);
            unlock();
        });
        const p3 = m.lock().then((unlock) => {
            finished.push(3);
            unlock();
        });
        const p4 = m.lock().then((unlock) => {
            finished.push(4);
            unlock();
        });

        unlock1();
        await Promise.all([p2, p3, p4]);

        expect(finished).toEqual([2, 3, 4]);
    });

    it("does not deadlock if unlock is called when no waiters are queued", async () => {
        const m = new Mutex();
        const unlock = await m.lock();
        unlock();
        const next = await m.lock();
        next();
    });

    it("isolates separate Mutex instances", async () => {
        const a = new Mutex();
        const b = new Mutex();
        const unlockA = await a.lock();
        const unlockB = await b.lock();
        unlockA();
        unlockB();
    });
});

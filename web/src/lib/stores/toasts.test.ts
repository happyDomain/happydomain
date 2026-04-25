import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

import { toasts } from "./toasts";

function reset() {
    // The toasts store has no clear() method; dismiss every entry instead.
    for (const t of [...get(toasts)]) toasts.dismiss(t.id);
}

describe("toasts store", () => {
    beforeEach(reset);

    it("starts empty", () => {
        expect(get(toasts)).toEqual([]);
    });

    it("addToast prepends to the list (newest first)", () => {
        toasts.addToast({ title: "first", message: "1" });
        toasts.addToast({ title: "second", message: "2" });
        const list = get(toasts);
        expect(list).toHaveLength(2);
        expect(list[0].title).toBe("second");
        expect(list[1].title).toBe("first");
    });

    it("addToast preserves an explicit type", () => {
        toasts.addToast({ type: "success", message: "yay" });
        expect(get(toasts)[0].type).toBe("success");
    });

    it("addToast defaults type to info when not set", () => {
        toasts.addToast({ message: "no type given" });
        expect(get(toasts)[0].type).toBe("info");
    });

    it("addErrorToast defaults to type 'error' and a default title", () => {
        toasts.addErrorToast({ message: "something exploded" });
        const t = get(toasts)[0];
        expect(t.type).toBe("error");
        expect(t.title).toBe("An error occured!");
    });

    it("addErrorToast preserves a caller-supplied title", () => {
        toasts.addErrorToast({ title: "Custom error", message: "x" });
        const t = get(toasts)[0];
        expect(t.title).toBe("Custom error");
        expect(t.type).toBe("error");
    });

    it("dismiss removes only the toast with the matching id", () => {
        toasts.addToast({ title: "a" });
        toasts.addToast({ title: "b" });
        const [b, a] = get(toasts);
        toasts.dismiss(a.id);
        const remaining = get(toasts);
        expect(remaining).toHaveLength(1);
        expect(remaining[0].id).toBe(b.id);
    });

    it("dismiss is a no-op for an unknown id", () => {
        toasts.addToast({ title: "a" });
        const before = get(toasts).length;
        toasts.dismiss("not-a-real-id");
        expect(get(toasts).length).toBe(before);
    });

    it("each new toast has a unique id", () => {
        toasts.addToast({ title: "1" });
        toasts.addToast({ title: "2" });
        toasts.addToast({ title: "3" });
        const ids = get(toasts).map((t) => t.id);
        expect(new Set(ids).size).toBe(3);
    });
});

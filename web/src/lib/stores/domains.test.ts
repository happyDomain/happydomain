import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";

vi.mock("$lib/api/domains", () => ({
    listDomains: vi.fn(),
}));

import {
    domains,
    groups,
    domains_idx,
    domains_by_name,
    domains_by_groups,
    domainLink,
    refreshDomains,
} from "./domains";
import { listDomains } from "$lib/api/domains";
import type { HappydnsDomainWithCheckStatus } from "$lib/api-base/types.gen";

const mockListDomains = vi.mocked(listDomains);

function d(
    id: string,
    domain: string,
    group: string | undefined = "",
): HappydnsDomainWithCheckStatus {
    return {
        id,
        id_owner: "u1",
        id_provider: "p1",
        domain,
        group: group as string,
        zone_history: [],
    } as HappydnsDomainWithCheckStatus;
}

describe("domains store", () => {
    beforeEach(() => {
        domains.set(undefined);
        mockListDomains.mockReset();
    });

    it("starts undefined", () => {
        expect(get(domains)).toBeUndefined();
    });

    it("groups derives [] from undefined domains", () => {
        expect(get(groups)).toEqual([]);
    });

    it("groups derives the unique sorted group set", () => {
        domains.set([
            d("1", "a.example.com", "alpha"),
            d("2", "b.example.com", "beta"),
            d("3", "c.example.com", "alpha"),
        ]);
        expect(get(groups)).toEqual(["alpha", "beta"]);
    });

    it("groups places the empty/no-group entry last", () => {
        domains.set([
            d("1", "a.example.com", "zebra"),
            d("2", "b.example.com", ""),
            d("3", "c.example.com", "alpha"),
        ]);
        expect(get(groups)).toEqual(["alpha", "zebra", ""]);
    });
});

describe("domains_idx", () => {
    beforeEach(() => domains.set(undefined));

    it("returns an empty object when domains is undefined", () => {
        expect(get(domains_idx)).toEqual({});
    });

    it("indexes entries by id and by unique domain name", () => {
        const a = d("1", "alpha.example.com");
        const b = d("2", "beta.example.com");
        domains.set([a, b]);
        const idx = get(domains_idx);
        expect(idx["1"]).toBe(a);
        expect(idx["2"]).toBe(b);
        expect(idx["alpha.example.com"]).toBe(a);
        expect(idx["beta.example.com"]).toBe(b);
    });

    it("removes the domain-name key when two records share the same domain (multiview)", () => {
        const a = d("1", "shared.example.com");
        const b = d("2", "shared.example.com");
        domains.set([a, b]);
        const idx = get(domains_idx);
        expect(idx["1"]).toBe(a);
        expect(idx["2"]).toBe(b);
        // Disambiguated: the name key is gone so callers must use the id.
        expect(idx["shared.example.com"]).toBeUndefined();
    });
});

describe("domains_by_name", () => {
    beforeEach(() => domains.set(undefined));

    it("returns an empty object when domains is undefined", () => {
        expect(get(domains_by_name)).toEqual({});
    });

    it("groups records by their domain name", () => {
        const a = d("1", "shared.example.com");
        const b = d("2", "shared.example.com");
        const c = d("3", "other.example.com");
        domains.set([a, b, c]);
        const idx = get(domains_by_name);
        expect(idx["shared.example.com"]).toEqual([a, b]);
        expect(idx["other.example.com"]).toEqual([c]);
    });
});

describe("domains_by_groups", () => {
    beforeEach(() => domains.set(undefined));

    it("returns an empty object when domains is undefined", () => {
        expect(get(domains_by_groups)).toEqual({});
    });

    it("buckets records by their group", () => {
        domains.set([
            d("1", "a.example.com", "alpha"),
            d("2", "b.example.com", "alpha"),
            d("3", "c.example.com", "beta"),
            d("4", "d.example.com", ""),
        ]);
        const g = get(domains_by_groups);
        expect(g.alpha?.map((x) => x.id)).toEqual(["1", "2"]);
        expect(g.beta?.map((x) => x.id)).toEqual(["3"]);
        expect(g[""]?.map((x) => x.id)).toEqual(["4"]);
    });
});

describe("domainLink", () => {
    beforeEach(() => domains.set(undefined));

    it("returns the domain name for a known id when the name resolves uniquely", () => {
        const a = d("abc", "alpha.example.com");
        domains.set([a]);
        expect(domainLink("abc")).toBe("alpha.example.com");
    });

    it("returns the id when the domain name has been multiview-disambiguated", () => {
        domains.set([d("1", "shared.example.com"), d("2", "shared.example.com")]);
        expect(domainLink("1")).toBe("1");
    });

    it("returns the original id when no entry exists for it", () => {
        domains.set([d("abc", "alpha.example.com")]);
        expect(domainLink("missing")).toBe("missing");
    });
});

describe("refreshDomains", () => {
    beforeEach(() => {
        mockListDomains.mockReset();
    });

    it("fetches via listDomains and stores the result", async () => {
        const data = [d("1", "alpha.example.com", "g1")];
        mockListDomains.mockResolvedValueOnce(data);

        const result = await refreshDomains();

        expect(result).toBe(data);
        expect(get(domains)).toEqual(data);
    });

    it("normalizes a missing group to an empty string", async () => {
        const raw = [{ ...d("1", "x.example.com"), group: undefined } as never];
        mockListDomains.mockResolvedValueOnce(raw);

        await refreshDomains();
        const stored = get(domains);
        expect(stored?.[0].group).toBe("");
    });
});

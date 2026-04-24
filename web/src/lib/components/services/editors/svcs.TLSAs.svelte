<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();

    if (!value["tlsa"]) {
        value["tlsa"] = [] as any;
    }
    const records = (): dnsRR[] => value["tlsa"] as any as dnsRR[];

    // Port + protocol are encoded in the owner name ("_443._tcp.host"). Parse
    // them out of the first record so an existing service opens with the
    // matching form values.
    let port = $state<number>(443);
    let protocol = $state<"tcp" | "udp">("tcp");
    if (records()?.[0]?.Hdr?.Name) {
        const m = records()[0].Hdr.Name.match(/^_(\d+)\._(tcp|udp)/);
        if (m) {
            port = parseInt(m[1], 10);
            protocol = m[2] as "tcp" | "udp";
        }
    }

    const fullDn = $derived(`_${port}._${protocol}`);

    $effect(() => {
        for (const r of records()) {
            if (r?.Hdr) r.Hdr.Name = fullDn;
        }
    });

    // Numeric values are the RFC 6698 TLSA fields.
    const USAGE = [
        {
            v: 3,
            label: "DANE-EE — End entity (most common)",
            hint: "TLSA hash matches the leaf certificate. No CA is required; this profile is recommended for Let's Encrypt with SPKI pinning.",
        },
        {
            v: 2,
            label: "DANE-TA — Trust anchor",
            hint: "TLSA hash matches a CA in the chain you run. PKIX validation is not required.",
        },
        {
            v: 1,
            label: "PKIX-EE — End entity + PKIX",
            hint: "Like DANE-EE but the chain must also validate through public trust roots.",
        },
        {
            v: 0,
            label: "PKIX-TA — Trust anchor + PKIX",
            hint: "Like DANE-TA plus PKIX validation. Rarely used.",
        },
    ];
    const SELECTOR = [
        {
            v: 1,
            label: "SPKI — Public key only (recommended)",
            hint: "Matches the Subject Public Key Info. Survives cert renewals that keep the same key pair.",
        },
        {
            v: 0,
            label: "Cert — Full certificate",
            hint: "Matches the whole certificate. You must rotate the TLSA at every cert renewal.",
        },
    ];
    const MATCHING = [
        {
            v: 1,
            label: "SHA-256 (recommended)",
            hint: "32-byte hash (64 hex chars). Universally supported.",
        },
        {
            v: 2,
            label: "SHA-512",
            hint: "64-byte hash. Stronger; same guarantees as SHA-256 in practice.",
        },
        {
            v: 0,
            label: "Full / exact",
            hint: "Match the raw bytes. Produces very long records; use only when a hash is not acceptable.",
        },
    ];

    const PRESETS = [
        {
            id: "le-dane-ee-spki",
            label: "Let's Encrypt / DANE-EE · SPKI · SHA-256",
            u: 3,
            s: 1,
            m: 1,
        },
        { id: "pkix-ee-spki", label: "Public CA / PKIX-EE · SPKI · SHA-256", u: 1, s: 1, m: 1 },
        {
            id: "dane-ta-spki",
            label: "Self-hosted CA / DANE-TA · SPKI · SHA-256",
            u: 2,
            s: 1,
            m: 1,
        },
    ];

    function addRecord() {
        const r = {
            Hdr: { Name: fullDn, Rrtype: 52, Class: 1, Ttl: 3600, Rdlength: 0 },
            Usage: 3,
            Selector: 1,
            MatchingType: 1,
            Certificate: "",
        } as unknown as dnsRR;
        records().push(r);
    }
    function removeRecord(i: number) {
        records().splice(i, 1);
    }
    function applyPreset(i: number, id: string) {
        const p = PRESETS.find((x) => x.id === id);
        if (!p) return;
        const r = records()[i];
        r.Usage = p.u;
        r.Selector = p.s;
        r.MatchingType = p.m;
    }

    let fetching = $state<number | null>(null);
    let errorMsg = $state<string>("");

    async function fetchLive(i: number) {
        fetching = i;
        errorMsg = "";
        try {
            const host = dn || origin.domain;
            const res = await fetch(`/api/domains/${origin.id}/fetch-certificate`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ host, port, proto: protocol }),
            });
            if (!res.ok) {
                const body = await res.json().catch(() => ({}));
                throw new Error(body.errmsg || `HTTP ${res.status}`);
            }
            const data = await res.json();
            const r = records()[i];
            // Usage 1/3 are end-entity, so the leaf cert (chain[0]); 0/2 are
            // trust-anchor, so the next link up if the chain has one.
            const slot = r.Usage === 1 || r.Usage === 3 ? 0 : Math.min(1, data.chain.length - 1);
            const c = data.chain[slot];
            r.Certificate = pickHash(c, r.Selector, r.MatchingType);
        } catch (e: any) {
            errorMsg = `Fetch failed: ${e.message || e}`;
        } finally {
            fetching = null;
        }
    }

    function pickHash(c: any, selector: number, matching: number): string {
        // Matching=0 (Full) expects hex of raw DER.
        const b64 = selector === 1 ? c.spki_der_base64 : c.der_base64;
        if (matching === 0) return hexFromBase64(b64);
        if (selector === 1) return matching === 2 ? c.spki_sha512 : c.spki_sha256;
        return matching === 2 ? c.cert_sha512 : c.cert_sha256;
    }

    function hexFromBase64(b64: string): string {
        const bin = atob(b64);
        let out = "";
        for (let i = 0; i < bin.length; i++) out += bin.charCodeAt(i).toString(16).padStart(2, "0");
        return out;
    }

    // Accepts PEM (one or more "-----BEGIN CERTIFICATE-----" blocks) or raw
    // DER. We compute the hash client-side so the user's certificate file
    // never leaves the browser.
    async function onUpload(i: number, ev: Event) {
        const input = ev.target as HTMLInputElement;
        const f = input.files?.[0];
        if (!f) return;
        try {
            const buf = new Uint8Array(await f.arrayBuffer());
            const der = isPEM(buf) ? firstPEMBlock(buf) : buf;
            const r = records()[i];
            const target = r.Selector === 1 ? extractSPKI(der) : der;
            const hash = await hashBytes(target, r.MatchingType);
            r.Certificate = hash;
        } catch (e: any) {
            errorMsg = `Upload failed: ${e.message || e}`;
        } finally {
            input.value = "";
        }
    }

    function isPEM(b: Uint8Array): boolean {
        const head = new TextDecoder().decode(b.slice(0, 27));
        return head.startsWith("-----BEGIN");
    }
    function firstPEMBlock(b: Uint8Array): Uint8Array {
        const text = new TextDecoder().decode(b);
        const m = text.match(/-----BEGIN [^-]+-----([\s\S]+?)-----END /);
        if (!m) throw new Error("No PEM block found");
        const b64 = m[1].replace(/\s+/g, "");
        const bin = atob(b64);
        const out = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
        return out;
    }
    // Walks Certificate → TBSCertificate → SubjectPublicKeyInfo. Strict (no
    // recovery on malformed DER) — the user can always upload again or paste
    // the hash by hand.
    function extractSPKI(der: Uint8Array): Uint8Array {
        const seq = readTLV(der, 0);
        const tbs = readTLV(der, seq.content);
        let p = tbs.content;
        if (der[p] === 0xa0) {
            // Skip optional [0] version tag
            p = readTLV(der, p).next;
        }
        p = readTLV(der, p).next; // serial
        p = readTLV(der, p).next; // signature AlgorithmIdentifier
        p = readTLV(der, p).next; // issuer
        p = readTLV(der, p).next; // validity
        p = readTLV(der, p).next; // subject
        const spki = readTLV(der, p);
        return der.slice(p, spki.next);
    }
    function readTLV(b: Uint8Array, o: number): { content: number; next: number } {
        o++; // tag
        let len = b[o++];
        if (len & 0x80) {
            const n = len & 0x7f;
            len = 0;
            for (let i = 0; i < n; i++) len = (len << 8) | b[o++];
        }
        return { content: o, next: o + len };
    }
    async function hashBytes(b: Uint8Array, matching: number): Promise<string> {
        if (matching === 0) {
            let out = "";
            for (let i = 0; i < b.length; i++) out += b[i].toString(16).padStart(2, "0");
            return out;
        }
        const algo = matching === 2 ? "SHA-512" : "SHA-256";
        const h = new Uint8Array(await crypto.subtle.digest(algo, b as BufferSource));
        let out = "";
        for (let i = 0; i < h.length; i++) out += h[i].toString(16).padStart(2, "0");
        return out;
    }
</script>

<div class="d-flex flex-column gap-3">
    <div class="row g-3 align-items-end">
        <div class="col-sm-3">
            <label for="tlsa-port" class="form-label fw-semibold mb-1">Service Port</label>
            <input
                id="tlsa-port"
                class="form-control form-control-sm"
                type="number"
                min="1"
                max="65535"
                bind:value={port}
                disabled={readonly}
            />
        </div>
        <div class="col-sm-3">
            <label for="tlsa-proto" class="form-label fw-semibold mb-1">Protocol</label>
            <select
                id="tlsa-proto"
                class="form-select form-select-sm"
                bind:value={protocol}
                disabled={readonly}
            >
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
            </select>
        </div>
        <div class="col-sm-6 text-muted">
            <small>TLSA owner: <code class="bg-light px-1 rounded">{fullDn}.{dn || origin.domain}</code></small>
        </div>
    </div>

    {#if errorMsg}
        <div class="alert alert-danger py-2 mb-0" role="alert">{errorMsg}</div>
    {/if}

    {#each records() as rec, i}
        <fieldset class="border rounded p-3">
            <legend class="float-none w-auto px-2 fs-6 text-secondary">Record #{i + 1}</legend>

            <div class="mb-3" style="max-width: 28rem;">
                <label for="tlsa-preset-{i}" class="form-label fw-semibold mb-1">Preset</label>
                <select
                    id="tlsa-preset-{i}"
                    class="form-select form-select-sm"
                    disabled={readonly}
                    onchange={(e) => applyPreset(i, (e.currentTarget as HTMLSelectElement).value)}
                >
                    <option value="">— Choose a preset —</option>
                    {#each PRESETS as p}
                        <option value={p.id}>{p.label}</option>
                    {/each}
                </select>
            </div>

            <div class="row g-3">
                <div class="col-md-4">
                    <label for="tlsa-usage-{i}" class="form-label fw-semibold mb-1">Certificate usage</label>
                    <select
                        id="tlsa-usage-{i}"
                        class="form-select form-select-sm"
                        bind:value={rec.Usage}
                        disabled={readonly}
                    >
                        {#each USAGE as u}
                            <option value={u.v}>{u.v} — {u.label}</option>
                        {/each}
                    </select>
                    <small class="form-text text-muted">{USAGE.find((x) => x.v === rec.Usage)?.hint || ""}</small>
                </div>
                <div class="col-md-4">
                    <label for="tlsa-sel-{i}" class="form-label fw-semibold mb-1">Selector</label>
                    <select
                        id="tlsa-sel-{i}"
                        class="form-select form-select-sm"
                        bind:value={rec.Selector}
                        disabled={readonly}
                    >
                        {#each SELECTOR as s}
                            <option value={s.v}>{s.v} — {s.label}</option>
                        {/each}
                    </select>
                    <small class="form-text text-muted">{SELECTOR.find((x) => x.v === rec.Selector)?.hint || ""}</small>
                </div>
                <div class="col-md-4">
                    <label for="tlsa-mt-{i}" class="form-label fw-semibold mb-1">Matching type</label>
                    <select
                        id="tlsa-mt-{i}"
                        class="form-select form-select-sm"
                        bind:value={rec.MatchingType}
                        disabled={readonly}
                    >
                        {#each MATCHING as m}
                            <option value={m.v}>{m.v} — {m.label}</option>
                        {/each}
                    </select>
                    <small class="form-text text-muted">{MATCHING.find((x) => x.v === rec.MatchingType)?.hint || ""}</small>
                </div>
            </div>

            <div class="mt-3">
                <label for="tlsa-cert-{i}" class="form-label fw-semibold mb-1">Certificate data (hex)</label>
                <textarea
                    id="tlsa-cert-{i}"
                    class="form-control font-monospace small"
                    rows="3"
                    bind:value={rec.Certificate}
                    disabled={readonly}
                    spellcheck="false"
                    placeholder="Paste a hex-encoded hash, or use the buttons below to compute it."
                ></textarea>
            </div>

            {#if !readonly}
                <div class="d-flex flex-wrap gap-2 mt-3 align-items-center">
                    <button
                        type="button"
                        class="btn btn-sm btn-outline-secondary"
                        onclick={() => fetchLive(i)}
                        disabled={fetching !== null}
                    >
                        {fetching === i ? "Fetching…" : "Fetch from live server"}
                    </button>
                    <label class="btn btn-sm btn-outline-secondary mb-0">
                        <input
                            type="file"
                            accept=".pem,.crt,.cer,.der"
                            onchange={(e) => onUpload(i, e)}
                            hidden
                        />
                        Upload certificate (PEM/DER)
                    </label>
                    <button
                        type="button"
                        class="btn btn-sm btn-outline-danger ms-auto"
                        onclick={() => removeRecord(i)}
                    >
                        Remove
                    </button>
                </div>
            {/if}
        </fieldset>
    {/each}

    {#if !readonly}
        <button type="button" class="btn btn-sm btn-outline-primary align-self-start" onclick={addRecord}>
            + Add TLSA record
        </button>
    {/if}
</div>

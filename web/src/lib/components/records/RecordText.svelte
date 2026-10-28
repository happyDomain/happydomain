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
    import type { ClassValue } from "svelte/elements";

    import { fqdn, nsttl, printRR } from "$lib/dns";
    import { nsrrtype, rdatatostr, type dnsRR } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    interface Props {
        class?: ClassValue;
        dn: string;
        origin: Domain;
        rr: dnsRR;
    }

    let { class: className, dn, origin, rr }: Props = $props();

    // The name of the record, relative to the zone: "@" for its apex.
    let name = $derived.by(() => {
        const full = fqdn(rr.Hdr.Name || "@", fqdn(dn, origin.domain));
        if (full === origin.domain) return "@";
        return full.endsWith("." + origin.domain) ? full.slice(0, -origin.domain.length - 1) : full;
    });
</script>

<div
    class={["record-text d-flex align-items-center min-w-0 font-monospace", className]}
    title={printRR(rr, dn, origin.domain)}
>
    <span class="record-type d-inline-block flex-shrink-0 fw-medium text-center"
        >{nsrrtype(rr.Hdr.Rrtype)}</span
    >
    <span class="record-name min-w-0 text-truncate">{name}</span>
    <span class="record-value min-w-0 text-truncate">{rdatatostr(rr)}</span>
    {#if rr.Hdr.Ttl}
        <span class="record-ttl flex-shrink-0" title={`${rr.Hdr.Ttl}s`}
            >{$t("service.ttl")} {nsttl(rr.Hdr.Ttl)}</span
        >
    {/if}
</div>

<style>
    .record-text {
        gap: 0.75rem;
        color: var(--hd-fg-2);
    }

    .record-type {
        min-width: 2.75rem;
        padding: 1px 6px;
        border-radius: var(--hd-radius-xs);
        background: var(--hd-bg-inset);
        color: var(--hd-fg-2);
        font-size: 0.625rem;
        letter-spacing: 0.04em;
    }

    .record-name {
        max-width: 35%;
        color: var(--hd-fg-3);
    }

    .record-value {
        flex: 1;
        color: var(--hd-fg-1);
    }

    .record-ttl {
        font-size: var(--hd-text-xs);
        color: var(--hd-fg-4);
    }
</style>

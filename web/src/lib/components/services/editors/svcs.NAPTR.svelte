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
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "svcs.NAPTR";

    // Initialize NAPTR record if needed
    if (!value["naptr"]) {
        value["naptr"] = newRR("", getRrtype("NAPTR")) as import("$lib/dns_rr").dnsTypeNAPTR;
    }
</script>

<div>
    <RecordLine {dn} {origin} bind:rr={value["naptr"]!} />

    <BasicInput
        class="mt-3"
        edit
        index="order"
        specs={{
            id: "order",
            label: "Order",
            description: "Order in which NAPTR records are processed",
            type: "uint16",
        }}
        bind:value={value["naptr"]!.Order}
    />

    <BasicInput
        edit
        index="preference"
        specs={{
            id: "preference",
            label: "Preference",
            description: "Preference for records with the same order",
            type: "uint16",
        }}
        bind:value={value["naptr"]!.Preference}
    />

    <BasicInput
        edit
        index="flags"
        specs={{
            id: "flags",
            label: "Flags",
            description: "Control aspects of the rewriting and interpretation",
            type: "string",
            placeholder: "S, A, U, P",
        }}
        bind:value={value["naptr"]!.Flags}
    />

    <BasicInput
        edit
        index="service"
        specs={{
            id: "service",
            label: "Service",
            description: "Service parameters",
            type: "string",
            placeholder: "E2U+sip",
        }}
        bind:value={value["naptr"]!.Service}
    />

    <BasicInput
        edit
        index="regexp"
        specs={{
            id: "regexp",
            label: "Regular Expression",
            description: "Substitution expression applied to the original string",
            type: "string",
        }}
        bind:value={value["naptr"]!.Regexp}
    />

    <BasicInput
        edit
        index="replacement"
        specs={{
            id: "replacement",
            label: "Replacement",
            description: "Next domain name to query",
            type: "string",
            placeholder: "example.com.",
        }}
        bind:value={value["naptr"]!.Replacement}
    />
</div>

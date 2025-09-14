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
    import type { dnsResource, dnsTypeCNAME } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import { getRrtype, newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "svcs.CNAME";

    // Initialize CNAME record if needed
    if (!value["cname"]) {
        value["cname"] = newRR("", getRrtype("CNAME")) as dnsTypeCNAME;
    }

    // Type-safe accessor for the CNAME record
    let cnameRecord = $derived(value["cname"] as dnsTypeCNAME);
</script>

<div>
    <RecordLine {dn} {origin} bind:rr={value["cname"]!} />
    <BasicInput
        class="mt-3"
        edit
        index="target"
        specs={{
            id: "target",
            label: "Target",
            description: "The canonical name this CNAME points to",
            type: "string",
            placeholder: "example.com.",
        }}
        bind:value={cnameRecord.Target}
    />
</div>

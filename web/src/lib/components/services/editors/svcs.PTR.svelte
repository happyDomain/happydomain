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
    import type { dnsResource, dnsTypePTR } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "svcs.PTR";

    // Initialize PTR record if needed
    if (!value["ptr"]) {
        value["ptr"] = newRR("", getRrtype("PTR")) as dnsTypePTR;
    }
</script>

<div>
    <RecordLine {dn} {origin} bind:rr={value["ptr"]!} />
    <BasicInput
        class="mt-3"
        edit
        index="ptr"
        specs={{
            id: "ptr",
            label: "Pointer",
            description: "The domain name this PTR record points to",
            type: "string",
            placeholder: "example.com.",
        }}
        bind:value={value["ptr"]!.Ptr}
    />
</div>

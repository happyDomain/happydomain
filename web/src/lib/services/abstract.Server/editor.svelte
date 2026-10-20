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
    import { Input } from "@sveltestrap/sveltestrap";

    import { getRrtype, newRR } from "$lib/dns_rr";
    import type { dnsTypeA, dnsTypeAAAA } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import type { AbstractServerBody } from "$lib/services_bodies";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import SSHFPEditor from "$lib/components/services/SSHFPEditor.svelte";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: AbstractServerBody;
    }

    let { dn, origin, readonly = false, value = $bindable() }: Props = $props();
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Server Connectivity (A/AAA records)</h4>

    {#if value["A"]}
        <BasicInput
            edit
            index="A"
            specs={{
                id: "A",
                label: "IPv4",
                type: "net.IP",
            }}
            bind:value={value["A"].A}
        />
    {:else}
        <Input
            onclick={() => (value["A"] = newRR("", getRrtype("A")) as dnsTypeA)}
            oninput={() => (value["A"] = newRR("", getRrtype("A")) as dnsTypeA)}
        />
    {/if}
    {#if value["AAAA"]}
        <BasicInput
            edit
            index="AAAA"
            specs={{
                id: "AAAA",
                label: "IPv6",
                type: "net.IP",
            }}
            bind:value={value["AAAA"].AAAA}
        />
    {:else}
        <Input
            label="test"
            onclick={() => (value["AAAA"] = newRR("", getRrtype("AAAA")) as dnsTypeAAAA)}
            oninput={() => (value["AAAA"] = newRR("", getRrtype("AAAA")) as dnsTypeAAAA)}
        />
    {/if}
</div>
<hr />
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">
        SSH Fingerprint
        <small class="text-muted">Server's SSH fingerprint</small>
    </h4>
    <SSHFPEditor {dn} {origin} {readonly} bind:value />
</div>

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
    import type { dnsTypeA, dnsTypeAAAA, dnsTypeSSHFP } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import type { AbstractServerBody } from "$lib/services_bodies";
    import TableRecords from "$lib/components/records/TableRecords.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import RawInput from "$lib/components/inputs/raw.svelte";

    interface Props {
        dn: string;
        origin: Domain;
        value: AbstractServerBody;
    }

    let { dn, origin, value = $bindable() }: Props = $props();

    // Read back through the body on each access: the state proxy hands out the
    // reactive array, which a plain assignment expression would not.
    if (!value.SSHFP) value.SSHFP = [];
    const fingerprints = (): dnsTypeSSHFP[] => value.SSHFP ?? [];

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
            onclick={() => (value["A"] = newRR(dn, getRrtype("A")) as dnsTypeA)}
            oninput={() => (value["A"] = newRR(dn, getRrtype("A")) as dnsTypeA)}
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
            onclick={() => (value["AAAA"] = newRR(dn, getRrtype("AAAA")) as dnsTypeAAAA)}
            oninput={() => (value["AAAA"] = newRR(dn, getRrtype("AAAA")) as dnsTypeAAAA)}
        />
    {/if}
</div>
<hr />
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">
        SSH Fingerprint
        <small class="text-muted">Server's SSH fingerprint</small>
    </h4>
    <!--RecordsLines {dn} {origin} bind:rrs={value["ns"]} /-->
    <TableRecords class="mt-3" {dn} edit rrs={fingerprints()} rrtype="SSHFP">
        {#snippet header(field: string)}
            {#if field == "Algorithm"}
                Algorithm
            {:else if field == "Type"}
                Type
            {:else if field == "FingerPrint"}
                Fingerprint
            {/if}
        {/snippet}
        {#snippet field(idx: number, field: string)}
            {@const rr = fingerprints()[idx] as Record<string, any>}
            {#if field == "Algorithm"}
                <RawInput
                    edit
                    index={"SSHFP-" + idx.toString()}
                    specs={{
                        id: "algorithm",
                        type: "uint",
                    }}
                    bind:value={rr[field]}
                />
            {:else if field == "Type"}
                <RawInput
                    edit
                    index={"SSHFP-" + idx.toString()}
                    specs={{
                        id: "type",
                        type: "uint",
                    }}
                    bind:value={rr[field]}
                />
            {:else if field == "FingerPrint"}
                <RawInput edit index={"SSHFP-" + idx.toString()} bind:value={rr[field]} />
            {/if}
        {/snippet}
    </TableRecords>
</div>

<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2020-2026 happyDomain
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
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import TableRecords from "$lib/components/records/TableRecords.svelte";
    import RawInput from "$lib/components/inputs/raw.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { dnsResource, dnsTypeMX, dnsTypeTXT } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import { servicesSpecs } from "$lib/stores/services";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource & { validationMX?: dnsTypeMX; };
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "google.GSuite";

    // Ensure mx is always an array at runtime
    $effect(() => {
        if (value["mx"] && !Array.isArray(value["mx"])) {
            value["mx"] = [value["mx"]];
        }
    });

    // Extract validation code from ValidationMX record
    let validationCode = $state(
        value["validationMX"]?.Mx?.replace(/\.mx-verification\.google\.com\.?$/, "") || ""
    );

    // Sync validation code to ValidationMX record
    $effect(() => {
        if (validationCode && validationCode.trim() !== "") {
            // Create ValidationMX record if it doesn't exist
            if (!value["validationMX"]) {
                value["validationMX"] = newRR(dn, getRrtype("MX")) as dnsTypeMX;
                value["validationMX"].Preference = 15;
            }
            // Update the Mx field with proper formatting
            const cleanCode = validationCode.trim().replace(/\.mx-verification\.google\.com\.?$/, "");
            value["validationMX"].Mx = cleanCode + ".mx-verification.google.com.";
        } else if (value["validationMX"] && (!validationCode || validationCode.trim() === "")) {
            // Remove ValidationMX if validation code is empty
            delete value["validationMX"];
        }
    });
</script>

{#if $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}

<div>
    <div class="alert alert-info mb-3">
        <strong>G Suite / Google Workspace Configuration</strong>
        <p class="mb-0">
            This service configures MX records for Google's mail servers and SPF directives.
            The validation MX record is optional and only needed during initial domain setup.
        </p>
    </div>

    <!-- Validation MX Record -->
    <div class="mb-4">
        <h4 class="text-primary pb-1 border-bottom border-1">Validation MX Record (Optional)</h4>
        <p class="text-muted small">
            This verification record is only needed during initial Google domain setup and can be removed after verification.
        </p>
        <BasicInput
            class="mt-3"
            edit={!readonly}
            index="validation-code"
            specs={{
                id: "validation-code",
                label: "Validation Code",
                description: "Enter the verification code from Google (e.g., abcdef0123)",
                type: "string",
                placeholder: "abcdef0123",
            }}
            bind:value={validationCode}
        />
        {#if value["validationMX"]}
        <div class="mt-3">
            <RecordLine {dn} {origin} bind:rr={value["validationMX"]!} />
        </div>
        {/if}
    </div>

    <!-- MX Records -->
    {#if value["mx"]}
    <div class="mb-4">
        <h4 class="text-primary pb-1 border-bottom border-1">Google MX Records</h4>
        <TableRecords
            class="mt-3"
            {dn}
            edit={!readonly}
            {origin}
            rrs={value["mx"] as dnsTypeMX[]}
            rrtype="MX"
        >
            {#snippet header(field: string)}
                {#if field == "Mx"}
                    Mail Server
                {:else if field == "Preference"}
                    Priority
                {/if}
            {/snippet}
            {#snippet field(idx: number, field: string)}
                {#if value["mx"] && (value["mx"] as dnsTypeMX[])[idx]}
                    {#if field == "Preference"}
                        <RawInput
                            edit={!readonly}
                            index={field + idx.toString()}
                            specs={{
                                  id: "preference",
                                  type: "uint",
                            }}
                            bind:value={(value["mx"] as dnsTypeMX[])[idx].Preference}
                        />
                    {:else if field == "Mx"}
                        <RawInput
                            edit={!readonly}
                            index={field + idx.toString()}
                            bind:value={(value["mx"] as dnsTypeMX[])[idx].Mx}
                        />
                    {/if}
                {/if}
            {/snippet}
        </TableRecords>
    </div>
    {/if}

    <!-- SPF TXT Record -->
    {#if value["txt"]}
    <div class="mb-4">
        <h4 class="text-primary pb-1 border-bottom border-1">SPF Record</h4>
        <RecordLine {dn} {origin} bind:rr={value["txt"]!} />
    </div>
    {/if}
</div>

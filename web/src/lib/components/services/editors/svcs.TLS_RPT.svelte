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
    import { Button, Icon, Input } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import type { dnsResource, dnsTypeTXT } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import { t } from "$lib/translations";
    import { TLSRPTPolicy } from "$lib/services/tlsrpt.svelte";

    interface Props {
        dn: string;
        origin: Domain;
        value: dnsResource;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    // Initialize TXT record if it doesn't exist
    $effect(() => {
        if (!value["txt"]) {
            value["txt"] = newRR(dn, getRrtype("TXT")) as dnsTypeTXT;
        }
    });

    // svelte-ignore state_referenced_locally
    let val = $derived(value["txt"] ? new TLSRPTPolicy(value["txt"]) : new TLSRPTPolicy({ Hdr: { Name: dn, Rrtype: 16, Class: 1, Ttl: 3600, Rdlength: 0 }, Txt: "" }));

    const type = "svcs.TLS_RPT";
</script>

{#if $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Aggregate Report URI</h4>
    {#if value["txt"]}
        <RecordLine class="mb-4" {dn} {origin} bind:rr={value["txt"]} />
    {/if}

    <BasicInput
        edit
        index="v"
        specs={{
              id: "v",
              label: "Version",
              placeholder: "TLSRPTv1",
              type: "string",
              description: "Defines the version of TLSRPT to use",
              }}
        bind:value={val.v}
    />

    <table class="table table-striped table-hover">
        <thead>
            <tr>
                <th>Aggregate Report URI</th>
            </tr>
        </thead>
        <tbody>
            {#if val.rua && val.rua.length}
                {#each val.rua as rua, idx}
                    <tr>
                        <td>
                            <Input
                                bsSize="sm"
                                value={val.rua[idx]}
                                oninput={(e) => val.updateRua(idx, (e.target as HTMLInputElement).value)}
                            />
                        </td>
                        <td>
                            <Button
                                type="button"
                                color="danger"
                                outline
                                size="sm"
                                onclick={() => val.removeRua(idx)}
                            >
                                <Icon name="trash" />
                            </Button>
                        </td>
                    </tr>
                {/each}
                {:else}
                <tr>
                    <td
                        colspan={2}
                        class="fst-italic text-center"
                    >
                        {$t("common.no-content")}
                    </td>
                </tr>
            {/if}
        </tbody>
        <tfoot>
            <tr>
                <td colspan="1">
                    <Button
                        type="button"
                        color="primary"
                        outline
                        size="sm"
                        onclick={() => val.addRua("")}
                    >
                        <Icon name="plus" />
                        {$t("common.new-row")}
                    </Button>
                </td>
            </tr>
        </tfoot>
    </table>
</div>

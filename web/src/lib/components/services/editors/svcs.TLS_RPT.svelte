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
    import { parseKeyValueTxt } from "$lib/dns";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        value: any;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    function parseTLSRPT(val) {
        const kv = parseKeyValueTxt(val);

        kv.rua = kv.rua ? kv.rua.split(",") : [];

        return kv;
    }
    function stringifyTLSRPT(val) {
        const sep = (value["txt"].Txt.indexOf("; ") >= 0 ? "; " : ";");

        return "v=" + (val["v"] ? val["v"] : "TLSRPTv1") + (val["rua"] && val["rua"].length ? sep + "rua=" + val["rua"].join(",") : "");
    }
    let val = $state(parseTLSRPT(value["txt"].Txt));

    $effect(() => {
        val = parseTLSRPT(value["txt"].Txt);
    });
    $effect(() => {
        value["txt"].Txt = stringifyTLSRPT(val);
    });

    const type = "svcs.TLS_RPT";
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Aggregate Report URI</h4>
    <RecordLine class="mb-4" {dn} {origin} bind:rr={value["txt"]} />

    <BasicInput
        edit
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
                                size="sm"
                                bind:value={val.rua[idx]}
                            />
                        </td>
                        <td>
                            <Button
                                type="button"
                                color="danger"
                                outline
                                size="sm"
                                onclick={() => val.rua.splice(idx, 1)}
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
                        onclick={() => val.rua.push("")}
                    >
                        <Icon name="plus" />
                        {$t("common.new-row")}
                    </Button>
                </td>
            </tr>
        </tfoot>
    </table>
</div>

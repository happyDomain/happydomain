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
    import type { Snippet } from "svelte";

    import {
        Button,
        Icon,
        Table,
    } from "@sveltestrap/sveltestrap";
    import { printRR } from "$lib/dns";
    import { getRrtype, rdatafields, type dnsRR } from "$lib/dns_rr";
    import { controls } from "$lib/components/modals/Record.svelte";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    interface Props {
        class?: string;
        dn: string;
        edit: boolean;
        field: Snippet;
        header: Snippet;
        origin: Domain;
        rrs: Array<dnsRR>;
        rrtype: dnsRR;
    }

    let { class: className = "", dn, edit, field, header, origin, rrs, rrtype }: Props = $props();

    function addLine() {
        if (!rrs) rrs = [];

        if (rrs.length) {
            const newrr = JSON.parse(JSON.stringify(rrs[rrs.length-1]));
            for (const field of rdatafields(rrtype)) {
                newrr[field] = "";
            }
            rrs.push(newrr);
        } else {
            const newrr = {
                Hdr: {
                    Name: dn,
                    Rrtype: getRrtype(rrtype),
                    Class: 1,
                    Ttl: 3600,
                },
            };
            for (const field of rdatafields(rrtype)) {
                newrr[field] = "";
            }
            rrs.push(newrr);
        }

        rrs = rrs;
    }

    function deleteLine(idx: number) {
        rrs.splice(idx, 1);
        rrs = rrs;
    }

    function openEditor(rr: dnsRR) {
        controls.Open(rr, dn);
    }
</script>

<Table hover striped>
    <thead>
        <tr>
            {#each rdatafields(rrtype) as field}
                <th>
                    {#if header}
                        {@render header(field)}
                    {:else}
                        {field}
                    {/if}
                </th>
            {/each}
        </tr>
    </thead>
    <tbody>
        {#if rrs && rrs.length}
            {#each rrs as rr, i}
                <tr>
                    {#each rdatafields(rrtype) as f}
                        <td>
                            {#if field}
                                {@render field(i, f)}
                            {:else}
                                {rrs[i][f]}
                            {/if}
                        </td>
                    {/each}
                    <td>
                        <Button
                            type="button"
                            color="info"
                            outline
                            size="sm"
                            on:click={() => openEditor(rrs[i])}
                        >
                            <Icon name="search" />
                        </Button>
                        {#if edit}
                            <Button
                                type="button"
                                color="danger"
                                outline
                                size="sm"
                                on:click={() => deleteLine(i)}
                            >
                                <Icon name="trash" />
                            </Button>
                        {/if}
                    </td>
                </tr>
            {/each}
        {:else}
            <tr>
                <td
                    colspan={rdatafields(rrtype).length}
                    class="fst-italic text-center"
                >
                    {$t("common.no-content")}
                </td>
            </tr>
        {/if}
    </tbody>
        {#if edit}
            <tfoot>
                <tr>
                    <td colspan={1}>
                        <Button type="button" color="primary" outline size="sm" on:click={addLine}>
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        {/if}
</Table>

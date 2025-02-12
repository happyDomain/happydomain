<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import {
        Badge,
        Card,
        CardBody,
        CardText,
        CardTitle,
        CardSubtitle,
        Icon,
        ListGroup,
        ListGroupItem,
        Table,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import Record from "$lib/components/domains/Record.svelte";
    import type { ServiceRecord } from "$lib/model/zone";

    export let service = null;
</script>

{#if service}
    <div
        class="flex-fill m-0 d-flex flex-column"
        style:max-width="100%"
    >
        {#each Object.keys(service) as i}
            {@const record = service[i]}
            {#if record && record.Hdr}
                <div
                    class="bg-light p-1 border-1 border-bottom"
                    style="border-color: darkgray"
                >
                    <Record {record} />
                </div>
            {:else if record && record.length}
                {#each record as r}
                    <div
                        class="bg-light p-1 border-1 border-bottom"
                        style="border-color: darkgray"
                    >
                        <Record record={r} />
                    </div>
                {/each}
            {/if}
        {/each}
    </div>
{/if}

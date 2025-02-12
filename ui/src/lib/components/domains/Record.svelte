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
     Button,
     Col,
     Icon,
     Input,
     Row,
 } from '@sveltestrap/sveltestrap';

 import { nsclass, nsrrtype, rdatatostr } from '$lib/dns';
 import type { ServiceRecord } from '$lib/model/zone';

 export let actBtn = false;
 export let expand: boolean = false;
 export let record;
</script>

{#if !record.edit}
    <div
        class="d-flex gap-1"
        on:click={() => {expand = !expand}}
        on:keypress={() => {expand = !expand}}
    >
        <Icon
            name={expand ? "chevron-down" : "chevron-right"}
        />
        <span
            class="font-monospace text-truncate"
            title={rdatatostr(record)}
        >
            {record.Hdr.Name?record.Hdr.Name:'@'} {nsrrtype(record.Hdr.Rrtype)} {rdatatostr(record)}
        </span>
    </div>
    {#if expand}
        <div class="grid mr-2">
            <dl class="g-col-md-4 grid ms-2 mb-0 mt-1" style="--bs-columns: 2; --bs-gap: 0 .5rem;">
                <dt class="text-end">
                    Class
                </dt>
                <dd class="text-muted font-monospace mb-1">
                    {nsclass(record.Hdr.Class)}
                </dd>
                <dt class="text-end">
                    TTL
                </dt>
                <dd class="text-muted font-monospace mb-1">
                    {record.Hdr.Ttl}
                </dd>
                <dt class="text-end">
                    RRType
                </dt>
                <dd class="text-muted font-monospace mb-1">
                    {nsrrtype(record.Hdr.Rrtype)} (<span title={record.Hdr.Rrtype}>0x{record.Hdr.Rrtype.toString(16)}</span>)
                </dd>
            </dl>
            <dl class="g-col-md-8 grid me-2" style="--bs-gap: 0 .5rem;">
                {#each Object.keys(record) as k}
                    {#if k != "Hdr"}
                        {@const v = record[k]}
                        <dt class="g-col-4 text-end">
                            {k}
                        </dt>
                        <dd
                            class="g-col-8 text-muted font-monospace text-truncate mb-1"
                            title={v}
                        >
                            {v}
                        </dd>
                    {/if}
                {/each}
            </dl>
        </div>
    {/if}
{:else}
    <form
        submit="$emit('save-rr')"
    >
        <Input
            autofocus
            class="font-monospace"
            bsSize="sm"
            bind:value={record.str}
        />
    </form>
{/if}
{#if record.edit || actBtn}
    {#if record.edit}
        <Button
            size="sm"
            color="success"
            click="$emit('save-rr')"
        >
            <Icon name="check" aria-hidden="true" />
        </Button>
    {:else if record.rr.Hdr.Rrtype != 6}
        <Button
            size="sm"
            color="danger"
            click="$emit('delete-rr')"
        >
            <Icon name="trash-fill" aria-hidden="true" />
        </Button>
    {/if}
{/if}

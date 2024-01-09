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
 import { createEventDispatcher } from 'svelte';

 import {
     Badge,
     ListGroupItem,
 } from '@sveltestrap/sveltestrap';

 import { nsrrtype } from '$lib/dns';
 import type { ServiceInfos } from '$lib/model/service_specs';
 import { userSession } from '$lib/stores/usersession';

 const dispatch = createEventDispatcher();

 export let active = false;
 export let disabled = false;
 export let reason: string = "";

 export let svc: ServiceInfos;
</script>

<ListGroupItem
    {active}
    class="d-flex"
    {disabled}
    tag="button"
    on:click={() => dispatch("click")}
>
    {#if svc._svcicon}
        <div class="d-inline-block align-self-center text-center" style="width: 75px;">
            <img src={svc._svcicon} alt={svc.name} style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
        </div>
    {/if}
    <div class="flex-fill">
        {svc.name}
        {#if reason}
            <small class="font-italic text-danger">{reason}</small>
        {:else}
            <small class="text-muted">{svc.description}</small>
        {/if}
        {#if svc.categories}
            {#each svc.categories as category}
                <Badge
                    color="secondary"
                    class="float-end ms-1"
                >
                    {category}
                </Badge>
            {/each}
        {/if}
        {#if svc.record_types && $userSession.settings && $userSession.settings.showrrtypes}
            {#each svc.record_types as rtype}
                <Badge
                    color="info"
                    class="float-end ms-1"
                >
                    {nsrrtype(rtype)}
                </Badge>
            {/each}
        {/if}
    </div>
</ListGroupItem>

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

 import HListGroup from '$lib/components/ListGroup.svelte';
 import { groups } from '$lib/stores/domains';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let flush = false;
 export let selectedGroup: string | null = null;

 function selectGroup(event: CustomEvent<string>) {
     if (selectedGroup != null && selectedGroup == event.detail) {
         selectedGroup = null;
     } else {
         selectedGroup = event.detail;
         dispatch("click", selectedGroup);
     }
 }
</script>

<HListGroup
    button
    items={$groups}
    {flush}
    isActive={(item) => (selectedGroup != null && item === selectedGroup)}
    on:click={selectGroup}
    let:item={item}
>
    {#if item === '' || item === 'undefined'}
        {$t('domaingroups.no-group')}
    {:else}
        {item}
    {/if}
</HListGroup>

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

 import KratosNode from '$lib/components/KratosNode.svelte';

 const dispatch = createEventDispatcher();

 export let flow: String;
 export let nodes: Array;
 export let only: String;
 export let submissionInProgress = false;

 let values = { };


 function initializeValues(nodes) {
     const vls = { };
     for (const node of nodes) {
         if (!only || node.group === only || node.group === 'default') {
             vls[node.attributes.name] = node.attributes.value;
         }
     }
     values = vls;
 }
 $: initializeValues(nodes);

 function submission() {
     dispatch('submit', values);
 }
</script>

<form
    class="container my-1"
    method="post"
    onsubmit="alert('test'); return false;"
    on:submit|preventDefault={submission}
>
    {#each nodes as node, i}
        {#if !only || node.group === only || node.group === 'default'}
            <KratosNode
                {flow}
                i={only + i}
                {node}
                {submissionInProgress}
                bind:value={values[node.attributes.name]}
            />
        {/if}
    {/each}
</form>

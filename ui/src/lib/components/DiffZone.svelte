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
     Icon,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import {
     diffZone as APIDiffZone,
 } from '$lib/api/zone';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let domain: DomainInList | Domain;
 export let zoneFrom: string;
 export let zoneTo: string;
 export let selectable = false;
 export let selectedDiff: Array<string> | null = null;

 let zoneDiff: Array<{className: string; msg: string;}>;
 $: computeDiff(domain, zoneTo, zoneFrom);

 function computeDiff(domain: DomainInList | Domain, zoneTo: string, zoneFrom: string) {
     APIDiffZone(domain, zoneTo, zoneFrom).then(
         (v: Array<string>) => {
             let zoneDiffCreated = 0;
             let zoneDiffDeleted = 0;
             let zoneDiffModified = 0;
             if (v) {
                 zoneDiff = v.map(
                     (msg: string) => {
                         let className = '';
                         if (/^± MODIFY/.test(msg)) {
                             className = 'text-warning';
                             zoneDiffModified += 1;
                         } else if (/^\+ CREATE/.test(msg)) {
                             className = 'text-success';
                             zoneDiffCreated += 1;
                         } else if (/^- DELETE/.test(msg)) {
                             className = 'text-danger';
                             zoneDiffDeleted += 1;
                         } else if (/^REFRESH/.test(msg)) {
                             className = 'text-info';
                         }

                         return {
                             className,
                             msg,
                         };
                     }
                 );
             } else {
                 zoneDiff = [];
             }
             selectedDiff = v;
             dispatch('computed-diff', {zoneDiffLength: zoneDiff.length, zoneDiffCreated, zoneDiffDeleted, zoneDiffModified});
         },
         (err: any) => {
             dispatch('error', err);
         }
     )
 }

</script>

{#if !zoneDiff}
    <div class="my-2 text-center">
        <Spinner color="warning" label="Spinning" />
        <p>{$t('wait.exporting')}</p>
    </div>
{:else if zoneDiff.length == 0}
    <slot name="nodiff">
        Aucune différence.
    </slot>
{:else}
    {#each zoneDiff as line, n}
        <div
            class={'font-monospace ' + line.className}
            class:col={selectable}
            class:form-check={selectable}
        >
            {#if selectable}
                <input
                    type="checkbox"
                    class="form-check-input"
                    id="zdiff{n}"
                    bind:group={selectedDiff}
                    value={line.msg}
                />
                <label
                    class="form-check-label"
                    for="zdiff{n}"
                    style="padding-left: 1em; text-indent: -1em;"
                >
                    {line.msg}
                </label>
            {:else}
                {line.msg}
            {/if}
        </div>
    {/each}
{/if}

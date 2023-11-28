<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Icon,
     Spinner,
 } from 'sveltestrap';

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

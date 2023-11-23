<script lang="ts">
 import { t } from '$lib/translations';

 export let zoneDiff: Array<string>;

 let zoneDiffCreated = 0;
 let zoneDiffDeleted = 0;
 let zoneDiffModified = 0;
 let zoneDiffOther = 0;

 $: {
     zoneDiffCreated = 0;
     zoneDiffDeleted = 0;
     zoneDiffModified = 0;
     zoneDiffOther = 0;

     if (zoneDiff && zoneDiff.length) {
         zoneDiff.forEach(
             (msg: string) => {
                 if (/^± MODIFY/.test(msg)) {
                     zoneDiffModified += 1;
                 } else if (/^\+ CREATE/.test(msg)) {
                     zoneDiffCreated += 1;
                 } else if (/^- DELETE/.test(msg)) {
                     zoneDiffDeleted += 1;
                 } else if (/^REFRESH/.test(msg)) {
                     zoneDiffOther += 1;
                 }
             }
         );
     }
 }
</script>

{#if zoneDiff}
    {#if zoneDiffCreated}
        <span class="text-success">
            {$t('domains.apply.additions', {count: zoneDiffCreated})}
        </span>
    {/if}
    {#if zoneDiffCreated && zoneDiffDeleted}
        &ndash;
    {/if}
    {#if zoneDiffDeleted}
        <span class="text-danger">
            {$t('domains.apply.deletions', {count: zoneDiffDeleted})}
        </span>
    {/if}
    {#if (zoneDiffCreated || zoneDiffDeleted) && zoneDiffModified}
        &ndash;
    {/if}
    {#if zoneDiffModified}
        <span class="text-warning">
            {$t('domains.apply.modifications', {count: zoneDiffModified})}
        </span>
    {/if}
    {#if (zoneDiffCreated || zoneDiffDeleted || zoneDiffModified) && (zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0)}
        &ndash;
    {/if}
    {#if zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
        <span class="text-info">
            {$t('domains.apply.others', {count: zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified})}
        </span>
    {/if}
{:else}
    {$t('domains.apply.modifications', {count: 0})}
{/if}

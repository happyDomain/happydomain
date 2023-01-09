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

<script lang="ts">
 import {
     ModalHeader,
 } from 'sveltestrap';

 import { t } from '$lib/translations';

 export let toggle: () => void;
 export let dn: string;
 export let update = false;
</script>

<ModalHeader {toggle}>
    {#if update}
        {$t('service.update')}
    {:else}
        {$t('service.form-new')}
    {/if}
    <span class="font-monospace">{dn}</span>
</ModalHeader>

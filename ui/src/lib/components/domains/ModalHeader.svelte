<script lang="ts">
 // @ts-ignore
 import { escape } from 'html-escaper';
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
        {@html $t('service.form-new', {"domain": `<span class="font-monospace">${escape(dn)}</span>`})}
    {/if}
</ModalHeader>

<script context="module" lang="ts">
 export const controls = { };
</script>

<script lang="ts">
 import {
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Spinner,
 } from 'sveltestrap';

 import {
     viewZone as APIViewZone,
 } from '$lib/api/zone';
 import { t } from '$lib/translations';

 export let isOpen = false;

 let zoneContent: null | string = null;
 function Open(domain: string, selectedHistory: string): void {
     zoneContent = null;
     isOpen = true;
     APIViewZone(domain, selectedHistory).then(
         (v: string) => zoneContent = v,
         (err: any) => {
             isOpen = false;
             throw err;
         }
     );
 }

 controls.Open = Open;
</script>

<Modal
    isOpen={isOpen}
    size="lg"
    scrollable
>
    <ModalHeader toggle={() => isOpen = false}>{$t('domains.view.title')}</ModalHeader>
    <ModalBody>
        {#if zoneContent}
            <pre style="overflow: initial">{zoneContent}</pre>
        {:else}
            <div class="my-2 text-center">
                <Spinner label="Spinning" />
                <p>{$t('wait.formating')}</p>
            </div>
        {/if}
    </ModalBody>
</Modal>

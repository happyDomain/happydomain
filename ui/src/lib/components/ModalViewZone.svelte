<script context="module" lang="ts">
 import type { ModalController } from '$lib/model/modal_controller';

 export const controls: ModalController = { };
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
 import type { Domain, DomainInList } from '$lib/model/domain';
 import { t } from '$lib/translations';

 export let isOpen = false;

 let zoneContent: null | string = null;
 function Open(domain: Domain | DomainInList, selectedHistory: string): void {
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

<script context="module" lang="ts">
 import type { ModalController } from '$lib/model/modal_controller';

 export const controls: ModalController = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
 } from 'sveltestrap';

 import {
     viewZone as APIViewZone,
 } from '$lib/api/zone';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let isOpen = false;

 function Open(): void {
     isOpen = true;
 }

 controls.Open = Open;
</script>

<Modal
    isOpen={isOpen}
    size="lg"
>
    <ModalHeader toggle={() => isOpen = false}>{$t('domains.removal')}</ModalHeader>
    <ModalBody>
        {$t('domains.alert.remove')}
    </ModalBody>
    <ModalFooter>
        <Button
            outline
            color="secondary"
            on:click={() => isOpen = false}
        >
            {$t('domains.view.cancel-title')}
        </Button>
        <Button
            color="danger"
            on:click={() => dispatch('detachDomain')}
        >
            {$t('domains.discard')}
        </Button>
    </ModalFooter>
</Modal>

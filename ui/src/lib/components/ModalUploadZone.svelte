<script context="module" lang="ts">
 export const controls = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Input,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Spinner,
     TabContent,
     TabPane,
 } from 'sveltestrap';

 import {
     importZone as APIImportZone,
 } from '$lib/api/zone';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let domain: string = '';
 export let selectedHistory: string = '';
 export let isOpen = false;

 let uploadInProgress = false;
 let zoneImportContent = "";
 let zoneImportFiles: FileList;
 let uploadModalActiveTab: string|number = 0;

 function importZone(): void {
     uploadInProgress = true;
     let file = new Blob([zoneImportContent], {"type": "text/plain"});
     if (uploadModalActiveTab != "uploadText") {
         file = zoneImportFiles[0];
     }
     APIImportZone(domain, selectedHistory, file).then(
         (v) => {
             isOpen = false;
             dispatch('retrieveZoneDone', v);
         },
         (err: any) => {
             uploadInProgress = false;
             throw err;
         }
     );
 }

 function Open(): void {
     isOpen = true;
     zoneImportContent = "";
     uploadModalActiveTab = 0;
 }

 controls.Open = Open;
</script>

<Modal
    isOpen={isOpen}
    size="lg"
>
    <ModalHeader toggle={() => isOpen = false}>{$t('zones.upload')}</ModalHeader>
    <ModalBody>
        <TabContent on:tab={(e) => (uploadModalActiveTab = e.detail)}>
            <TabPane tabId="uploadText" tab={$t('zones.import-text')} active>
                <Input
                    class="mt-3"
                    type="textarea"
                    style="height: 200px;"
                    placeholder="@         4269 IN SOA   root ns 2042070136 ..."
                    bind:value={zoneImportContent}
                />
            </TabPane>
            <TabPane tabId="uploadFile" tab={$t('zones.import-file')}>
                {#if isOpen}
                    <Input
                        class="mt-3"
                        type="file"
                        bind:files={zoneImportFiles}
                    />
                {/if}
            </TabPane>
        </TabContent>
    </ModalBody>
    <ModalFooter>
        <Button
            outline
            color="secondary"
            on:click={() => isOpen = false}
        >
            {$t('common.cancel')}
        </Button>
        <Button
            color="primary"
            disabled={uploadInProgress}
            on:click={importZone}
        >
            {#if uploadInProgress}
                <Spinner size="sm" />
            {/if}
            {$t('domains.actions.upload')}
        </Button>
    </ModalFooter>
</Modal>

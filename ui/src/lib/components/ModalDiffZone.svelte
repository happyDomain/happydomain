<script context="module" lang="ts">
 import type { ModalController } from '$lib/model/modal_controller';

 export const controls: ModalController = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Icon,
     Input,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Spinner,
 } from 'sveltestrap';

 import {
     applyZone as APIApplyZone,
     diffZone as APIDiffZone,
 } from '$lib/api/zone';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let domain: DomainInList | Domain;
 export let selectedHistory: string = '';
 export let isOpen = false;

 let zoneDiff: Array<{className: string; msg: string;}> | null = null;
 let zoneDiffCreated = 0;
 let zoneDiffDeleted = 0;
 let zoneDiffModified = 0;

 let selectedDiff: Array<string> | null = null;
 let diffCommitMsg = '';
 let selectedDiffCreated = 0;
 let selectedDiffDeleted = 0;
 let selectedDiffModified = 0;
 $: selectedDiffCreated = !selectedDiff?0:selectedDiff.filter((msg: string) => /^\+ CREATE/.test(msg)).length;
 $: selectedDiffDeleted = !selectedDiff?0:selectedDiff.filter((msg: string) => /^- DELETE/.test(msg)).length;
 $: selectedDiffModified = !selectedDiff?0:selectedDiff.filter((msg: string) => /^± MODIFY/.test(msg)).length;

 function Open(): void {
     zoneDiff = null;
     selectedDiff = null;
     isOpen = true;
     propagationInProgress = false;
     diffCommitMsg = '';

     APIDiffZone(domain, '@', selectedHistory).then(
         (v: Array<string>) => {
             zoneDiffCreated = 0;
             zoneDiffDeleted = 0;
             zoneDiffModified = 0;
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
         },
         (err: any) => {
             isOpen = false;
             throw err;
         }
     )
 }

 let propagationInProgress = false;
 async function applyDiff() {
     if (!domain || !selectedHistory || !selectedDiff) return;

     propagationInProgress = true;
     try {
         dispatch('retrieveZoneDone', await APIApplyZone(domain, selectedHistory, selectedDiff, diffCommitMsg));
     } finally {
         isOpen = false;
     }
 }

 controls.Open = Open;
</script>

<Modal
    isOpen={isOpen}
    size="lg"
    scrollable
>
    {#if domain}
        <ModalHeader toggle={() => isOpen = false}>
            {@html $t('domains.view.description', {"domain": `<span class="font-monospace">${escape(domain.domain)}</span>`})}
        </ModalHeader>
    {/if}
    <ModalBody>
        {#if !zoneDiff}
            <div class="my-2 text-center">
                <Spinner color="warning" label="Spinning" />
                <p>{$t('wait.exporting')}</p>
            </div>
        {:else if zoneDiff.length == 0}
            <div class="d-flex gap-3 align-items-center justify-content-center">
                <Icon name="check2-all" class="display-5 text-success" />
                {$t('domains.apply.nochange')}
            </div>
        {:else}
            {#each zoneDiff as line, n}
                <div
                    class={'col font-monospace form-check ' + line.className}
                >
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
                </div>
            {/each}
        {/if}
    </ModalBody>
    <ModalFooter>
        {#if zoneDiff}
            {#if zoneDiff.length > 0}
                <Input
                    id="commitmsg"
                    placeholder={$t('domains.commit-msg')}
                    size="sm"
                    bind:value={diffCommitMsg}
                />
            {/if}
            {#if zoneDiffCreated}
                <span class="text-success">
                    {$t('domains.apply.additions', {count: selectedDiffCreated})}
                </span>
            {/if}
            {#if zoneDiffCreated && zoneDiffDeleted}
                &ndash;
            {/if}
            {#if zoneDiffDeleted}
                <span class="text-danger">
                    {$t('domains.apply.deletions', {count: selectedDiffDeleted})}
                </span>
            {/if}
            {#if (zoneDiffCreated || zoneDiffDeleted) && zoneDiffModified}
                &ndash;
            {/if}
            {#if zoneDiffModified}
                <span class="text-warning">
                    {$t('domains.apply.modifications', {count: selectedDiffModified})}
                </span>
            {/if}
            {#if (zoneDiffCreated || zoneDiffDeleted || zoneDiffModified) && (zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0)}
                &ndash;
            {/if}
            {#if selectedDiff && zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
                <span class="text-info">
                    {$t('domains.apply.others', {count: selectedDiff.length - selectedDiffCreated - selectedDiffDeleted - selectedDiffModified})}
                </span>
            {/if}
        {/if}
        <div class="d-flex gap-1">
            <Button outline color="secondary" on:click={() => isOpen = false}>
                {$t('common.cancel')}
            </Button>
            <Button color="success" disabled={propagationInProgress || !zoneDiff || !selectedDiff || selectedDiff.length === 0} on:click={applyDiff}>
                {#if propagationInProgress}
                    <Spinner label="Spinning" size="sm" />
                {/if}
                {$t('domains.apply.button')}
            </Button>
        </div>
    </ModalFooter>
</Modal>

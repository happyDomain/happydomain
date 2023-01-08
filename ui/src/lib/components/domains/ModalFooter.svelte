<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     ModalFooter,
     Spinner,
 } from 'sveltestrap';

 import HelpButton from '$lib/components/Help.svelte';
 import type { ServiceCombined } from '$lib/model/service';
 import { locale, t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let toggle: () => void;
 export let step: number;
 export let service: ServiceCombined | null = null;
 export let form = "addSvcForm";
 export let update = false;
 export let canDelete = false;
 export let canContinue = false;

 export let addServiceInProgress = false;
 export let deleteServiceInProgress = false;

 let helpHref = "";
 $: {
     if (service && service._svctype) {
         const svcPart = service._svctype.toLowerCase().split('.')
         if (svcPart.length === 2) {
             if (svcPart[0] === 'svcs') {
                 helpHref = 'records/' + svcPart[1].toUpperCase() + "/";
             } else if (svcPart[0] === 'abstract') {
                 helpHref = 'services/' + svcPart[1] + "/";
             } else if (svcPart[0] === 'provider') {
                 helpHref = 'services/providers/' + svcPart[1] + "/";
             } else {
                 helpHref = svcPart[svcPart.length - 1] + "/";
             }
         } else {
             helpHref = svcPart[svcPart.length - 1] + "/";
         }
     } else {
         helpHref = "";
     }
     helpHref = "https://help.happydomain.org/" + $locale + "/" + helpHref;
 }
</script>

<ModalFooter>
    {#if step === 2}
        <HelpButton
            href={helpHref}
            color="info"
        />
    {/if}
    {#if update}
        <Button
            disabled={addServiceInProgress || deleteServiceInProgress || !canDelete}
            color="danger"
            on:click={() => dispatch("delete-service")}
        >
            {#if deleteServiceInProgress}
                <Spinner label="Spinning" size="sm" />
            {/if}
            {$t('service.delete')}
        </Button>
    {/if}
    <Button
        color="secondary"
        on:click={toggle}
    >
        {$t('common.cancel')}
    </Button>
    {#if step === 2 && update}
        <Button
            disabled={addServiceInProgress || deleteServiceInProgress}
            {form}
            type="submit"
            color="success"
        >
            {#if addServiceInProgress}
                <Spinner label="Spinning" size="sm" />
            {/if}
            {$t('service.update')}
        </Button>
    {:else if step === 2}
        <Button
            {form}
            type="submit"
            color="primary"
        >
            {$t('service.add')}
        </Button>
    {:else}
        <Button
            disabled={!canContinue}
            {form}
            type="submit"
            color="primary"
            >
            {$t('common.continue')}
        </Button>
    {/if}
</ModalFooter>

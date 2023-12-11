<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Icon,
     ModalFooter,
     Spinner,
 } from 'sveltestrap';

 import { getServiceRecords } from '$lib/api/zone';
 import HelpButton from '$lib/components/Help.svelte';
 import TableRecords from '$lib/components/domains/TableRecords.svelte';
 import type { ServiceCombined } from '$lib/model/service';
 import { locale, t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let toggle: () => void;
 export let step: number;
 export let service: ServiceCombined | null = null;
 export let form = "addSvcForm";
 export let origin: Domain | DomainInList | undefined;
 export let update = false;
 export let zoneId: number | undefined;
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

 let showRecords = false;
</script>

{#if showRecords}
    <ModalFooter class="p-0 border-top border-dark border-2 d-flex justify-content-center">
        {#await getServiceRecords(origin, zoneId, service)}
            <Spinner class="my-1" />
        {:then serviceRecords}
            <TableRecords {serviceRecords} />
        {/await}
    </ModalFooter>
{/if}
<ModalFooter>
    {#if origin && zoneId}
        <Button
            color="dark"
            outline={!showRecords}
            title={$t('domains.see-records')}
            on:click={() => showRecords = !showRecords}
        >
            <Icon name="code-square" />
        </Button>
    {/if}
    <div class="ms-auto"></div>
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
                <Spinner size="sm" />
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

<script lang="ts">
 import { tick } from 'svelte';
 import { goto } from '$app/navigation';

 // @ts-ignore
 import { escape } from 'html-escaper';
 import {
     Alert,
     Button,
     ButtonGroup,
     Col,
     Container,
     Icon,
     Input,
     Row,
     Spinner,
 } from 'sveltestrap';

 import {
     getDomain as APIGetDomain,
     deleteDomain as APIDeleteDomain,
 } from '$lib/api/domains';
 import {
     retrieveZone as APIRetrieveZone,
 } from '$lib/api/zone';
 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import ModalDiffZone, { controls as ctrlDiffZone } from '$lib/components/ModalDiffZone.svelte';
 import ModalDomainDelete, { controls as ctrlDomainDelete } from '$lib/components/ModalDomainDelete.svelte';
 import ModalUploadZone, { controls as ctrlUploadZone } from '$lib/components/ModalUploadZone.svelte';
 import ModalViewZone, { controls as ctrlViewZone } from '$lib/components/ModalViewZone.svelte';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ZoneMeta } from '$lib/model/zone';
 import { domains, domains_idx, refreshDomains } from '$lib/stores/domains';
 import { providers, providers_idx, refreshProviders } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 export let data: {domain: string; history: string;};

 let selectedDomain = data.domain;
 $: if (selectedDomain != data.domain) {
     main_error = null;
     goto('/domains/' + encodeURIComponent(selectedDomain));
 }

 if (!$domains) refreshDomains();
 if (!$providers) refreshProviders();

 let domainsByGroup: Record<string, Array<DomainInList>> = {};
 $: {
     if ($domains) {
         const tmp: Record<string, Array<DomainInList>> = { };

         for (const domain of $domains) {
             if (tmp[domain.group] === undefined) {
                 tmp[domain.group] = [];
             }

             tmp[domain.group].push(domain);
         }

         domainsByGroup = tmp;
     }
 }

 let main_error: string | null = null;

 let selectedHistory: string | undefined = data.history;
 $: if (!data.history && $domains_idx[selectedDomain] && $domains_idx[selectedDomain].zone_history && $domains_idx[selectedDomain].zone_history.length > 0) {
     selectedHistory = $domains_idx[selectedDomain].zone_history[0] as string;
 }
 $: if (selectedHistory && data.history != selectedHistory) {
     main_error = null;
     goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(selectedHistory));
 }

 let retrievalInProgress = false;
 function retrieveZone(): void {
     if (domain) {
         retrievalInProgress = true;
         APIRetrieveZone(domain).then(
             retrieveZoneDone,
             (err: any) => {
                 retrievalInProgress = false;
                 throw err;
             }
         );
     }
 }

 function retrieveZoneDone(zm: ZoneMeta): void {
     retrievalInProgress = false;
     refreshDomains();
     selectedHistory = zm.id;
     main_error = null;
 }

 async function getDomain(id: string): Promise<Domain> {
     return await APIGetDomain(id);
 }

 let domain: null | Domain = null;
 $: if ($domains_idx[selectedDomain]) {
     if (!$domains_idx[selectedDomain].zone_history || $domains_idx[selectedDomain].zone_history.length == 0) {
         retrievalInProgress = true;
         APIRetrieveZone($domains_idx[selectedDomain]).then(
             retrieveZoneDone,
             (err: any) => {
                 retrievalInProgress = false;

                 tick().then(() => {
                     main_error = err.toString();
                 });
             }
         )
     } else {
         domain = null;
         getDomain($domains_idx[selectedDomain].id).then(
             (dn) => {
                 domain = dn;
             }
         );
     }
 }

 function viewZone(): void {
     if (!domain || !selectedHistory) {
         return;
     }

     ctrlViewZone.Open(domain, selectedHistory);
 }

 function showDiff(): void {
     if (!domain || !selectedHistory) {
         return;
     }

     ctrlDiffZone.Open(domain, selectedHistory);
 }

 let deleteInProgress = false;
 function detachDomain(): void {
     deleteInProgress = true;
     APIDeleteDomain($domains_idx[selectedDomain].id).then(
         () => {
             refreshDomains().then(() => {
                 deleteInProgress = false;
                 goto('/domains');
             }, () => {
                 deleteInProgress = false;
                 goto('/domains');
             });
         },
         (err: any) => {
             deleteInProgress = false;
             throw err;
         }
     );
 }
</script>

<Container
    fluid
    class="d-flex flex-column flex-fill"
>
    <Row
        class="flex-fill"
    >
        <Col
            sm={4}
            md={3}
            class="bg-light py-2 sticky-top d-flex flex-column justify-content-between"
            style="overflow-y: auto; max-height: 100vh; z-index: 0"
        >
            {#if $domains_idx[selectedDomain]}
                <div class="d-flex">
                    <Button href="/domains/" class="fw-bolder" color="link">
                        <Icon name="chevron-up" />
                    </Button>
                    <Input
                        type="select"
                        bind:value={selectedDomain}
                    >
                        {#each Object.keys(domainsByGroup) as gname}
                            {@const group = domainsByGroup[gname]}
                            <optgroup label={gname=="undefined"?$t("domaingroups.no-group"):gname}>
                                {#each group as domain}
                                    <option value={domain.domain}>{domain.domain}</option>
                                {/each}
                            </optgroup>
                        {/each}
                    </Input>
                </div>

                {#if domain && domain.zone_history && $domains_idx[selectedDomain] && domain.id === $domains_idx[selectedDomain].id}
                    <form class="mt-3">
                        <div class="d-flex justify-content-between">
                            <label class="fw-bolder" for="zhistory">{$t('domains.history')}:</label>
                            <div class="d-flex gap-1">
                                <Button
                                    outline
                                    color="secondary"
                                    size="sm"
                                    title={$t('domains.actions.upload')}
                                    on:click={ctrlUploadZone.Open}
                                >
                                    <Icon name="cloud-upload" />
                                </Button>
                                <Button
                                    outline
                                    color="secondary"
                                    size="sm"
                                    title={$t('domains.actions.reimport')}
                                    disabled={retrievalInProgress}
                                    on:click={retrieveZone}
                                >
                                    {#if retrievalInProgress}
                                        <Spinner size="sm" />
                                    {:else}
                                        <Icon name="cloud-download" />
                                    {/if}
                                </Button>
                            </div>
                        </div>
                        {#key domain.zone_history}
                            <select
                                class="form-select"
                                id="zhistory"
                                bind:value={selectedHistory}
                            >
                                {#each domain.zone_history as history}
                                    <option value={history.id}>{history.last_modified}</option>
                                {/each}
                            </select>
                        {/key}
                    </form>

                    <ButtonGroup class="mt-3 w-100">
                        <Button
                            size="sm"
                            outline
                            color="info"
                            title={$t('domains.actions.view')}
                            on:click={viewZone}
                        >
                            <Icon name="list-ul" aria-hidden="true" /><br>
                            {$t('domains.actions.view')}
                        </Button>
                        {#if $domains_idx[selectedDomain].zone_history && selectedHistory === $domains_idx[selectedDomain].zone_history[0]}
                            <Button
                                size="sm"
                                color="success"
                                title={$t('domains.actions.propagate')}
                                on:click={showDiff}
                            >
                                <Icon name="cloud-upload" aria-hidden="true" /><br>
                                {$t('domains.actions.propagate')}
                            </Button>
                        {:else}
                            <Button
                                size="sm"
                                color="warning"
                                title={$t('domains.actions.rollback')}
                                on:click={showDiff}
                            >
                                <Icon name="cloud-upload" aria-hidden="true" /><br>
                                {$t('domains.actions.rollback')}
                            </Button>
                        {/if}
                    </ButtonGroup>
                {/if}

                <div class="flex-fill my-3" />

                <Button
                    class="w-100"
                    type="button"
                    outline
                    color="danger"
                    disabled={deleteInProgress}
                    on:click={() => ctrlDomainDelete.Open()}
                >
                    {#if deleteInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="trash-fill" />
                    {/if}
                    {$t('domains.stop')}
                </Button>

                {#if $providers_idx && $providers_idx[$domains_idx[selectedDomain].id_provider]}
                    <form class="mt-2">
                        <!-- svelte-ignore a11y-label-has-associated-control -->
                        <label class="font-weight-bolder">
                            {$t('domains.view.provider')}:
                        </label>
                        <div class="pr-2 pl-2">
                            <Button
                                href={"/providers/" + encodeURIComponent($domains_idx[selectedDomain].id_provider)}
                                class="p-3 w-100 text-left"
                                type="button"
                                color="info"
                                outline
                            >
                                <div
                                    class="d-inline-block text-center"
                                    style="width: 50px;"
                                >
                                    <ImgProvider id_provider={$domains_idx[selectedDomain].id_provider} />
                                </div>
                                {$providers_idx[$domains_idx[selectedDomain].id_provider]._comment}
                            </Button>
                        </div>
                    </form>
                {/if}
            {:else}
                <div class="mt-4 text-center">
                    <Spinner color="primary" />
                </div>
            {/if}
        </Col>
        <Col
            sm={8}
            md={9}
            class="d-flex pe-0"
        >
            {#if main_error}
                <div class="d-flex flex-column mt-3">
                    <Alert
                        color="danger"
                        fade={false}
                    >
                        <strong>{$t('errors.domain-import')}</strong>
                        {main_error}
                    </Alert>
                </div>
            {:else if data.history == selectedHistory}
                <slot />
            {:else}
                <div class="mt-5 text-center flex-fill">
                    <Spinner label="Spinning" />
                    <p>{$t('wait.loading')}</p>
                </div>
            {/if}
        </Col>
    </Row>
</Container>

<ModalUploadZone
    {domain}
    {selectedHistory}
    on:retrieveZoneDone={retrieveZoneDone}
/>

<ModalDomainDelete
    on:detachDomain={detachDomain}
/>

<ModalViewZone />

<ModalDiffZone
    {domain}
    {selectedHistory}
    on:retrieveZoneDone={retrieveZoneDone}
/>

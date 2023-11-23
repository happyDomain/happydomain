<script lang="ts">
 import { tick } from 'svelte';
 import { goto } from '$app/navigation';

 // @ts-ignore
 import { escape } from 'html-escaper';
 import {
     Alert,
     Button,
     ButtonDropdown,
     ButtonGroup,
     Col,
     Container,
     DropdownItem,
     DropdownMenu,
     DropdownToggle,
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
 import { fqdn } from '$lib/dns';
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

 let selectedHistory: string | undefined;
 $: selectedHistory = data.history;
 $: if (!data.history && $domains_idx[selectedDomain] && $domains_idx[selectedDomain].zone_history && $domains_idx[selectedDomain].zone_history.length > 0) {
     selectedHistory = $domains_idx[selectedDomain].zone_history[0] as string;
 }
 $: if (selectedHistory && data.history != selectedHistory) {
     main_error = null;
     //goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(selectedHistory));
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

                {#if data && data.streamed && data.streamed.sortedDomains}
                    <div class="d-flex gap-2 pb-2 sticky-top bg-light" style="padding-top: 10px">
                        <Button
                            type="button"
                            color="secondary"
                            outline
                            size="sm"
                            class="flex-fill"
                        >
                            <Icon name="server" />
                            {$t('domains.add-a-subdomain')}
                        </Button>
                        <ButtonDropdown>
                            <DropdownToggle
                                color="secondary"
                                outline
                                size="sm"
                            >
                                <Icon name="wrench-adjustable-circle" aria-hidden="true" />
                            </DropdownToggle>
                            <DropdownMenu>
                                <DropdownItem header class="font-monospace">
                                    {data.selectedDomain.domain}
                                </DropdownItem>
                                <DropdownItem
                                    href={`/domains/${data.selectedDomain.domain}/history`}
                                >
                                    {$t('domains.actions.history')}
                                </DropdownItem>
                                <DropdownItem
                                    href={`/domains/${data.selectedDomain.domain}/logs`}
                                >
                                    {$t('domains.actions.audit')}
                                </DropdownItem>
                                <DropdownItem divider />
                                <DropdownItem
                                    on:click={viewZone}
                                >
                                    {$t('domains.actions.view')}
                                </DropdownItem>
                                <DropdownItem
                                    on:click={retrieveZone}
                                >
                                    {$t('domains.actions.reimport')}
                                </DropdownItem>
                                <DropdownItem
                                    on:click={ctrlUploadZone.Open}
                                >
                                    {$t('domains.actions.upload')}
                                </DropdownItem>
                                <DropdownItem divider />
                                <DropdownItem disabled title="Coming soon...">
                                    {$t('domains.actions.share')}
                                </DropdownItem>
                                <DropdownItem
                                    on:click={() => ctrlDomainDelete.Open()}
                                >
                                    {$t('domains.stop')}
                                </DropdownItem>
                                <DropdownItem divider />
                                <DropdownItem
                                    href={"/providers/" + encodeURIComponent($domains_idx[selectedDomain].id_provider)}
                                >
                                    {$t('provider.update')}
                                </DropdownItem>
                            </DropdownMenu>
                        </ButtonDropdown>
                    </div>
                    {#await data.streamed.sortedDomains then sortedDomains}
                        <div style="min-height:0; overflow-y: auto;">
                        {#each sortedDomains as dn}
                            <a
                                href={'#' + (dn?dn:'@')}
                                title={fqdn(dn, data.selectedDomain.domain)}
                                class="d-block text-truncate font-monospace text-muted text-decoration-none"
                                style={'max-width: none; padding-left: ' + (dn === '' ? 0 : (dn.split('.').length * 10)) + 'px'}
                            >
                                {fqdn(dn, data.selectedDomain.domain)}
                            </a>
                        {/each}
                        </div>
                    {/await}
                {/if}

                <div class="flex-fill" />

                {#if domain && domain.zone_history && $domains_idx[selectedDomain] && domain.id === $domains_idx[selectedDomain].id}
                    <ButtonGroup class="mt-2 w-100">
                        {#if $domains_idx[selectedDomain].zone_history && selectedHistory === $domains_idx[selectedDomain].zone_history[0]}
                            <Button
                                size="lg"
                                color="success"
                                title={$t('domains.actions.propagate')}
                                on:click={showDiff}
                            >
                                <Icon name="cloud-upload" aria-hidden="true" />
                                {$t('domains.actions.propagate')}
                            </Button>
                        {:else}
                            <Button
                                size="lg"
                                color="warning"
                                title={$t('domains.actions.rollback')}
                                on:click={showDiff}
                            >
                                <Icon name="cloud-upload" aria-hidden="true" />
                                {$t('domains.actions.rollback')}
                            </Button>
                        {/if}
                    </ButtonGroup>
                {/if}
                <p class="mt-2 mb-1 text-center">
                    X changes
                </p>
            {:else}
                <div class="mt-4 text-center">
                    <Spinner color="primary" />
                </div>
            {/if}
        </Col>
        <Col
            sm={8}
            md={9}
            class="d-flex"
        >
            {#if main_error}
                <div class="d-flex flex-column mt-4">
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

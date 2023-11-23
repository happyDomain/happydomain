<script lang="ts">
 import { tick } from 'svelte';
 import { goto, invalidateAll } from '$app/navigation';

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
 import ModalDiffZone, { controls as ctrlDiffZone } from '$lib/components/ModalDiffZone.svelte';
 import ModalDomainDelete, { controls as ctrlDomainDelete } from '$lib/components/ModalDomainDelete.svelte';
 import ModalUploadZone, { controls as ctrlUploadZone } from '$lib/components/ModalUploadZone.svelte';
 import ModalViewZone, { controls as ctrlViewZone } from '$lib/components/ModalViewZone.svelte';
 import NewSubdomainPath, { controls as ctrlNewSubdomain } from '$lib/components/NewSubdomainPath.svelte';
 import NewServicePath from '$lib/components/NewServicePath.svelte';
 import NewSubdomainModal from '$lib/components/domains/NewSubdomainModal.svelte';
 import ServiceModal from '$lib/components/domains/ServiceModal.svelte';
 import { fqdn } from '$lib/dns';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ZoneMeta } from '$lib/model/zone';
 import { domains, domains_by_groups, domains_idx, refreshDomains } from '$lib/stores/domains';
 import { retrieveZone as StoreRetrieveZone, sortedDomains, thisZone } from '$lib/stores/thiszone';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList; history: string;};

 let selectedDomain = data.domain.domain;
 $: if (selectedDomain != data.domain.domain) {
     goto('/domains/' + encodeURIComponent(selectedDomain));
 }

 let selectedHistory: string | undefined;
 $: selectedHistory = data.history;
 $: if (!data.history && $domains_idx[selectedDomain] && $domains_idx[selectedDomain].zone_history && $domains_idx[selectedDomain].zone_history.length > 0) {
     selectedHistory = $domains_idx[selectedDomain].zone_history[0] as string;
 }
 $: if (selectedHistory && data.history != selectedHistory) {
     goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(selectedHistory));
 }

 let retrievalInProgress = false;
 async function retrieveZone(): void {
     retrievalInProgress = true;
     retrieveZoneDone(await StoreRetrieveZone(data.domain));
 }

 function retrieveZoneDone(zm: ZoneMeta): void {
     retrievalInProgress = false;
     if (data.history) {
         selectedHistory = zm.id;
     } else {
         invalidateAll();
     }
 }

 async function getDomain(id: string): Promise<Domain> {
     return await APIGetDomain(id);
 }

 function viewZone(): void {
     if (!selectedHistory) {
         return;
     }

     ctrlViewZone.Open(data.domain, selectedHistory);
 }

 function showDiff(): void {
     if (!selectedHistory) {
         return;
     }

     ctrlDiffZone.Open(data.domain, selectedHistory);
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
                        {#each Object.keys($domains_by_groups) as gname}
                            {@const group = $domains_by_groups[gname]}
                            <optgroup label={gname=="undefined"?$t("domaingroups.no-group"):gname}>
                                {#each group as domain}
                                    <option value={domain.domain}>{domain.domain}</option>
                                {/each}
                            </optgroup>
                        {/each}
                    </Input>
                </div>

                {#if data && data.streamed && $sortedDomains}
                    <div class="d-flex gap-2 pb-2 sticky-top bg-light" style="padding-top: 10px">
                        <Button
                            type="button"
                            color="secondary"
                            outline
                            size="sm"
                            class="flex-fill"
                            on:click={() => ctrlNewSubdomain.Open()}
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
                                {#if retrievalInProgress}
                                    <Spinner size="sm" />
                                {:else}
                                    <Icon name="wrench-adjustable-circle" aria-hidden="true" />
                                {/if}
                            </DropdownToggle>
                            <DropdownMenu>
                                <DropdownItem header class="font-monospace">
                                    {data.domain.domain}
                                </DropdownItem>
                                <DropdownItem
                                    href={`/domains/${data.domain.domain}/history`}
                                >
                                    {$t('domains.actions.history')}
                                </DropdownItem>
                                <DropdownItem
                                    href={`/domains/${data.domain.domain}/logs`}
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
                    {#await data.streamed.zone then z}
                        <div style="min-height:0; overflow-y: auto;">
                        {#each $sortedDomains as dn}
                            <a
                                href={'#' + (dn?dn:'@')}
                                title={fqdn(dn, data.domain.domain)}
                                class="d-block text-truncate font-monospace text-muted text-decoration-none"
                                style={'max-width: none; padding-left: ' + (dn === '' ? 0 : (dn.split('.').length * 10)) + 'px'}
                            >
                                {fqdn(dn, data.domain.domain)}
                            </a>
                        {/each}
                        </div>
                    {/await}
                {/if}

                <div class="flex-fill" />

                {#if data.domain.zone_history && $domains_idx[selectedDomain] && data.domain.id === $domains_idx[selectedDomain].id}
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
            {#if data.history == selectedHistory}
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

<NewSubdomainPath
    origin={data.domain}
/>
{#await data.streamed.zone then zone}
    <NewServicePath
        origin={data.domain}
        {zone}
    />
    <ServiceModal
        origin={data.domain}
        {zone}
        on:update-zone-services={(event) => thisZone.set(event.detail)}
    />
{/await}

<ModalUploadZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={retrieveZoneDone}
/>

<ModalDomainDelete
    on:detachDomain={detachDomain}
/>

<ModalViewZone />

<ModalDiffZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={retrieveZoneDone}
/>

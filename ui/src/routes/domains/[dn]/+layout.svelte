<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
 import { tick } from 'svelte';
 import { goto, invalidateAll } from '$app/navigation';
 import { page } from '$app/stores';

 // @ts-ignore
 import { escape } from 'html-escaper';
 import {
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
 } from '@sveltestrap/sveltestrap';

 import {
     getDomain as APIGetDomain,
     deleteDomain as APIDeleteDomain,
 } from '$lib/api/domains';
 import {
     diffZone as APIDiffZone,
 } from '$lib/api/zone';
 import DiffSummary from '$lib/components/DiffSummary.svelte';
 import ModalDiffZone, { controls as ctrlDiffZone } from '$lib/components/ModalDiffZone.svelte';
 import ModalDomainDelete, { controls as ctrlDomainDelete } from '$lib/components/ModalDomainDelete.svelte';
 import ModalUploadZone, { controls as ctrlUploadZone } from '$lib/components/ModalUploadZone.svelte';
 import ModalViewZone, { controls as ctrlViewZone } from '$lib/components/ModalViewZone.svelte';
 import NewSubdomainPath, { controls as ctrlNewSubdomain } from '$lib/components/NewSubdomainPath.svelte';
 import NewSubdomainModal from '$lib/components/domains/NewSubdomainModal.svelte';
 import SubdomainListTiny from '$lib/components/domains/SubdomainListTiny.svelte';
 import { fqdn, isReverseZone } from '$lib/dns';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ZoneMeta } from '$lib/model/zone';
 import { domains, domains_by_groups, domains_idx, refreshDomains } from '$lib/stores/domains';
 import { retrieveZone as StoreRetrieveZone, sortedDomains, sortedDomainsWithIntermediate, thisZone } from '$lib/stores/thiszone';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList;};

 let selectedDomain = data.domain.domain;
 $: if (selectedDomain != data.domain.domain) {
     goto('/domains/' + encodeURIComponent(selectedDomain) + ($page.data.isAuditPage ? '/logs' : ($page.data.isHistoryPage ? '/history' : '')));
 }

 let selectedHistory: string | undefined;
 $: selectedHistory = $page.data.history;

 let retrievalInProgress = false;
 async function retrieveZone(): void {
     retrievalInProgress = true;
     retrieveZoneDone(await StoreRetrieveZone(data.domain));
 }

 function retrieveZoneDone(zm: ZoneMeta): void {
     retrievalInProgress = false;
     if ($page.data.definedhistory) {
         refreshDomains().then(() => {
             goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(zm.id));
         });
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
                            <optgroup label={gname=="undefined"||!gname?$t("domaingroups.no-group"):gname}>
                                {#each group as domain}
                                    <option value={domain.domain}>{domain.domain}</option>
                                {/each}
                            </optgroup>
                        {/each}
                    </Input>
                </div>

                {#if $page.data.isHistoryPage || $page.data.isAuditPage}
                    <Button
                        class="mt-2"
                        outline
                        color="primary"
                        href={"/domains/" + encodeURIComponent(data.domain.domain)}
                    >
                        <Icon name="chevron-left" />
                        Retour à la zone
                    </Button>
                {:else if $page.data.streamed && $sortedDomainsWithIntermediate}
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
                    {#await $page.data.streamed.zone then z}
                        <div style="min-height:0; overflow-y: auto;">
                            {#if isReverseZone(data.domain.domain)}
                                <SubdomainListTiny
                                    domains={$sortedDomains}
                                    origin={data.domain}
                                />
                            {:else}
                                <SubdomainListTiny
                                    domains={$sortedDomainsWithIntermediate}
                                    origin={data.domain}
                                />
                            {/if}
                        </div>
                    {/await}
                {/if}

                <div class="flex-fill" />

                {#if $page.data.isZonePage && data.domain.zone_history && $domains_idx[selectedDomain] && data.domain.id === $domains_idx[selectedDomain].id}
                    {#if !($page.data.streamed && $sortedDomainsWithIntermediate)}
                        <Button
                            color="danger"
                            class="mt-3"
                            outline
                            on:click={() => ctrlDomainDelete.Open()}
                        >
                            <Icon name="trash" />
                            {$t('domains.stop')}
                        </Button>
                    {:else if $domains_idx[selectedDomain].zone_history && selectedHistory === $domains_idx[selectedDomain].zone_history[0]}
                        <Button
                            size="lg"
                            color="success"
                            title={$t('domains.actions.propagate')}
                            on:click={showDiff}
                        >
                            <Icon name="cloud-upload" aria-hidden="true" />
                            {$t('domains.actions.propagate')}
                        </Button>
                        <p class="mt-2 mb-1 text-center">
                            {#key $thisZone}
                                {#await APIDiffZone(data.domain, '@', selectedHistory)}
                                    {$t('wait.wait')}
                                {:then zoneDiff}
                                    <DiffSummary
                                        {zoneDiff}
                                    />
                                {/await}
                            {/key}
                        </p>
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
                        <p class="mt-2 mb-1 text-center">
                            {#await getDomain(data.domain.id)}
                                Chargement des informations de l'historique
                            {:then domain}
                                {#each domain.zone_history.filter((e) => e.id === selectedHistory) as history}
                                    {#if history.published}
                                        Publiée le
                                        {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.published))}
                                    {:else if history.commit_date}
                                        Enregistrée le
                                        {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.commit_date))}
                                    {:else}
                                        Dernière modification le
                                        {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.last_modified))}
                                    {/if}
                                    {#if history.commit_message}
                                        <br>
                                        {history.commit_message}
                                    {/if}
                                {/each}
                            {/await}
                        </p>
                    {/if}
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
            class="d-flex"
        >
            <slot />
        </Col>
    </Row>
</Container>

<NewSubdomainPath
    origin={data.domain}
/>

<ModalUploadZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)}
/>

<ModalDomainDelete
    on:detachDomain={detachDomain}
/>

<ModalViewZone />

<ModalDiffZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)}
/>

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
 // @ts-ignore
 import { escape } from 'html-escaper';
 import {
     Badge,
     Button,
     Card,
     CardHeader,
     Icon,
     ListGroupItem,
     Spinner,
 } from 'sveltestrap';

 import { addDomain } from '$lib/api/domains';
 import { listImportableDomains } from '$lib/api/provider';
 import ZoneList from '$lib/components/ZoneList.svelte';
 import { fqdnCompare } from '$lib/dns';
 import type { DomainInList } from '$lib/model/domain';
 import type { Provider } from '$lib/model/provider';
 import { providersSpecs } from '$lib/stores/providers';
 import { domains_idx, refreshDomains } from '$lib/stores/domains';
 import { toasts } from '$lib/stores/toasts';
 import { t } from '$lib/translations';

 export let provider: Provider;

 let importableDomainsList: Array<string>|null = null;
 let discoveryError: string|null = null;
 export let noDomainsList = false;

 $: {
     importableDomainsList = null;
     discoveryError = null;
     noDomainsList = false;
     listImportableDomains(provider).then(
         (l) => {
             if (l === null) {
                 importableDomainsList = [];
             } else {
                 l.sort(fqdnCompare);
                 importableDomainsList = l;
             }
         },
         (err) => {
             importableDomainsList = [];
             if (err.name == "ProviderNoDomainListingSupport") {
                 noDomainsList = true;
             } else {
                 discoveryError = err.message;
                 throw err;
             }
         }
     );
 }

 function haveDomain($domains_idx: Record<string, DomainInList>, name: string) {
     let domain: DomainInList | undefined = undefined;
     if (name[name.length-1] == ".") {
         domain = $domains_idx[name];
     } else {
         domain = $domains_idx[name+"."];
     }
     return domain !== undefined && domain.id_provider == provider._id;
 }

 async function importDomain(domain: {domain: string; wait: boolean}) {
     domain.wait = true;
     addDomain(domain.domain, provider)
     .then(
         (mydomain) => {
             domain.wait = false;
             toasts.addToast({
                 title: $t('domains.attached-new'),
                 message: $t('domains.added-success', { domain: mydomain.domain }),
                 href: '/domains/' + mydomain.domain,
                 color: 'success',
                 timeout: 5000,
             });

             if (!allImportInProgress) refreshDomains();
         },
         (error) => {
             domain.wait = false;
             throw error;
         }
     );
 }

 let allImportInProgress = false;
 async function importAllDomains() {
     if (importableDomainsList) {
         allImportInProgress = true;
         for (const d of importableDomainsList) {
             if (!haveDomain($domains_idx, d)) {
                 await importDomain({domain: d, wait: false});
             }
         }
         allImportInProgress = false;
         refreshDomains();
     }
 }
</script>

<Card {...$$restProps}>
    {#if !noDomainsList && !discoveryError}
        <CardHeader>
            <div class="d-flex justify-content-between">
                <div>
                    {@html $t("provider.provider", {"provider": '<em>' + escape(provider._comment?provider._comment:$providersSpecs?$providersSpecs[provider._srctype].name:"") + '</em>'})}
                </div>
                {#if importableDomainsList != null}
                    <Button
                        type="button"
                        color="secondary"
                        disabled={allImportInProgress}
                        size="sm"
                        on:click={importAllDomains}
                    >
                        {#if allImportInProgress}
                            <Spinner size="sm" />
                        {/if}
                        {$t('provider.import-domains')}
                    </Button>
                {/if}
            </div>
        </CardHeader>
    {/if}
    {#if importableDomainsList == null}
        <div class="d-flex justify-content-center align-items-center gap-2 my-3">
            <Spinner color="primary" /> {$t('wait.asking-domains')}
        </div>
    {:else}
        <ZoneList
            flush
            domains={importableDomainsList.map((dn) => ({domain: dn, id_provider: provider._id, wait: false}))}
        >
            <div slot="badges" let:item={domain}>
                {#if haveDomain($domains_idx, domain.domain)}
                    <Badge class="ml-1" color="success">
                        <Icon name="check" />
                        {$t('service.already')}
                    </Badge>
                {:else}
                    <Button
                        type="button"
                        class="ml-1"
                        color="primary"
                        size="sm"
                        disabled={domain.wait || allImportInProgress}
                        on:click={() => importDomain(domain)}
                    >
                        {#if domain.wait}
                            <Spinner size="sm" />
                        {/if}
                        {$t('domains.add-now')}
                    </Button>
                {/if}
            </div>
            <svelte:fragment slot="no-domain">
                {#if discoveryError}
                    <ListGroupItem class="mx-2 my-3">
                        <p class="text-danger">
                            <Icon name="exclamation-octagon-fill" class="float-start display-5 me-2" />
                            {discoveryError}
                        </p>
                        <div class="text-center">
                            <Button
                                href={"/providers/" + encodeURIComponent(provider._id)}
                                outline
                            >
                                {$t('provider.check-config')}
                            </Button>
                        </div>
                    </ListGroupItem>
                {:else if noDomainsList}
                    <ListGroupItem class="text-center my-3">
                        {$t('errors.domain-list')}
                    </ListGroupItem>
                {:else if !importableDomainsList || importableDomainsList.length === 0}
                    <ListGroupItem class="text-center my-3">
                        {$t('errors.domain-have')}
                    </ListGroupItem>
                {:else if importableDomainsList.length === 0}
                    <ListGroupItem class="text-center my-3">
                        {#if $providersSpecs}
                            {$t("errors.domain-all-imported", {"provider": $providersSpecs[provider._srctype].name})}
                        {/if}
                    </ListGroupItem>
                {/if}
            </svelte:fragment>
        </ZoneList>
    {/if}
</Card>

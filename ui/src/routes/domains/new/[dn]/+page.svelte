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
 import { goto } from '$app/navigation';

 import {
     Button,
     Col,
     Container,
     Icon,
     Row,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { addDomain } from '$lib/api/domains';
 import ProviderList from '$lib/components/providers/List.svelte';
 import ProviderNewModal from '$lib/components/providers/NewModal.svelte';
 import type { Provider } from '$lib/model/provider';
 import { domains, refreshDomains } from '$lib/stores/domains';
 import { providers, providersSpecs, refreshProviders, refreshProvidersSpecs } from '$lib/stores/providers';
 import { toasts } from '$lib/stores/toasts';
 import { t } from '$lib/translations';

 if (!$domains) refreshDomains();
 if (!$providers) refreshProviders();
 if (!$providersSpecs) refreshProvidersSpecs();

 export let data: {dn: string};

 let addingNewDomain = false;

 function addDomainToProvider(event: CustomEvent<Provider>) {
     addingNewDomain = true;

     addDomain(data.dn, event.detail)
     .then(
         (domain) => {
             addingNewDomain = false;
             toasts.addToast({
                 title: $t('domains.attached-new'),
                 message: $t('domains.added-success', { domain: domain.domain }),
                 href: '/domains/' + domain.domain,
                 color: 'success',
                 timeout: 5000,
             });

             refreshDomains();
             goto("/domains/")
         },
         (error) => {
             addingNewDomain = false;
             throw error;
         }
     );
 }

 let newModalOpened = false;
 function newProvider() {
     newModalOpened = true;
 }

 function doneAdd() {

 }
</script>

<Container class="d-flex flex-column flex-fill" fluid>
    <h1 class="text-center my-2">
        <Button
            type="button"
            class="fw-bolder"
            color="link"
            on:click={() => history.go(-1)}
        >
            <Icon name="chevron-left" />
        </Button>
        {$t("provider.select-provider")}
    </h1>
    <hr class="mt-0 mb-0">

    {#if addingNewDomain || !$providers}
        <div class="flex-fill d-flex justify-content-center align-items-center">
            <Spinner color="primary" label="Spinning" class="me-3" /> {$t('wait.validating')}
        </div>
    {:else}
        <Row>
            <Col md={{size: 8, offset: 2}}>
                <ProviderList
                    class="mt-3"
                    items={$providers}
                    emit-new-if-empty
                    on:new-provider={newProvider}
                    on:click={addDomainToProvider}
                />

                <p class="mt-3 d-flex justify-content-center align-items-center gap-1">
                    {$t('provider.find')} <button type="button" class="btn btn-link p-0" on:click|preventDefault={newProvider}>{$t('domains.add-now')}</button>
                </p>
            </Col>
        </Row>
    {/if}
</Container>
<ProviderNewModal
    bind:isOpen={newModalOpened}
    on:update-my-providers={doneAdd}
/>

<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Badge,
     Button,
     Card,
     Col,
     Container,
     Icon,
     Row,
     Spinner,
 } from 'sveltestrap';

 import CardImportableDomains from '$lib/components/providers/CardImportableDomains.svelte';
 import DomainGroupList from '$lib/components/DomainGroupList.svelte';
 import DomainGroupModal from '$lib/components/DomainGroupModal.svelte';
 import Logo from '$lib/components/Logo.svelte';
 import NewDomainInput from '$lib/components/NewDomainInput.svelte';
 import ZoneList from '$lib/components/ZoneList.svelte';
 import ProviderList from '$lib/components/providers/List.svelte';
 import type { DomainInList } from '$lib/model/domain';
 import type { Provider } from '$lib/model/provider';
 import { domains, refreshDomains } from '$lib/stores/domains';
 import { providers, providersSpecs, refreshProviders, refreshProvidersSpecs } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 if (!$domains) refreshDomains();
 if (!$providers) refreshProviders();
 if (!$providersSpecs) refreshProvidersSpecs();

 let noDomainsList = false;

 let filteredDomains: Array<DomainInList> = [];
 export let filteredProvider: Provider | null = null;
 let filteredGroup: string | null = null;
 let isGroupModalOpen = false;

 $: {
     if ($domains) {
         filteredDomains = $domains.filter(
             (d) => (!filteredProvider || d.id_provider === filteredProvider._id) &&
                  (filteredGroup === null || d.group === filteredGroup || ((filteredGroup === '' || filteredGroup === 'undefined') && (d.group === '' || d.group === undefined)))
         )
     }
 }
</script>

<Container class="flex-fill pt-4 pb-5">
    <h1 class="text-center mb-4">
        {$t('common.welcome.start')}<Logo height="40" />{$t('common.welcome.end')}
    </h1>

    <Row>
        <Col md="8" class="order-1 order-md-0">
            <ZoneList
                button
                display_by_groups
                domains={filteredDomains}
                links
            >
                <Badge slot="badges" color="success">
                    OK
                </Badge>
            </ZoneList>
            {#if filteredProvider}
                <CardImportableDomains
                    class={filteredDomains.length > 0 ? "mt-4":""}
                    provider={filteredProvider}
                    bind:noDomainsList={noDomainsList}
                />
            {/if}
            {#if !filteredProvider || noDomainsList}
                <!-- svelte-ignore a11y-autofocus -->
                <NewDomainInput
                    autofocus
                    class="mt-3"
                    provider={filteredProvider}
                />
            {/if}
        </Col>

        <Col md="4" class="order-0 order-md-1">
            <Card
                class="mb-3"
            >
                <div class="card-header d-flex justify-content-between">
                    {$t("provider.title")}
                    <Button
                        size="sm"
                        color="light"
                        href="/providers/new"
                    >
                        <Icon name="plus" />
                    </Button>
                </div>
                {#if !$providers || !$providersSpecs}
                    <div class="d-flex justify-content-center">
                        <Spinner color="primary" />
                    </div>
                {:else}
                    <ProviderList
                        flush
                        items={$providers}
                        noLabel
                        bind:selectedProvider={filteredProvider}
                        on:new-provider={() => goto('/providers/new')}
                    />
                {/if}
            </Card>

            {#if $domains && $domains.length}
                <Card
                    class="mb-3"
                >
                    <div class="card-header d-flex justify-content-between">
                        {$t("domaingroups.title")}
                        <Button
                            type="button"
                            size="sm"
                            color="light"
                            title={$t('domaingroups.manage')}
                            on:click={() => isGroupModalOpen = true}
                        >
                            <Icon name="grid-fill" />
                        </Button>
                    </div>
                    <DomainGroupList
                        flush
                        bind:selectedGroup={filteredGroup}
                    />
                    <DomainGroupModal bind:isOpen={isGroupModalOpen} />
                </Card>
            {/if}
        </Col>
    </Row>
</Container>

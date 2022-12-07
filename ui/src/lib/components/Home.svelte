<script lang="ts">
 import {
     Badge,
     CardHeader,
     Col,
     Container,
     Row,
 } from 'sveltestrap';

 import Logo from '$lib/components/Logo.svelte';
 import ZoneList from '$lib/components/ZoneList.svelte';
 import { t } from '$lib/translations';

 export let domains: Array<Domain>|undefined;

 if (domains === undefined) {

 }

 export let filteredDomains = [];
 export let filteredProvider = null;

 function showDomain(dn) {

 }
</script>

<Container class="pt-4 pb-5">
    <h1 class="text-center mb-4">
        {$t('common.welcome.start')}
        <Logo height="40" />
        {$t('common.welcome.end')}
    </h1>

    <Row>
        <Col md="8" class="order-1 order-md-0">
            <ZoneList
                button
                display-by-groups
                domains={filteredDomains}
                on:click={showDomain}
            >
                <Badge slot="badges" color="success">
                    OK
                </Badge>
            </ZoneList>
            {#if filteredProvider}
                <div class="card" class:mt-4={filteredDomains.length > 0}>
                    {#if !noDomainsList}
                        <CardHeader class="d-flex justify-content-between">
                            {$t("provider.provider")}
                            <em>{filteredProvider._comment}</em>
                            <Button
                                type="button"
                                color="secondary"
                                size="sm"
                            >
                                {$t('provider.import-domains')}
                            </Button>
                        </CardHeader>
                    {/if}
                    <h-provider-list-domains
                        ref="newDomains"
                        provider={filteredProvider}
                        show-domains-with-actions
                        on:no-domains-list-change={noDomainsList = $event}
                    />
                </div>
            {/if}
            {#if !filteredProvider || noDomainsList}
                <h-list-group-input-new-domain
                    autofocus
                    class="mt-2"
                    my-provider={filteredProvider}
                />
            {/if}
        </Col>
    </Row>
</Container>

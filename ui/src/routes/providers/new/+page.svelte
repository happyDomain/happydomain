<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Col,
     Container,
     Icon,
     Row,
 } from 'sveltestrap';

 import ProviderSelector from '$lib/components/providers/Selector.svelte';
 import { t } from '$lib/translations';

 function selectNewProvider(event: CustomEvent<{ptype: string}>) {
     goto('new/' + encodeURIComponent(event.detail.ptype));
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
    <Row class="my-3 flex-grow-1">
        <Col md={{size: 8, offset: 2}}>
            <ProviderSelector on:provider-selected={selectNewProvider} />
        </Col>
    </Row>
</Container>

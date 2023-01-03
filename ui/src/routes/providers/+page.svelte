<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Container,
     Col,
     Icon,
     Row,
     Spinner,
 } from 'sveltestrap';

 import ProviderList from '$lib/components/providers/List.svelte';
 import { providers, refreshProviders } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 refreshProviders();
</script>

<Container class="flex-fill pt-4 pb-5">
    <Button
        type="button"
        color="primary"
        class="float-end"
        on:click={() => goto('providers/new')}
    >
        <Icon name="plus" />
        {$t('common.add-new-thing', { thing: $t('provider.kind') })}
    </Button>
    <h1 class="text-center mb-4">
        {$t('provider.title')}
    </h1>
    {#if !$providers}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <Row>
            <Col md={{size: 8, offset: 2}}>
                <ProviderList
                    items={$providers}
                    on:new-provider={() => goto('providers/new')}
                    on:click={(event) => goto('providers/' + encodeURIComponent(event.detail._id))}
                />
            </Col>
        </Row>
    {/if}
</Container>

<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Container,
     Col,
     Row,
 } from 'sveltestrap';

 import ResolverForm from '$lib/components/resolver/Form.svelte';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';

 export let data = { };
 let request_pending = false;

 function resolveDomain(event) {
     const form = event.detail.value;
     const showDNSSEC = event.detail.showDNSSEC;

     request_pending = true;
     goto('/resolver/' + encodeURIComponent(form.domain), {
         state: {form, showDNSSEC},
     });
 }
</script>

<Container fluid class="d-flex flex-column">
    <Row class="flex-grow-1">
        <Col md={{offset: 2, size: 8}} class="pt-4 pb-5">
            <h1 class="text-center mb-3">
                {$t('menu.dns-resolver')}
            </h1>
            <ResolverForm
                bind:request_pending={request_pending}
                on:submit={resolveDomain}
            />
        </Col>
    </Row>
</Container>

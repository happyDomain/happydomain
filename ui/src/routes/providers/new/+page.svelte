<script lang="ts">
 import {
     Col,
     Row,
 } from 'sveltestrap';

 import ProviderSelector from '$lib/components/providers/Selector.svelte';

 function selectNewProvider() {

 }
</script>

<Row class="my-3">
    <Col offset-md="2" md="8">
        <ProviderSelector on:provider-selected={selectNewProvider} />
    </Col>
</Row>

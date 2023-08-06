<script lang="ts">
 import { page } from '$app/stores';

 import {
     Col,
     Container,
     Row,
 } from 'sveltestrap';
</script>

<Container class="my-3">
    <Row>
        <Col xs="4" class="d-flex flex-column">
            <img src="/img/cat-error.webp" alt="The error cat" class="rounded rounded-lg">
        </Col>
        <Col xs="8" class="d-flex flex-column pt-2">
            <div class="d-none d-sm-block my-4"></div>
            <div class="d-flex align-items-end">
                <h1>
                    Error {$page.status}
                </h1>
                <hr class="ms-2 border-top border-primary border-3 flex-fill">
            </div>
            {#if $page.error}
                <p class="lead">
                    {$page.error.message}
                </p>
            {/if}
            <hr class="mt-0 border-top border-secondary border-3">
        </Col>
    </Row>
</Container>

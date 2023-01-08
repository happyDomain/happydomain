<script lang="ts">
 import { page } from '$app/stores';

 import {
     Container,
 } from 'sveltestrap';
</script>

<Container>
    <h1>
        Error {$page.status}
    </h1>
    {#if $page.error}
        <p class="lead">
            {$page.error.message}
        </p>
    {/if}
</Container>

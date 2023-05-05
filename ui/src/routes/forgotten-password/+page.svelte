<script lang="ts">
 import {
     Alert,
     Container,
 } from 'sveltestrap';

 import ForgottenPasswordForm from '$lib/components/ForgottenPasswordForm.svelte';
 import RecoverAccountForm from '$lib/components/RecoverAccountForm.svelte';

 let error = "";
 export let data;
</script>

<Container class="my-3">
    {#if error}
        <Alert color="danger">
            {error}
        </Alert>
    {:else if data.user && data.key}
        <RecoverAccountForm user={data.user} key={data.key} />
    {:else}
        <ForgottenPasswordForm />
    {/if}
</Container>

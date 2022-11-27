<script lang="ts">
 import {
     Alert,
     Container,
 } from 'sveltestrap';

 import ForgottenPasswordForm from '$lib/components/ForgottenPasswordForm.svelte';
 import RecoverAccountForm from '$lib/components/RecoverAccountForm.svelte';

 let error = "";
 export let data = { user: "", key: "" };
</script>

<Container class="my-3">
    {#if error}
        <Alert color="danger">
            {error}
        </Alert>
    {:else if !data.user}
        <ForgottenPasswordForm />
    {:else}
        <RecoverAccountForm {data} />
    {/if}
</Container>

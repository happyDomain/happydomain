<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Alert,
     Container,
     Spinner
 } from 'sveltestrap';

 import { validateEmail } from '$lib/api/user';
 import EmailConfirmationForm from '$lib/components/EmailConfirmationForm.svelte';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';

 let error = "";
 export let data;

 if (data.user || data.key) {
     if (!data.user || !data.key) {
         error = $t('email.instruction.bad-link');
     } else {
         error = "";

         validateEmail(data.user, data.key)
         .then(
             () => {
                 toasts.addToast({
                     title: $t('account.ready-login'),
                     message: $t('email.instruction.validated'),
                     timeout: 5000,
                     type: 'success',
                 });
                 goto('/login');
             },
             (err) => {
                 error = err;
             }
         )
     }
 }
</script>

<Container class="my-3">
    {#if error}
        <Alert color="danger">
            {error}
        </Alert>
    {:else if data.user}
        <div class="d-flex justify-content-center align-items-center">
            <Spinner color="primary" label={$t('common.spinning')} class="me-3" />
            {$t('wait.wait')}
        </div>
    {:else}
        <EmailConfirmationForm />
    {/if}
</Container>

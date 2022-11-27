<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Col,
     Input,
     Row,
     Spinner,
 } from 'sveltestrap';

 import { resendValidationEmail } from '$lib/api/user';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';

 export let email = "";
 let emailState: boolean|undefined;
 let formSent = false;

 let formElm: HTMLFormElement;

 function goResend() {
     const valid = formElm.checkValidity()
     emailState = valid

     if (valid) {
         formSent = true;
         resendValidationEmail(email)
         .then(
             () => {
                 formSent = false;
                 toasts.addToast({
                     title: $t('email.sent'),
                     message: $t('email.instruction.check-inbox'),
                     timeout: 5000,
                     type: 'success',
                 });
                 goto('/');
            },
             (error) => {
                 formSent = false;
                 toasts.addErrorToast({
                     title: $t('errors.registration'),
                     message: error,
                     timeout: 20000,
                 });
             }
         );
     }
 }
</script>

<form
    class="container my-1"
    bind:this={formElm}
    on:submit|preventDefault={goResend}
>
    <p>
        {$t('email.instruction.validate-address')}
    </p>
    <p>
        {$t('email.instruction.new-confirmation')}
    </p>
    <Row>
        <label for="email-input" class="col-md-4 col-form-label text-truncate text-md-right fw-bold">
            {$t('email.address')}
        </label>
        <Col md="6">
            <Input
                id="email-input"
                invalid={emailState}
                required
                autofocus
                type="email"
                placeholder="jPostel@isi.edu"
                autocomplete="username"
                bind:value={email}
            />
        </Col>
    </Row>
    <Row class="mt-3">
        <Button class="offset-sm-4 col-sm-4" type="submit" color="primary" disabled={formSent}>
            {#if formSent}
                <Spinner label={$t('common.spinning')} size="sm" />
            {/if}
            {$t('email.send-again')}
        </Button>
    </Row>
</form>

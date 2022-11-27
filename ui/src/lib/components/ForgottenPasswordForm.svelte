<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Col,
     Input,
     Row,
     Spinner,
 } from 'sveltestrap';

 import { t } from '$lib/translations';
 import { forgotAccountPassword } from '$lib/api/user';
 import { toasts } from '$lib/stores/toasts';

 let value = "";
 let emailState: boolean|undefined = undefined;
 let formSent = false;

 let formElm: HTMLFormElement;
 function goSendLink() {
     const valid = formElm.checkValidity();
     emailState = valid;

     if (valid) {
         formSent = true;

         forgotAccountPassword(value)
         .then(
             () => {
                 formSent = false;
                 toasts.addToast({
                     title: $t('email.sent-recovery'),
                     message: $t('email.instruction.check-inbox'),
                     timeout: 20000,
                     color: 'success',
                 })
                 goto('/login');
             },
             (error) => {
                 formSent = false
                 toasts.addErrorToast({
                     title: $t('errors.recovery'),
                     message: error,
                     timeout: 10000,
                 })
             }
         )
     }
 }
</script>

<form
    bind:this={formElm}
    on:submit|preventDefault={goSendLink}
>
    <p class="text-center">
        {$t('email.recover')}.
    </p>
    <Row>
        <label for="email-input" class="col-md-4 col-form-label text-truncate text-md-right fw-bold">
            {$t('email.address')}
        </label>
        <Col md="6">
            <Input
                id="email-input"
                required
                autofocus
                type="email"
                placeholder="jPostel@isi.edu"
                autocomplete="username"
                invalid={emailState}
                bind:value={value}
            />
        </Col>
    </Row>
    <Row class="mt-3">
        <Button
            class="offset-1 col-10 offset-sm-2 col-sm-8 offset-md-3 col-md-6 offset-lg-4 col-lg-4"
            type="submit"
            color="primary"
            disabled={formSent}
        >
            {#if formSent}
                <Spinner
                    label="Spinning"
                    size="sm"
                />
            {/if}
            {$t('email.send-recover')}
        </Button>
    </Row>
</form>

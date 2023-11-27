<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     FormGroup,
     Input,
     Label,
     Spinner,
 } from 'sveltestrap';

 import { t } from '$lib/translations';
 import { authUser, cleanUserSession } from '$lib/api/user';
 import type { LoginForm } from '$lib/model/user';
 import { providers } from '$lib/stores/providers';
 import { toasts } from '$lib/stores/toasts';
 import { refreshUserSession } from '$lib/stores/usersession';

 let loginForm: LoginForm = {
     email: "",
     password: "",
 };
 let emailState: boolean|undefined;
 let passwordState: boolean|undefined;
 let formSent = false;

 let formElm: HTMLFormElement;

 function testLogin() {
     const valid = formElm.checkValidity()

     if (valid) {
         formSent = true;
         emailState = undefined;
         passwordState = undefined;

         authUser(loginForm)
         .then(
             () => {
                 cleanUserSession();
                 providers.set(null);
                 formSent = false;
                 emailState = true;
                 passwordState = true;
                 refreshUserSession();
                 goto('/');
             },
             (error) => {
                 formSent = false;
                 emailState = false;
                 passwordState = false;
                 toasts.addErrorToast({
                     title: $t('errors.login'),
                     message: error,
                     timeout: 20000,
                 })
             }
         )
     }
 }
</script>

<form
    class="container my-1"
    bind:this={formElm}
    on:submit|preventDefault={testLogin}
>
    <FormGroup>
        <Label for="email-input">{$t('email.address')}</Label>
        <Input
            aria-describedby="emailHelpBlock"
            autocomplete="username"
            autofocus
            id="email-input"
            placeholder="pMockapetris@usc.edu"
            required
            type="email"
            invalid={emailState !== undefined && !emailState}
            valid={emailState}
            bind:value={loginForm.email}
            on:change={() => emailState = loginForm.email.indexOf('@') > 0}
        />
    </FormGroup>
    <FormGroup>
        <Label for="password-input">{$t('common.password')}</Label>
        <Input
            autocomplete="current-password"
            id="password-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordState !== undefined && !passwordState}
            valid={passwordState}
            bind:value={loginForm.password}
        />
    </FormGroup>
    <div class="d-flex justify-content-around">
        <Button
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
            {$t('common.go')}
        </Button>
        <Button
            href="/forgotten-password"
            outline
            color="dark"
        >
            {$t('password.forgotten')}
        </Button>
    </div>
</form>

<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     FormGroup,
     Icon,
     Input,
     Label,
     Spinner,
 } from 'sveltestrap';

 import { t, locale } from '$lib/translations';
 import { registerUser } from '$lib/api/user';
 import type { SignUpForm } from '$lib/model/user';
 import { checkWeakPassword, checkPasswordConfirmation } from '$lib/password';
 import { toasts } from '$lib/stores/toasts';


 let signupForm: SignUpForm = {
     email: "",
     password: "",
     wantReceiveUpdate: false,
     lang: "",
 };
 let passwordConfirmation: string = "";
 let emailState: boolean|undefined;
 let passwordState: boolean|undefined;
 let passwordConfirmState: boolean|undefined;
 let formSent = false;

 $: {
     if (passwordState == false) {
         passwordState = checkWeakPassword(signupForm.password);
     }
 }

 let formElm: HTMLFormElement;

 function goSignUp() {
     const valid = formElm.checkValidity()

     if (valid && emailState && passwordState && passwordConfirmState) {
         formSent = true;
         signupForm.lang = $locale
         registerUser(signupForm)
         .then(
             () => {
                 formSent = false;
                 toasts.addToast({
                     title: $t('account.signup.success'),
                     message: $t('email.instruction.check-inbox'),
                     type: 'success',
                     timeout: 5000,
                 });
                 goto('/login');
             },
             (error) => {
                 formSent = false;
                 toasts.addErrorToast({
                     title: $t('errors.registration'),
                     message: error,
                     timeout: 10000,
                 })
             }
         )
     }
 }
</script>

<form
    class="container my-1"
    bind:this={formElm}
    on:submit|preventDefault={goSignUp}
>
    <FormGroup>
        <Label for="email-input">{$t('email.address')}</Label>
        <Input
            aria-describedby="emailHelpBlock"
            autocomplete="username"
            autofocus
            feedback={!emailState?$t('errors.address-valid'):null}
            id="email-input"
            placeholder="jPostel@isi.edu"
            required
            type="email"
            invalid={emailState !== undefined && !emailState}
            valid={emailState}
            bind:value={signupForm.email}
            on:change={() => emailState = signupForm.email.indexOf('@') > 0}
        />
        <div id="emailHelpBlock" class="form-text">
            {$t('account.signup.address-why', {
                identify: $t('account.signup.identify'),
                'security-operations': $t('account.signup.security-operations'),
            })}
        </div>
    </FormGroup>
    <FormGroup>
        <Label for="password-input">{$t('common.password')}</Label>
        <Input
            autocomplete="new-password"
            feedback={!passwordState?$t('errors.password-weak'):null}
            id="password-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordState !== undefined && !passwordState}
            valid={passwordState}
            bind:value={signupForm.password}
            on:change={() => passwordState = checkWeakPassword(signupForm.password)}
        />
    </FormGroup>
    <FormGroup>
        <Label for="passwordconfirm-input">{$t('password.confirmation')}</Label>
        <Input
            feedback={!passwordConfirmState?$t('errors.password-match'):null}
            id="passwordconfirm-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordConfirmState !== undefined && !passwordConfirmState}
            valid={passwordConfirmState}
            bind:value={passwordConfirmation}
            on:change={() => passwordConfirmState = checkPasswordConfirmation(signupForm.password, passwordConfirmation)}
        />
    </FormGroup>
    <FormGroup>
        <Input
            id="signup-newsletter"
            type="checkbox"
            label={$t('account.signup.receive-update')}
            bind:value={signupForm.wantReceiveUpdate}
        />
    </FormGroup>
    <div class="d-flex justify-content-around">
        <Button type="submit" color="primary" disabled={formSent}>
            {#if formSent}
                <Spinner
                    label="Spinning"
                    size="sm"
                />
            {:else}
                <Icon name="person-plus" />
            {/if}
            {$t('account.signup.signup')}
        </Button>
        <Button href="/login" outline color="dark">
            {$t('account.signup.already')}
        </Button>
    </div>
</form>

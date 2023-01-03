<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Input,
     Spinner,
 } from 'sveltestrap';

 import { t } from '$lib/translations';
 import { checkWeakPassword, checkPasswordConfirmation } from '$lib/password';
 import { changeUserPassword } from '$lib/api/user';
 import { userSession } from '$lib/stores/usersession';
 import { toasts } from '$lib/stores/toasts';

 let form = {
     current: "",
     password: "",
     passwordconfirm: "",
 };
 let passwordState: boolean|undefined = undefined;
 let passwordConfirmState: boolean|undefined = undefined;
 let formSent = false;

 let formElm: HTMLFormElement;
 function sendChPassword() {
     passwordConfirmState = checkPasswordConfirmation(form.password, form.passwordconfirm);
     const valid = formElm.checkValidity() && passwordConfirmState === true;

     if (valid && $userSession != null) {
         formSent = true;

         changeUserPassword($userSession, form)
         .then(
             () => {
                 formSent = false;
                 toasts.addToast({
                     title: $t('password.changed'),
                     message: $t('password.success-change'),
                     timeout: 5000,
                     color: 'success',
                 });
                 goto('/login');
             },
             (error) => {
                 formSent = false
                 toasts.addErrorToast({
                     title: $t('errors.password-change'),
                     message: error,
                     timeout: 10000,
                 });
             }
         )
     }
 }
</script>

<form
    bind:this={formElm}
    on:submit|preventDefault={sendChPassword}
>
    <div class="mb-3">
        <label for="currentPassword-input">
            {$t('password.enter')}
        </label>
        <Input
            id="currentPassword-input"
            bind:value={form.current}
            type="password"
            required
            placeholder="xXxXxXxXxX"
            autocomplete="current-password"
        />
    </div>
    <div class="mb-3">
        <label for="password-input">
            {$t('password.enter-new')}
        </label>
        <Input
            autocomplete="new-password"
            feedback={!passwordState?$t('errors.password-weak'):null}
            id="password-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordState !== undefined && !passwordState}
            valid={passwordState}
            bind:value={form.password}
            on:change={() => passwordState = checkWeakPassword(form.password)}
        />
    </div>
    <div class="mb-3">
        <label for="passwordconfirm-input">
            {$t('password.confirm-new')}
        </label>
        <Input
            feedback={!passwordConfirmState?$t('errors.password-match'):null}
            id="passwordconfirm-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordConfirmState !== undefined && !passwordConfirmState}
            valid={passwordConfirmState}
            bind:value={form.passwordconfirm}
            on:change={() => passwordConfirmState = checkPasswordConfirmation(form.password, form.passwordconfirm)}
        />
    </div>
    <div class="d-flex justify-content-around">
        <Button
            type="submit"
            color="primary"
            disabled={formSent}
        >
            {#if formSent}
                <Spinner size="sm" class="me-2" />
            {/if}
            {$t('password.change')}
        </Button>
    </div>
</form>

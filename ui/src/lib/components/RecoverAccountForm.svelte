<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Col,
     Input,
     Row,
     Spinner,
 } from 'sveltestrap';

 import { recoverAccount } from '$lib/api/user';
 import { checkWeakPassword, checkPasswordConfirmation } from '$lib/password';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';

 export let user: string;
 export let key: string;
 let value = "";
 let passwordConfirmation = "";
 let passwordState: boolean|undefined;
 let passwordConfirmState: boolean|undefined;
 let formSent = false;

 $: {
     if (passwordState == false) {
         passwordState = checkWeakPassword(value);
     }
 }

 let formElm: HTMLFormElement;

 function goRecover() {
     const valid = formElm.checkValidity()

     if (valid && passwordState && passwordConfirmState) {
         formSent = true;
         recoverAccount(user, key, value)
         .then(
             () => {
                 formSent = false;
                 toasts.addToast({
                     title: $t('password.redefined'),
                     message: $t('password.success'),
                     type: 'success',
                     timeout: 5000,
                 });
                 goto('/login');
             },
             (error) => {
                 formSent = false;
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
    class="container my-1"
    on:submit|preventDefault={goRecover}
    bind:this={formElm}
>
    <p>
        {$t('password.fill')}
    </p>
    <Row>
        <label for="password-input" class="col-md-4 col-form-label text-truncate text-md-right fw-bold">
            {$t('password.new')}
        </label>
        <Col md="6">
            <Input
                autocomplete="new-password"
                feedback={!passwordState?$t('errors.password-weak'):null}
                id="password-input"
                placeholder="xXxXxXxXxX"
                required
                type="password"
                invalid={passwordState !== undefined && !passwordState}
                valid={passwordState}
                bind:value={value}
                on:change={() => passwordState = checkWeakPassword(value)}
            />
        </Col>
    </Row>
    <Row class="mt-2">
        <label for="passwordconfirm-input" class="col-md-4 col-form-label text-truncate text-md-right fw-bold">
            {$t('password.confirmation')}
        </label>
        <Col md="6">
            <Input
                feedback={!passwordConfirmState?$t('errors.password-match'):null}
                id="passwordconfirm-input"
                placeholder="xXxXxXxXxX"
                required
                type="password"
                invalid={passwordConfirmState !== undefined && !passwordConfirmState}
                valid={passwordConfirmState}
                bind:value={passwordConfirmation}
                on:change={() => passwordConfirmState = checkPasswordConfirmation(value, passwordConfirmation)}
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
                <Spinner label="Spinning" size="sm" />
            {/if}
            {$t('password.redefine')}
        </Button>
    </Row>
</form>

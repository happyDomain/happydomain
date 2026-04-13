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
    import { navigate } from "$lib/stores/config";

    import { Button, FormGroup, Input, Spinner } from "@sveltestrap/sveltestrap";

    import { recoverAccount } from "$lib/api/user";
    import { checkWeakPassword, checkPasswordConfirmation } from "$lib/password";
    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        user: string;
        key: string;
    }

    let { user, key }: Props = $props();
    let value = $state("");
    let passwordConfirmation = $state("");
    let passwordState: boolean | undefined = $derived(checkWeakPassword(value));
    let passwordConfirmState: boolean | undefined = $state();
    let formSent = $state(false);

    let formElm: HTMLFormElement | undefined = $state();

    function goRecover(e: SubmitEvent) {
        e.preventDefault();

        if (!formElm) return;

        const valid = formElm.checkValidity();

        if (valid && passwordState && passwordConfirmState) {
            formSent = true;
            recoverAccount(user, key, value).then(
                () => {
                    formSent = false;
                    toasts.addToast({
                        title: $t("password.redefined"),
                        message: $t("password.success"),
                        type: "success",
                        timeout: 5000,
                    });
                    navigate("/login");
                },
                (error) => {
                    formSent = false;
                    toasts.addErrorToast({
                        title: $t("errors.recovery"),
                        message: error,
                        timeout: 10000,
                    });
                },
            );
        }
    }
</script>

<form onsubmit={goRecover} bind:this={formElm}>
    <FormGroup floating label={$t("password.new")}>
        <Input
            autocomplete="new-password"
            feedback={!passwordState ? $t("errors.password-weak") : null}
            id="password-input"
            placeholder={$t("password.new")}
            required
            type="password"
            invalid={passwordState !== undefined && !passwordState}
            valid={passwordState}
            bind:value
            on:change={() => (passwordState = checkWeakPassword(value))}
        />
    </FormGroup>
    <FormGroup floating label={$t("password.confirmation")}>
        <Input
            feedback={!passwordConfirmState ? $t("errors.password-match") : null}
            id="passwordconfirm-input"
            placeholder={$t("password.confirmation")}
            required
            type="password"
            invalid={passwordConfirmState !== undefined && !passwordConfirmState}
            valid={passwordConfirmState}
            bind:value={passwordConfirmation}
            on:change={() =>
                (passwordConfirmState = checkPasswordConfirmation(value, passwordConfirmation))}
        />
    </FormGroup>
    <div class="d-grid mt-3">
        <Button
            type="submit"
            color="primary"
            disabled={formSent}
        >
            {#if formSent}
                <Spinner size="sm" class="me-1" />
            {/if}
            {$t("password.redefine")}
        </Button>
    </div>
</form>

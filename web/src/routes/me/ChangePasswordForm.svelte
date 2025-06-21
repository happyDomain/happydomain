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
    import { preventDefault } from 'svelte/legacy';

    import { goto } from "$app/navigation";

    import { Button, Input, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { checkWeakPassword, checkPasswordConfirmation } from "$lib/password";
    import { changeUserPassword } from "$lib/api/user";
    import { userSession } from "$lib/stores/usersession";
    import { toasts } from "$lib/stores/toasts";

    let form = $state({
        current: "",
        password: "",
        passwordconfirm: "",
    });
    let passwordState: boolean | undefined = $state(undefined);
    let passwordConfirmState: boolean | undefined = $state(undefined);
    let formSent = $state(false);

    let formElm: HTMLFormElement | undefined = $state(undefined);
    function sendChPassword() {
        if (!formElm) return;

        passwordConfirmState = checkPasswordConfirmation(form.password, form.passwordconfirm);
        const valid = formElm.checkValidity() && passwordConfirmState === true;

        if (valid) {
            formSent = true;

            changeUserPassword($userSession, form).then(
                () => {
                    formSent = false;
                    toasts.addToast({
                        title: $t("password.changed"),
                        message: $t("password.success-change"),
                        timeout: 5000,
                        color: "success",
                    });
                    goto("/login");
                },
                (error) => {
                    formSent = false;
                    toasts.addErrorToast({
                        title: $t("errors.password-change"),
                        message: error,
                        timeout: 10000,
                    });
                },
            );
        }
    }
</script>

<form bind:this={formElm} onsubmit={preventDefault(sendChPassword)}>
    <div class="mb-3">
        <label for="currentPassword-input">
            {$t("password.enter")}
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
            {$t("password.enter-new")}
        </label>
        <Input
            autocomplete="new-password"
            feedback={!passwordState ? $t("errors.password-weak") : null}
            id="password-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordState !== undefined && !passwordState}
            valid={passwordState}
            bind:value={form.password}
            on:change={() => (passwordState = checkWeakPassword(form.password))}
        />
    </div>
    <div class="mb-3">
        <label for="passwordconfirm-input">
            {$t("password.confirm-new")}
        </label>
        <Input
            feedback={!passwordConfirmState ? $t("errors.password-match") : null}
            id="passwordconfirm-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordConfirmState !== undefined && !passwordConfirmState}
            valid={passwordConfirmState}
            bind:value={form.passwordconfirm}
            on:change={() =>
                (passwordConfirmState = checkPasswordConfirmation(
                    form.password,
                    form.passwordconfirm,
                ))}
        />
    </div>
    <div class="d-flex justify-content-around">
        <Button type="submit" color="primary" disabled={formSent}>
            {#if formSent}
                <Spinner size="sm" class="me-2" />
            {/if}
            {$t("password.change")}
        </Button>
    </div>
</form>

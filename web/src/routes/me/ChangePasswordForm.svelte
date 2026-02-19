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
    let passwordState: boolean | undefined = $derived(checkWeakPassword(form.password));
    let passwordConfirmState: boolean | undefined = $derived(checkPasswordConfirmation(form.password, form.passwordconfirm));
    let formSent = $state(false);

    let formElm: HTMLFormElement | undefined = $state(undefined);
    function sendChPassword(e: SubmitEvent) {
        e.preventDefault();

        if (!formElm) return;

        const valid = formElm.checkValidity() && passwordState === true && passwordConfirmState === true;

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
                    navigate("/login");
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

<form class="list-group" bind:this={formElm} onsubmit={sendChPassword}>
    <div class="list-group-item">
        <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-md-center gap-3">
            <label for="currentPassword-input" class="flex-grow-1 cursor-pointer mb-0">
                <div class="h5 mb-1">{$t("password.current-title")}</div>
                <p class="mb-0 text-muted small">{$t("password.current-description")}</p>
            </label>
            <div class="password-input">
                <Input
                    id="currentPassword-input"
                    bind:value={form.current}
                    type="password"
                    required
                    placeholder="••••••••"
                    autocomplete="current-password"
                />
            </div>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-md-center gap-3">
            <label for="password-input" class="flex-grow-1 cursor-pointer mb-0">
                <div class="h5 mb-1">{$t("password.new-title")}</div>
                <p class="mb-0 text-muted small">{$t("password.new-description")}</p>
            </label>
            <div class="password-input">
                <Input
                    autocomplete="new-password"
                    feedback={!passwordState ? $t("errors.password-weak") : null}
                    id="password-input"
                    placeholder="••••••••"
                    required
                    type="password"
                    invalid={passwordState !== undefined && !passwordState}
                    valid={passwordState}
                    bind:value={form.password}
                />
            </div>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-md-center gap-3">
            <label for="passwordconfirm-input" class="flex-grow-1 cursor-pointer mb-0">
                <div class="h5 mb-1">{$t("password.confirm-title")}</div>
                <p class="mb-0 text-muted small">{$t("password.confirm-description")}</p>
            </label>
            <div class="password-input">
                <Input
                    feedback={!passwordConfirmState ? $t("errors.password-match") : null}
                    id="passwordconfirm-input"
                    placeholder="••••••••"
                    required
                    type="password"
                    invalid={passwordConfirmState !== undefined && !passwordConfirmState}
                    valid={passwordConfirmState}
                    bind:value={form.passwordconfirm}
                />
            </div>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex justify-content-center justify-content-md-end align-items-center">
            <Button type="submit" color="primary" disabled={formSent} class="password-input">
                {#if formSent}
                    <Spinner size="sm" class="me-2" />
                {/if}
                {$t("password.change")}
            </Button>
        </div>
    </div>
</form>

<style>
    .password-input {
        width: 100%;
    }

    @media (min-width: 768px) {
        .password-input {
            width: 250px;
        }
    }
</style>

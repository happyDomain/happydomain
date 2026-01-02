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
    import { goto } from "$app/navigation";
    import { page } from "$app/stores";

    import { Button, FormGroup, Input, Label, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { authUser, cleanUserSession } from "$lib/api/user";
    import type { LoginForm } from "$lib/model/user";
    import { appConfig } from "$lib/stores/config";
    import { providers } from "$lib/stores/providers";
    import { toasts } from "$lib/stores/toasts";
    import { refreshUserSession } from "$lib/stores/usersession";

    let loginForm: LoginForm = {
        email: "",
        password: "",
    };
    let emailState: boolean | undefined;
    let passwordState: boolean | undefined;
    let formSent = false;

    let formElm: HTMLFormElement;

    function testLogin() {
        const valid = formElm.checkValidity();

        if (valid) {
            formSent = true;
            emailState = undefined;
            passwordState = undefined;

            authUser(loginForm).then(
                () => {
                    cleanUserSession();
                    providers.set(null);
                    formSent = false;
                    emailState = true;
                    passwordState = true;
                    refreshUserSession();

                    const nextParam = $page.url.searchParams.get("next");
                    if (nextParam) {
                        goto(decodeURIComponent(nextParam));
                    } else {
                        goto("/");
                    }
                },
                (error) => {
                    formSent = false;
                    emailState = false;
                    passwordState = false;
                    toasts.addErrorToast({
                        title: $t("errors.login"),
                        message: error,
                        timeout: 20000,
                    });
                },
            );
        }
    }

    function handleForgottenPassword() {
        if ($appConfig.no_mail) {
            toasts.addErrorToast({
                title: $t("password.recovery-unavailable.title"),
                message: $t("password.recovery-unavailable.description"),
                timeout: 10000,
            });
        } else {
            goto("/forgotten-password");
        }
    }
</script>

<form class="container my-1" bind:this={formElm} on:submit|preventDefault={testLogin}>
    <FormGroup>
        <Label for="email-input">{$t("email.address")}</Label>
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
            on:change={() => (emailState = loginForm.email.indexOf("@") > 0)}
        />
    </FormGroup>
    <FormGroup>
        <Label for="password-input">{$t("common.password")}</Label>
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
    <div class="d-flex flex-column flex-lg-row gap-2 justify-content-around">
        <Button type="submit" color="primary" disabled={formSent}>
            {#if formSent}
                <Spinner size="sm" />
            {/if}
            {$t("common.go")}
        </Button>
        {#if $appConfig.oidc_configured}
            {#await fetch("/auth/has_oidc") then res}
                {#await res.json() then oidc}
                    <Button href="/auth/oidc" color="secondary">
                        {#if oidc.provider == "google.com"}
                            <i class="bi bi-google" />
                        {:else if oidc.provider == "gitlab.com" || oidc.provider == "framagit.org"}
                            <i class="bi bi-gitlab" />
                        {:else if oidc.provider == "github.com"}
                            <i class="bi bi-github" />
                        {:else if oidc.provider == "microsoft.com"}
                            <i class="bi bi-microsoft" />
                        {:else if oidc.provider == "apple.com"}
                            <i class="bi bi-apple" />
                        {/if}
                        {$t("account.oidc-login", { provider: oidc.provider })}
                    </Button>
                {/await}
            {/await}
        {/if}
        <Button type="button" on:click={handleForgottenPassword} outline color="dark">
            {$t("password.forgotten")}
        </Button>
    </div>
</form>

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
    import { Button, FormGroup, Icon, Input, Label, Spinner } from "@sveltestrap/sveltestrap";

    import { t, locale } from "$lib/translations";
    import { registerUser } from "$lib/api/user";
    import type { SignUpForm } from "$lib/model/user";
    import { checkWeakPassword, checkPasswordConfirmation } from "$lib/password";
    import { appConfig, navigate } from "$lib/stores/config";
    import { toasts } from "$lib/stores/toasts";
    import CaptchaWidget from "$lib/components/CaptchaWidget.svelte";

    let signupForm: SignUpForm = $state({
        email: "",
        password: "",
        wantReceiveUpdate: false,
        lang: "",
    });
    let passwordConfirmation: string = $state("");
    let emailState: boolean | undefined = $state();
    let passwordState: boolean | undefined = $derived(checkWeakPassword(signupForm.password));
    let passwordConfirmState: boolean | undefined = $state();
    let formSent = $state(false);
    let captchaToken: string | null = $state(null);
    let captchaWidget: ReturnType<typeof CaptchaWidget> | undefined = $state();

    let formElm: HTMLFormElement | undefined = $state();

    function goSignUp(e: SubmitEvent) {
        e.preventDefault();
        if (!formElm) return;

        const valid = formElm.checkValidity();

        if (valid && emailState && passwordState && passwordConfirmState) {
            formSent = true;
            signupForm.lang = $locale;
            const formWithCaptcha = captchaToken
                ? { ...signupForm, captcha_token: captchaToken }
                : signupForm;
            registerUser(formWithCaptcha).then(
                () => {
                    formSent = false;
                    toasts.addToast({
                        title: $t("account.signup.success"),
                        message: $appConfig.no_mail
                            ? $t("account.signup.login-now")
                            : $t("email.instruction.check-inbox"),
                        type: "success",
                        timeout: 5000,
                    });
                    navigate("/login");
                },
                (error) => {
                    formSent = false;
                    captchaToken = null;
                    if (captchaWidget) captchaWidget.reset();
                    toasts.addErrorToast({
                        title: $t("errors.registration"),
                        message: error,
                        timeout: 10000,
                    });
                },
            );
        }
    }
</script>

<form class="container my-1" bind:this={formElm} onsubmit={goSignUp}>
    <FormGroup>
        <Label for="email-input">{$t("email.address")}</Label>
        <Input
            aria-describedby="emailHelpBlock"
            autocomplete="username"
            autofocus
            feedback={!emailState ? $t("errors.address-valid") : null}
            id="email-input"
            placeholder="jPostel@isi.edu"
            required
            type="email"
            invalid={emailState !== undefined && !emailState}
            valid={emailState}
            bind:value={signupForm.email}
            on:change={() => (emailState = signupForm.email.indexOf("@") > 0)}
        />
        <div id="emailHelpBlock" class="form-text">
            {$t("account.signup.address-why", {
                identify: $t("account.signup.identify"),
                "security-operations": $t("account.signup.security-operations"),
            })}
        </div>
    </FormGroup>
    <FormGroup>
        <Label for="password-input">{$t("common.password")}</Label>
        <Input
            autocomplete="new-password"
            feedback={!passwordState ? $t("errors.password-weak") : null}
            id="password-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordState !== undefined && !passwordState}
            valid={passwordState}
            bind:value={signupForm.password}
            on:change={() => (passwordState = checkWeakPassword(signupForm.password))}
        />
    </FormGroup>
    <FormGroup>
        <Label for="passwordconfirm-input">{$t("password.confirmation")}</Label>
        <Input
            feedback={!passwordConfirmState ? $t("errors.password-match") : null}
            id="passwordconfirm-input"
            placeholder="xXxXxXxXxX"
            required
            type="password"
            invalid={passwordConfirmState !== undefined && !passwordConfirmState}
            valid={passwordConfirmState}
            bind:value={passwordConfirmation}
            on:change={() =>
                (passwordConfirmState = checkPasswordConfirmation(
                    signupForm.password,
                    passwordConfirmation,
                ))}
        />
    </FormGroup>
    <FormGroup>
        <Input
            id="signup-newsletter"
            type="checkbox"
            label={$t("account.signup.receive-update")}
            bind:checked={signupForm.wantReceiveUpdate}
        />
    </FormGroup>
    {#if $appConfig.captcha_provider}
        <p>{$t("captcha.human-check")}</p>
        <CaptchaWidget bind:this={captchaWidget} bind:token={captchaToken} />
    {/if}
    <div class="d-flex justify-content-around gap-2">
        <Button type="submit" color="primary" disabled={formSent}>
            {#if formSent}
                <Spinner size="sm" />
            {:else}
                <Icon name="person-plus" />
            {/if}
            {$t("account.signup.signup")}
        </Button>
        <Button href="/login" outline color="dark">
            {$t("account.signup.already")}
        </Button>
    </div>
</form>

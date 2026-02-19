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

    import { Button, Col, Input, Row, Spinner } from "@sveltestrap/sveltestrap";

    import { resendValidationEmail } from "$lib/api/user";
    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        email?: string;
    }

    let { email = $bindable("") }: Props = $props();

    let emailState: boolean | undefined = $state();
    let formSent = $state(false);
    let formElm: HTMLFormElement | undefined = $state();

    function goResend(e: SubmitEvent) {
        e.preventDefault();

        if (!formElm) return;

        const valid = formElm.checkValidity();
        emailState = valid;

        if (valid) {
            formSent = true;
            resendValidationEmail(email).then(
                () => {
                    formSent = false;
                    toasts.addToast({
                        title: $t("email.sent"),
                        message: $t("email.instruction.check-inbox"),
                        timeout: 5000,
                        type: "success",
                    });
                    navigate("/");
                },
                (error) => {
                    formSent = false;
                    toasts.addErrorToast({
                        title: $t("errors.registration"),
                        message: error,
                        timeout: 20000,
                    });
                },
            );
        }
    }
</script>

<form class="container my-1" bind:this={formElm} onsubmit={goResend}>
    <p>
        {$t("email.instruction.validate-address")}
    </p>
    <p>
        {$t("email.instruction.new-confirmation")}
    </p>
    <Row>
        <label
            for="email-input"
            class="col-md-4 col-form-label text-truncate text-md-right fw-bold"
        >
            {$t("email.address")}
        </label>
        <Col md="6">
            <Input
                id="email-input"
                autocomplete="username"
                autofocus
                invalid={emailState}
                placeholder="jPostel@isi.edu"
                required
                type="email"
                bind:value={email}
            />
        </Col>
    </Row>
    <Row class="mt-3">
        <Button class="offset-sm-4 col-sm-4" color="primary" disabled={formSent} type="submit">
            {#if formSent}
                <Spinner size="sm" />
            {/if}
            {$t("email.send-again")}
        </Button>
    </Row>
</form>

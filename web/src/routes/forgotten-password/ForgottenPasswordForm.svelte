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

    import { Button, Col, Input, Row, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { forgotAccountPassword } from "$lib/api/user";
    import { toasts } from "$lib/stores/toasts";

    let value = "";
    let emailState: boolean | undefined = undefined;
    let formSent = false;

    let formElm: HTMLFormElement;
    function goSendLink() {
        const valid = formElm.checkValidity();
        emailState = valid;

        if (valid) {
            formSent = true;

            forgotAccountPassword(value).then(
                () => {
                    formSent = false;
                    toasts.addToast({
                        title: $t("email.sent-recovery"),
                        message: $t("email.instruction.check-inbox"),
                        timeout: 20000,
                        color: "success",
                    });
                    goto("/login");
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

<form bind:this={formElm} on:submit|preventDefault={goSendLink}>
    <p class="text-center">
        {$t("email.recover")}.
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
                required
                autofocus
                type="email"
                placeholder="jPostel@isi.edu"
                autocomplete="username"
                invalid={emailState}
                bind:value
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
            {$t("email.send-recover")}
        </Button>
    </Row>
</form>

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

    import { Alert, Col, Container, Row, Spinner } from "@sveltestrap/sveltestrap";

    import { validateEmail } from "$lib/api/user";
    import { appConfig } from "$lib/stores/config";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import EmailConfirmationForm from "./EmailConfirmationForm.svelte";

    let error = $state("");
    let { data } = $props();

    $effect(() => {
        if (data.user || data.key) {
            if (!data.user || !data.key) {
                error = $t("email.instruction.bad-link");
            } else {
                error = "";

                validateEmail(data.user, data.key).then(
                    () => {
                        toasts.addToast({
                            title: $t("account.ready-login"),
                            message: $t("email.instruction.validated"),
                            timeout: 5000,
                            type: "success",
                        });
                        goto("/login");
                    },
                    (err) => {
                        error = err;
                    },
                );
            }
        }
    });
</script>

<svelte:head>
    <title>happyDomain</title>
</svelte:head>

<Container class="my-3">
    {#if $appConfig.no_mail}
        <Row>
            <Col
                md={{ offset: 1, size: 10 }}
                lg={{ offset: 2, size: 8 }}
                xl={{ offset: 3, size: 6 }}
            >
                <Alert color="warning">
                    <h4 class="alert-heading">{$t("email.validation-unavailable.title")}</h4>
                    <p>
                        {$t("email.validation-unavailable.description")}
                    </p>
                </Alert>
            </Col>
        </Row>
    {:else if error}
        <Alert color="danger">
            {error}
        </Alert>
    {:else if data.user}
        <div class="d-flex justify-content-center align-items-center">
            <Spinner color="primary" class="me-3" />
            {$t("wait.wait")}
        </div>
    {:else}
        <EmailConfirmationForm />
    {/if}
</Container>

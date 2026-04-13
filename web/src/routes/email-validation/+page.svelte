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
    import { Alert, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import Logo from "$lib/components/Logo.svelte";
    import { validateEmail } from "$lib/api/user";
    import { appConfig, navigate } from "$lib/stores/config";
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
                        navigate("/login");
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

<Container class="my-auto py-4">
    <Row class="justify-content-center align-items-stretch g-0">
        <Col md="5" lg="4" class="d-none d-md-block">
            <div class="validation-visual">
                <div class="text-center">
                    <Icon name="envelope-check" class="validation-icon" />
                </div>
            </div>
        </Col>
        <Col sm="10" md="7" lg="5">
            <div class="validation-card">
                <div class="text-center mb-4 d-md-none">
                    <Logo height="32" />
                </div>
                {#if $appConfig.no_mail}
                    <Alert color="warning" class="mb-0">
                        <h6 class="alert-heading fw-bold">
                            {$t("email.validation-unavailable.title")}
                        </h6>
                        <p class="mb-0 small">
                            {$t("email.validation-unavailable.description")}
                        </p>
                    </Alert>
                {:else if error}
                    <h5 class="fw-bold mb-3">{$t("email.address")}</h5>
                    <Alert color="danger" class="mb-0">
                        {error}
                    </Alert>
                {:else if data.user}
                    <div class="d-flex flex-column align-items-center py-4">
                        <Spinner color="primary" class="mb-3" />
                        <p class="text-body-secondary mb-0">{$t("wait.wait")}</p>
                    </div>
                {:else}
                    <h5 class="fw-bold mb-1">{$t("email.address")}</h5>
                    <p class="text-body-secondary small mb-2">
                        {$t("email.instruction.validate-address")}
                    </p>
                    <p class="text-body-secondary small mb-4">
                        {$t("email.instruction.new-confirmation")}
                    </p>
                    <EmailConfirmationForm />
                {/if}
                <div class="text-center mt-4 pt-3 border-top">
                    <a href="/login" class="text-body-secondary text-decoration-none small">
                        <Icon name="arrow-left" class="me-1" />
                        {$t("common.go-back")}
                    </a>
                </div>
            </div>
        </Col>
    </Row>
</Container>

<style>
    .validation-visual {
        height: 100%;
        border-radius: 1rem 0 0 1rem;
        overflow: hidden;
        background: linear-gradient(135deg, #edf7f4 0%, #e0f0ed 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 2rem;
    }

    .validation-visual :global(.validation-icon) {
        font-size: 5rem;
        color: #1cb487;
        opacity: 0.6;
    }

    .validation-card {
        background: #fff;
        border-radius: 0 1rem 1rem 0;
        padding: 2.5rem 2rem;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
        height: 100%;
        display: flex;
        flex-direction: column;
        justify-content: center;
    }

    @media (max-width: 767.98px) {
        .validation-card {
            border-radius: 1rem;
        }
    }
</style>
